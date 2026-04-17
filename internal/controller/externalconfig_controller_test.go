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
	"os"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
)

var _ = Describe("ExternalConfig Controller", Ordered, func() {
	const ns = "default"

	var (
		repoURL    string
		repoBranch string
	)

	// Set up a temporary local git repo once for all tests in this block.
	BeforeAll(func() {
		dir, err := os.MkdirTemp("", "lumos-ctrl-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, dir)

		branch, err := initTestRepo(dir)
		Expect(err).NotTo(HaveOccurred())

		repoURL = "file://" + dir
		repoBranch = branch
	})

	// ── helpers ──────────────────────────────────────────────────────────────

	reconciler := func() *ExternalConfigReconciler {
		return &ExternalConfigReconciler{
			Client:   k8sClient,
			Scheme:   k8sClient.Scheme(),
			Recorder: record.NewFakeRecorder(100),
		}
	}

	reconcile_ := func(name string) (reconcile.Result, error) {
		return reconciler().Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: ns},
		})
	}

	createStore := func(name, url, branch string) {
		store := &syncv1alpha1.ConfigStore{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: syncv1alpha1.ConfigStoreSpec{
				Provider: syncv1alpha1.ProviderGit,
				Git: &syncv1alpha1.GitProvider{
					URL:    url,
					Branch: branch,
				},
			},
		}
		Expect(k8sClient.Create(ctx, store)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, store)
	}

	createEC := func(name, storeName string, data []syncv1alpha1.ExternalConfigData) {
		ec := &syncv1alpha1.ExternalConfig{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: syncv1alpha1.ExternalConfigSpec{
				StoreRef:        syncv1alpha1.ConfigStoreRef{Name: storeName},
				RefreshInterval: metav1.Duration{Duration: 5 * time.Minute},
				Data:            data,
			},
		}
		Expect(k8sClient.Create(ctx, ec)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, ec)
	}

	getEC := func(name string) *syncv1alpha1.ExternalConfig {
		ec := &syncv1alpha1.ExternalConfig{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, ec)).To(Succeed())
		return ec
	}

	getConfigMap := func(name string) *corev1.ConfigMap {
		cm := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, cm)).To(Succeed())
		return cm
	}

	// ── StoreNotFound ─────────────────────────────────────────────────────────

	Context("when ConfigStore does not exist", func() {
		const ecName = "ec-no-store"

		BeforeEach(func() {
			createEC(ecName, "nonexistent-store", []syncv1alpha1.ExternalConfigData{
				{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
			})
		})

		It("sets Ready=False with reason StoreNotFound", func() {
			_, err := reconcile_(ecName)
			Expect(err).NotTo(HaveOccurred())

			ec := getEC(ecName)
			cond := apimeta.FindStatusCondition(ec.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("StoreNotFound"))
		})
	})

	// ── Success: Raw ─────────────────────────────────────────────────────────

	Context("when ConfigStore and repo are valid", func() {
		const (
			storeName = "git-store-ctrl"
			ecName    = "ec-raw"
		)

		BeforeEach(func() {
			createStore(storeName, repoURL, repoBranch)
			createEC(ecName, storeName, []syncv1alpha1.ExternalConfigData{
				{Source: "plain.txt", Key: "plain", Format: syncv1alpha1.FormatRaw},
			})
		})

		It("creates ConfigMap with fetched data and sets Ready=True", func() {
			result, err := reconcile_(ecName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(5 * time.Minute))

			cm := getConfigMap(ecName)
			Expect(cm.Data).To(HaveKeyWithValue("plain", "hello world"))

			ec := getEC(ecName)
			cond := apimeta.FindStatusCondition(ec.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal("Synced"))
			Expect(ec.Status.SyncedAt).NotTo(BeNil())
			Expect(ec.Status.ObservedVersion).NotTo(BeEmpty())
		})
	})

	// ── Success: Env (.env) ──────────────────────────────────────────────────

	Context("when format is Env with .env file", func() {
		const (
			storeName = "git-store-dotenv"
			ecName    = "ec-dotenv"
		)

		BeforeEach(func() {
			createStore(storeName, repoURL, repoBranch)
			createEC(ecName, storeName, []syncv1alpha1.ExternalConfigData{
				{Source: "config/.env", Format: syncv1alpha1.FormatEnv},
			})
		})

		It("creates ConfigMap with entries from .env", func() {
			_, err := reconcile_(ecName)
			Expect(err).NotTo(HaveOccurred())

			cm := getConfigMap(ecName)
			Expect(cm.Data).To(HaveKeyWithValue("DB_HOST", "localhost"))
			Expect(cm.Data).To(HaveKeyWithValue("DB_PORT", "5432"))
		})
	})

	// ── Success: Env (JSON) ──────────────────────────────────────────────────

	Context("when format is Env with JSON file", func() {
		const (
			storeName = "git-store-json"
			ecName    = "ec-json"
		)

		BeforeEach(func() {
			createStore(storeName, repoURL, repoBranch)
			createEC(ecName, storeName, []syncv1alpha1.ExternalConfigData{
				{Source: "config/env.json", Format: syncv1alpha1.FormatEnv},
			})
		})

		It("creates ConfigMap with entries from JSON", func() {
			_, err := reconcile_(ecName)
			Expect(err).NotTo(HaveOccurred())

			cm := getConfigMap(ecName)
			Expect(cm.Data).To(HaveKeyWithValue("API_KEY", "secret"))
			Expect(cm.Data).To(HaveKeyWithValue("TIMEOUT", "30s"))
		})
	})
})

// initTestRepo creates a minimal git repo with test files and returns the branch name.
func initTestRepo(dir string) (string, error) {
	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		return "", err
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	files := map[string]string{
		"plain.txt": "hello world",
		"config/.env": `
DB_HOST=localhost
DB_PORT=5432
`,
		"config/env.json": `{"API_KEY": "secret", "TIMEOUT": "30s"}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return "", err
		}
		if _, err := w.Add(path); err != nil {
			return "", err
		}
	}

	if _, err := w.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	}); err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	return head.Name().Short(), nil
}
