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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
	"github.com/weichen-lin/lumos/internal/provider"
)

// ExternalConfigReconciler reconciles a ExternalConfig object.
type ExternalConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=sync.lumos.io,resources=externalconfigs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=sync.lumos.io,resources=externalconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sync.lumos.io,resources=configstores,verbs=get;list;watch
// +kubebuilder:rbac:groups=sync.lumos.io,resources=clusterconfigstores,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ExternalConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch ExternalConfig. If deleted, nothing to do.
	var ec syncv1alpha1.ExternalConfig
	if err := r.Get(ctx, req.NamespacedName, &ec); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Resolve the referenced ConfigStore or ClusterConfigStore.
	storeSpec, err := r.resolveStoreSpec(ctx, &ec)
	if err != nil {
		r.recordEvent(&ec, corev1.EventTypeWarning, "StoreNotFound", err.Error())
		return r.setFailed(ctx, &ec, "StoreNotFound", err.Error())
	}

	// 3. Build the provider from the store configuration.
	p, err := r.buildProvider(ctx, &ec, storeSpec)
	if err != nil {
		r.recordEvent(&ec, corev1.EventTypeWarning, "ProviderError", err.Error())
		return r.setFailed(ctx, &ec, "ProviderError", err.Error())
	}

	// 4. Fetch config data from the remote source.
	fetchResult, err := p.Fetch(ctx, ec.Spec.Data)
	if err != nil {
		log.Error(err, "failed to fetch config from provider")
		r.recordEvent(&ec, corev1.EventTypeWarning, "FetchFailed", err.Error())
		if statusErr := r.markFailed(ctx, &ec, "FetchFailed", err.Error()); statusErr != nil {
			log.Error(statusErr, "failed to update status after fetch failure")
		}
		return ctrl.Result{RequeueAfter: ec.Spec.RefreshInterval.Duration}, nil
	}

	// 5. Create or update the target ConfigMap.
	cmName := ec.Name
	if ec.Spec.Target != nil && ec.Spec.Target.Name != "" {
		cmName = ec.Spec.Target.Name
	}
	if err := r.syncConfigMap(ctx, &ec, cmName, fetchResult.Data); err != nil {
		r.recordEvent(&ec, corev1.EventTypeWarning, "SyncFailed", err.Error())
		return r.setFailed(ctx, &ec, "SyncFailed", err.Error())
	}

	// 6. Update ExternalConfig status to reflect the successful sync.
	now := metav1.Now()
	ec.Status.SyncedAt = &now
	ec.Status.ObservedVersion = fetchResult.Version

	// Store key mappings so the API can show source → keys relationships.
	ec.Status.KeyMappings = make([]syncv1alpha1.KeyMapping, 0, len(fetchResult.Mappings))
	for _, m := range fetchResult.Mappings {
		ec.Status.KeyMappings = append(ec.Status.KeyMappings, syncv1alpha1.KeyMapping{
			Source: m.Source,
			Keys:   m.Keys,
		})
	}

	apimeta.SetStatusCondition(&ec.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            fmt.Sprintf("Synced %d key(s) from %s", len(fetchResult.Data), storeSpec.Provider),
		ObservedGeneration: ec.Generation,
	})
	if err := r.Status().Update(ctx, &ec); err != nil {
		return ctrl.Result{}, err
	}

	msg := fmt.Sprintf("Synced %d key(s) to ConfigMap %s", len(fetchResult.Data), cmName)
	r.recordEvent(&ec, corev1.EventTypeNormal, "Synced", msg)

	log.Info("sync successful",
		"configmap", cmName,
		"keys", len(fetchResult.Data),
		"version", fetchResult.Version,
	)

	// 7. Re-queue after the configured refresh interval.
	return ctrl.Result{RequeueAfter: ec.Spec.RefreshInterval.Duration}, nil
}

// syncConfigMap creates or updates the target ConfigMap with the fetched data.
// It sets the ExternalConfig as the owner so the ConfigMap is garbage-collected
// when the ExternalConfig is deleted.
func (r *ExternalConfigReconciler) syncConfigMap(
	ctx context.Context,
	ec *syncv1alpha1.ExternalConfig,
	name string,
	data map[string]string,
) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ec.Namespace,
		},
	}
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = data
		// Owner reference: deleting ExternalConfig will also delete the ConfigMap.
		return ctrl.SetControllerReference(ec, cm, r.Scheme)
	})
	return err
}

func (r *ExternalConfigReconciler) recordEvent(ec *syncv1alpha1.ExternalConfig, eventType, reason, message string) {
	r.Recorder.Eventf(ec, nil, eventType, reason, reason, message)
}

// resolveStoreSpec looks up the ConfigStore or ClusterConfigStore referenced by
// the ExternalConfig and returns its spec. ClusterConfigStore is cluster-scoped
// (no namespace); ConfigStore is looked up in the ExternalConfig's namespace.
func (r *ExternalConfigReconciler) resolveStoreSpec(
	ctx context.Context,
	ec *syncv1alpha1.ExternalConfig,
) (*syncv1alpha1.ConfigStoreSpec, error) {
	ref := ec.Spec.StoreRef

	switch ref.Kind {
	case "", "ConfigStore":
		var store syncv1alpha1.ConfigStore
		if err := r.Get(ctx, types.NamespacedName{
			Name:      ref.Name,
			Namespace: ec.Namespace,
		}, &store); err != nil {
			return nil, fmt.Errorf("ConfigStore %q not found: %w", ref.Name, err)
		}
		return &store.Spec, nil

	case "ClusterConfigStore":
		var store syncv1alpha1.ClusterConfigStore
		if err := r.Get(ctx, types.NamespacedName{
			Name: ref.Name,
		}, &store); err != nil {
			return nil, fmt.Errorf("ClusterConfigStore %q not found: %w", ref.Name, err)
		}
		return &store.Spec, nil

	default:
		return nil, fmt.Errorf("unknown store kind %q, must be ConfigStore or ClusterConfigStore", ref.Kind)
	}
}

