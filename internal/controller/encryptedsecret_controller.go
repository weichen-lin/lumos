/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/getsops/sops/v3/decrypt"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
	"github.com/weichen-lin/lumos/internal/provider"
)

// sopsDecryptMu serialises SOPS decrypt calls so that the SOPS_AGE_KEY env var
// injection is race-free across concurrent reconciler goroutines.
var sopsDecryptMu sync.Mutex

// EncryptedSecretReconciler reconciles an EncryptedSecret object.
type EncryptedSecretReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=sync.lumos.io,resources=encryptedsecrets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=sync.lumos.io,resources=encryptedsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sync.lumos.io,resources=configstores,verbs=get;list;watch
// +kubebuilder:rbac:groups=sync.lumos.io,resources=clusterconfigstores,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *EncryptedSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch EncryptedSecret.
	var es syncv1alpha1.EncryptedSecret
	if err := r.Get(ctx, req.NamespacedName, &es); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Resolve the referenced ConfigStore.
	storeSpec, err := r.resolveESStoreSpec(ctx, &es)
	if err != nil {
		r.Recorder.Event(&es, corev1.EventTypeWarning, "StoreNotFound", err.Error())
		return r.esSetFailed(ctx, &es, "StoreNotFound", err.Error())
	}

	// 3. Build the Git provider.
	gitProvider, err := r.buildGitProvider(ctx, &es, storeSpec)
	if err != nil {
		r.Recorder.Event(&es, corev1.EventTypeWarning, "ProviderError", err.Error())
		return r.esSetFailed(ctx, &es, "ProviderError", err.Error())
	}

	// 4. Collect the list of source files to fetch.
	sources := make([]string, len(es.Spec.Data))
	for i, d := range es.Spec.Data {
		sources[i] = d.Source
	}

	// 5. Clone the repo and read raw encrypted bytes.
	rawFiles, commitSHA, err := gitProvider.FetchRaw(ctx, sources)
	if err != nil {
		log.Error(err, "failed to fetch encrypted files from git")
		r.Recorder.Event(&es, corev1.EventTypeWarning, "FetchFailed", err.Error())
		if statusErr := r.esMarkFailed(ctx, &es, "FetchFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "failed to update status after fetch failure")
		}
		return ctrl.Result{RequeueAfter: es.Spec.RefreshInterval.Duration}, nil
	}

	// 6. Read the age private key from the referenced K8s Secret.
	ageKey, err := r.resolveAgeKey(ctx, &es)
	if err != nil {
		r.Recorder.Event(&es, corev1.EventTypeWarning, "AgeKeyError", err.Error())
		return r.esSetFailed(ctx, &es, "AgeKeyError", err.Error())
	}

	// 7. Decrypt each file and merge all top-level keys into one map.
	secretData := make(map[string][]byte)
	for src, raw := range rawFiles {
		decrypted, decryptErr := decryptSOPS(raw, ageKey, src)
		if decryptErr != nil {
			msg := fmt.Sprintf("decrypting %q: %v", src, decryptErr)
			log.Error(decryptErr, "sops decryption failed", "source", src)
			r.Recorder.Event(&es, corev1.EventTypeWarning, "DecryptFailed", msg)
			if statusErr := r.esMarkFailed(ctx, &es, "DecryptFailed", msg); statusErr != nil {
				log.Error(statusErr, "failed to update status after decrypt failure")
			}
			return ctrl.Result{RequeueAfter: es.Spec.RefreshInterval.Duration}, nil
		}
		if err := mergeSecretData(decrypted, secretData); err != nil {
			r.Recorder.Event(&es, corev1.EventTypeWarning, "ParseFailed", err.Error())
			return r.esSetFailed(ctx, &es, "ParseFailed", err.Error())
		}
	}

	// 8. Create or update the target K8s Secret.
	targetName := es.Name
	if es.Spec.Target != nil && es.Spec.Target.Name != "" {
		targetName = es.Spec.Target.Name
	}
	if err := r.syncSecret(ctx, &es, targetName, secretData); err != nil {
		r.Recorder.Event(&es, corev1.EventTypeWarning, "SyncFailed", err.Error())
		return r.esSetFailed(ctx, &es, "SyncFailed", err.Error())
	}

	// 9. Update status.
	now := metav1.Now()
	es.Status.SyncedAt = &now
	es.Status.ObservedVersion = commitSHA

	apimeta.SetStatusCondition(&es.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            fmt.Sprintf("Decrypted %d key(s) from %d file(s)", len(secretData), len(sources)),
		ObservedGeneration: es.Generation,
	})
	if err := r.Status().Update(ctx, &es); err != nil {
		return ctrl.Result{}, err
	}

	msg := fmt.Sprintf("Synced %d key(s) to Secret %s", len(secretData), targetName)
	r.Recorder.Event(&es, corev1.EventTypeNormal, "Synced", msg)
	log.Info("sync successful", "secret", targetName, "keys", len(secretData), "version", commitSHA)

	// 10. Re-queue after the configured refresh interval.
	return ctrl.Result{RequeueAfter: es.Spec.RefreshInterval.Duration}, nil
}

// syncSecret creates or updates the target K8s Secret with the decrypted data.
func (r *EncryptedSecretReconciler) syncSecret(
	ctx context.Context,
	es *syncv1alpha1.EncryptedSecret,
	name string,
	data map[string][]byte,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: es.Namespace,
		},
	}
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Data = data
		return ctrl.SetControllerReference(es, secret, r.Scheme)
	})
	return err
}

// resolveESStoreSpec resolves the ConfigStore or ClusterConfigStore referenced by
// the EncryptedSecret. Only the Git provider is supported.
func (r *EncryptedSecretReconciler) resolveESStoreSpec(
	ctx context.Context,
	es *syncv1alpha1.EncryptedSecret,
) (*syncv1alpha1.ConfigStoreSpec, error) {
	ref := es.Spec.StoreRef
	switch ref.Kind {
	case "", "ConfigStore":
		var store syncv1alpha1.ConfigStore
		if err := r.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: es.Namespace,
		}, &store); err != nil {
			return nil, fmt.Errorf("ConfigStore %q not found: %w", ref.Name, err)
		}
		return &store.Spec, nil
	case "ClusterConfigStore":
		var store syncv1alpha1.ClusterConfigStore
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name}, &store); err != nil {
			return nil, fmt.Errorf("ClusterConfigStore %q not found: %w", ref.Name, err)
		}
		return &store.Spec, nil
	default:
		return nil, fmt.Errorf("unknown store kind %q", ref.Kind)
	}
}

// buildGitProvider builds a GitProvider from the store spec.
// Returns an error if the store provider is not Git.
func (r *EncryptedSecretReconciler) buildGitProvider(
	ctx context.Context,
	es *syncv1alpha1.EncryptedSecret,
	store *syncv1alpha1.ConfigStoreSpec,
) (*provider.GitProvider, error) {
	if store.Provider != syncv1alpha1.ProviderGit {
		return nil, fmt.Errorf("EncryptedSecret requires a Git-backed store, got %q", store.Provider)
	}
	if store.Git == nil {
		return nil, fmt.Errorf("store %q has provider Git but no git config", es.Spec.StoreRef.Name)
	}
	auth, err := resolveGitAuthFromClient(ctx, r.Client, es.Namespace, store.Git)
	if err != nil {
		return nil, err
	}
	return provider.NewGit(store.Git.URL, store.Git.Branch, auth), nil
}

// resolveAgeKey reads the age private key from the K8s Secret referenced by AgeKeyRef.
// It looks for the key under "keys.txt".
func (r *EncryptedSecretReconciler) resolveAgeKey(
	ctx context.Context,
	es *syncv1alpha1.EncryptedSecret,
) (string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      es.Spec.AgeKeyRef.Name,
		Namespace: es.Namespace,
	}, &secret); err != nil {
		return "", fmt.Errorf("reading age key secret %q: %w", es.Spec.AgeKeyRef.Name, err)
	}
	raw, ok := secret.Data["keys.txt"]
	if !ok {
		return "", fmt.Errorf("secret %q has no key \"keys.txt\"", es.Spec.AgeKeyRef.Name)
	}
	return string(raw), nil
}

// decryptSOPS decrypts SOPS-encrypted bytes using the given age private key.
// The format is inferred from the source file extension (yaml/json/env).
// A package-level mutex ensures env var injection is safe across goroutines.
func decryptSOPS(data []byte, agePrivateKey, source string) ([]byte, error) {
	format := sopsFormat(source)

	sopsDecryptMu.Lock()
	defer sopsDecryptMu.Unlock()

	prev := os.Getenv("SOPS_AGE_KEY")
	if err := os.Setenv("SOPS_AGE_KEY", agePrivateKey); err != nil {
		return nil, fmt.Errorf("setting SOPS_AGE_KEY: %w", err)
	}
	defer os.Setenv("SOPS_AGE_KEY", prev) //nolint:errcheck

	return decrypt.Data(data, format)
}