// buildProvider instantiates the correct provider based on the ConfigStore spec.
func (r *ExternalConfigReconciler) buildProvider(
	ctx context.Context,
	ec *syncv1alpha1.ExternalConfig,
	store *syncv1alpha1.ConfigStoreSpec,
) (provider.Provider, error) {
	switch store.Provider {
	case syncv1alpha1.ProviderGit:
		if store.Git == nil {
			return nil, fmt.Errorf("store %q has provider Git but no git config", ec.Spec.StoreRef.Name)
		}
		auth, err := r.resolveGitAuth(ctx, ec.Namespace, store.Git)
		if err != nil {
			return nil, err
		}
		return provider.NewGit(store.Git.URL, store.Git.Branch, auth), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", store.Provider)
	}
}

// resolveGitAuth reads credentials from the referenced Secret (if any).
func (r *ExternalConfigReconciler) resolveGitAuth(
	ctx context.Context,
	namespace string,
	cfg *syncv1alpha1.GitProvider,
) (*provider.GitAuth, error) {
	return resolveGitAuthFromClient(ctx, r.Client, namespace, cfg)
}

// setFailed marks the ExternalConfig as not-Ready and requeues after refreshInterval.
func (r *ExternalConfigReconciler) setFailed(
	ctx context.Context,
	ec *syncv1alpha1.ExternalConfig,
	reason, message string,
) (ctrl.Result, error) {
	if err := r.markFailed(ctx, ec, reason, message); err != nil {
		logf.FromContext(ctx).Error(err, "failed to update status condition")
	}
	return ctrl.Result{RequeueAfter: ec.Spec.RefreshInterval.Duration}, nil
}

// markFailed updates the Ready condition to False and clears stale KeyMappings.
func (r *ExternalConfigReconciler) markFailed(
	ctx context.Context,
	ec *syncv1alpha1.ExternalConfig,
	reason, message string,
) error {
	logf.FromContext(ctx).Error(errors.New(reason), message)
	ec.Status.KeyMappings = nil
	apimeta.SetStatusCondition(&ec.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: ec.Generation,
	})
	return r.Status().Update(ctx, ec)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExternalConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&syncv1alpha1.ExternalConfig{},
		".spec.storeRef.name",
		func(obj client.Object) []string {
			ec := obj.(*syncv1alpha1.ExternalConfig)
			return []string{ec.Spec.StoreRef.Name}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&syncv1alpha1.ExternalConfig{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		// Also watch ConfigMaps we own — if someone deletes one manually,
		// the controller will re-create it.
		Owns(&corev1.ConfigMap{},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		// Watch ClusterConfigStore: if a cluster-scoped store changes, re-reconcile
		// all ExternalConfigs that reference it.
		Watches(&syncv1alpha1.ClusterConfigStore{}, handler.EnqueueRequestsFromMapFunc(
			r.externalConfigsForClusterStore,
		)).
		// Watch ConfigStore: if a namespace-scoped store changes, re-reconcile
		// all ExternalConfigs in the same namespace that reference it.
		Watches(&syncv1alpha1.ConfigStore{}, handler.EnqueueRequestsFromMapFunc(
			r.externalConfigsForStore,
		)).
		Named("externalconfig").
		Complete(r)
}

// externalConfigsForClusterStore maps a ClusterConfigStore change to reconcile
// requests for every ExternalConfig that references it (across all namespaces).
func (r *ExternalConfigReconciler) externalConfigsForClusterStore(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	var ecList syncv1alpha1.ExternalConfigList
	if err := r.List(ctx, &ecList, client.MatchingFields{".spec.storeRef.name": obj.GetName()}); err != nil {
		logf.FromContext(ctx).Error(err, "failed to list ExternalConfigs for ClusterConfigStore", "store", obj.GetName())
		return nil
	}

	var requests []reconcile.Request
	for _, ec := range ecList.Items {
		if ec.Spec.StoreRef.Kind == "ClusterConfigStore" {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ec.Name,
					Namespace: ec.Namespace,
				},
			})
		}
	}
	return requests
}

// externalConfigsForStore maps a ConfigStore change to reconcile requests for
// every ExternalConfig in the same namespace that references it.
func (r *ExternalConfigReconciler) externalConfigsForStore(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	var ecList syncv1alpha1.ExternalConfigList
	if err := r.List(ctx, &ecList,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{".spec.storeRef.name": obj.GetName()},
	); err != nil {
		logf.FromContext(ctx).Error(err, "failed to list ExternalConfigs for ConfigStore", "store", obj.GetName())
		return nil
	}

	var requests []reconcile.Request
	for _, ec := range ecList.Items {
		if ec.Spec.StoreRef.Kind == "" || ec.Spec.StoreRef.Kind == "ConfigStore" {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ec.Name,
					Namespace: ec.Namespace,
				},
			})
		}
	}
	return requests
}