// sopsFormat returns the SOPS format string for a given file path.
func sopsFormat(source string) string {
	switch strings.ToLower(filepath.Ext(source)) {
	case ".json":
		return "json"
	case ".env":
		return "dotenv"
	case ".ini":
		return "ini"
	default:
		return "yaml" // covers .yaml, .yml, and unknown extensions
	}
}

// mergeSecretData parses decrypted YAML/JSON bytes and merges all top-level
// string keys into the destination map as []byte values.
func mergeSecretData(decrypted []byte, dest map[string][]byte) error {
	var m map[string]interface{}
	if err := yaml.Unmarshal(decrypted, &m); err != nil {
		return fmt.Errorf("parsing decrypted content: %w", err)
	}
	if m == nil {
		return fmt.Errorf("decrypted content is not a YAML map")
	}
	for k, v := range m {
		dest[k] = []byte(fmt.Sprintf("%v", v))
	}
	return nil
}

// esSetFailed marks the EncryptedSecret as not-Ready and requeues.
func (r *EncryptedSecretReconciler) esSetFailed(
	ctx context.Context,
	es *syncv1alpha1.EncryptedSecret,
	reason, message string,
) (ctrl.Result, error) {
	if err := r.esMarkFailed(ctx, es, reason, message); err != nil {
		logf.FromContext(ctx).Error(err, "failed to update status condition")
	}
	return ctrl.Result{RequeueAfter: es.Spec.RefreshInterval.Duration}, nil
}

// esMarkFailed updates the Ready condition to False.
func (r *EncryptedSecretReconciler) esMarkFailed(
	ctx context.Context,
	es *syncv1alpha1.EncryptedSecret,
	reason, message string,
) error {
	logf.FromContext(ctx).Error(fmt.Errorf("%s", reason), message) //nolint:goerr113
	apimeta.SetStatusCondition(&es.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: es.Generation,
	})
	return r.Status().Update(ctx, es)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EncryptedSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&syncv1alpha1.EncryptedSecret{},
		".spec.storeRef.name",
		func(obj client.Object) []string {
			es := obj.(*syncv1alpha1.EncryptedSecret)
			return []string{es.Spec.StoreRef.Name}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1alpha1.EncryptedSecret{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		// Re-create the Secret if someone deletes it manually.
		Owns(&corev1.Secret{},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		// Re-reconcile when the referenced ConfigStore changes.
		Watches(&syncv1alpha1.ClusterConfigStore{}, handler.EnqueueRequestsFromMapFunc(
			r.encryptedSecretsForClusterStore,
		)).
		Watches(&syncv1alpha1.ConfigStore{}, handler.EnqueueRequestsFromMapFunc(
			r.encryptedSecretsForStore,
		)).
		Named("encryptedsecret").
		Complete(r)
}

func (r *EncryptedSecretReconciler) encryptedSecretsForClusterStore(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	var esList syncv1alpha1.EncryptedSecretList
	if err := r.List(ctx, &esList, client.MatchingFields{".spec.storeRef.name": obj.GetName()}); err != nil {
		logf.FromContext(ctx).Error(err, "failed to list EncryptedSecrets for ClusterConfigStore", "store", obj.GetName())
		return nil
	}
	var reqs []reconcile.Request
	for _, es := range esList.Items {
		if es.Spec.StoreRef.Kind == "ClusterConfigStore" {
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: es.Name, Namespace: es.Namespace},
			})
		}
	}
	return reqs
}

func (r *EncryptedSecretReconciler) encryptedSecretsForStore(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	var esList syncv1alpha1.EncryptedSecretList
	if err := r.List(ctx, &esList,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{".spec.storeRef.name": obj.GetName()},
	); err != nil {
		logf.FromContext(ctx).Error(err, "failed to list EncryptedSecrets for ConfigStore", "store", obj.GetName())
		return nil
	}
	var reqs []reconcile.Request
	for _, es := range esList.Items {
		if es.Spec.StoreRef.Kind == "" || es.Spec.StoreRef.Kind == "ConfigStore" {
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: es.Name, Namespace: es.Namespace},
			})
		}
	}
	return reqs
}
