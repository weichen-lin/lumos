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

package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
	lumosfrontend "github.com/weichen-lin/lumos/frontend"
)

// Server exposes a REST API for the Lumos dashboard and serves the compiled
// frontend assets.
type Server struct {
	client client.Client
	addr   string
}

const (
	storeKindConfigStore        = "ConfigStore"
	storeKindClusterConfigStore = "ClusterConfigStore"
	statusSynced                = "Synced"
	statusError                 = "Error"
	statusStale                 = "Stale"
	conditionTypeReady          = "Ready"
)

// New creates a new Server. addr is the listen address, e.g. ":8090".
func New(c client.Client, addr string) *Server {
	return &Server{client: c, addr: addr}
}

// Start registers routes and blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// ── API routes ────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/config-stores", s.handleConfigStores)
	mux.HandleFunc("GET /api/config-stores/{uid}", s.handleConfigStoreDetail)
	mux.HandleFunc("GET /api/external-configs", s.handleExternalConfigs)
	mux.HandleFunc("GET /api/external-configs/{uid}", s.handleExternalConfigDetail)
	mux.HandleFunc("GET /api/dashboard/config-stats", s.handleConfigDashboardStats)
	mux.HandleFunc("GET /api/encrypted-secrets", s.handleEncryptedSecrets)
	mux.HandleFunc("GET /api/encrypted-secrets/{uid}", s.handleEncryptedSecretDetail)

	// ── Static frontend (SPA) ─────────────────────────────────────────────────
	distFS, err := fs.Sub(lumosfrontend.FS, "dist")
	if err != nil {
		return fmt.Errorf("sub frontend FS: %w", err)
	}
	mux.Handle("/", spaHandler(http.FileServerFS(distFS), distFS))

	srv := &http.Server{Addr: s.addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// spaHandler wraps a file server to support SPA client-side routing.
// If a requested file doesn't exist, it serves index.html instead of a 404.
func spaHandler(fileServer http.Handler, staticFS fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If it's an API request, don't fallback to index.html
		if strings.HasPrefix(r.URL.Path, "/api") {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if the file exists in the filesystem
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := staticFS.Open(path)
		if err != nil {
			// File doesn't exist, serve index.html for client-side routing
			index, err := staticFS.Open("index.html")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusInternalServerError)
				return
			}
			defer func() {
				_ = index.Close()
			}()

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = io.Copy(w, index)
			return
		}
		_ = f.Close()

		fileServer.ServeHTTP(w, r)
	})
}

// ── Response types ────────────────────────────────────────────────────────────

type configStoreResp struct {
	UID        string `json:"uid"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Provider   string `json:"provider"`
	Namespace  string `json:"namespace,omitempty"`
	UsageCount int    `json:"usageCount"`
}

type externalConfigResp struct {
	UID         string     `json:"uid"`
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Namespace   string     `json:"namespace"`
	ConfigStore string     `json:"configStore"`
	StoreType   string     `json:"storeType"`
	Status      string     `json:"status"`
	Message     string     `json:"message,omitempty"`
	LastSync    *time.Time `json:"lastSync"`
	NextSync    *time.Time `json:"nextSync"`
	CommitSha   string     `json:"commitSha,omitempty"`
}

type configDetailResp struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Status          string            `json:"status"`
	Message         string            `json:"message,omitempty"`
	LastSync        *time.Time        `json:"lastSync"`
	NextSync        *time.Time        `json:"nextSync"`
	RefreshInterval string            `json:"refreshInterval"`
	Store           storeInfo         `json:"store"`
	Data            []dataEntry       `json:"data"`
	Events          []eventEntry      `json:"events"`
	Labels          map[string]string `json:"labels"`
}

type storeInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Provider string `json:"provider"`
}

type dataEntry struct {
	Source      string     `json:"source"`
	Key         string     `json:"key"`
	Value       string     `json:"value"`
	Format      string     `json:"format"`
	CommitSha   string     `json:"commitSha,omitempty"`
	LastChanged *time.Time `json:"lastChanged"`
}

type eventEntry struct {
	ID      string     `json:"id"`
	Type    string     `json:"type"`
	Reason  string     `json:"reason"`
	Message string     `json:"message"`
	Time    *time.Time `json:"time"`
}

type configStoreDetailResp struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Provider        string   `json:"provider"`
	Namespace       string   `json:"namespace,omitempty"`
	Git             *gitInfo `json:"git,omitempty"`
	ExternalConfigs []ecRef  `json:"externalConfigs"`
}

type gitInfo struct {
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type ecRef struct {
	UID       string     `json:"uid"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Namespace string     `json:"namespace"`
	Status    string     `json:"status"`
	LastSync  *time.Time `json:"lastSync"`
}

type configDashboardStats struct {
	Summary         summaryStats      `json:"summary"`
	Providers       []providerStat    `json:"providers"`
	RecentEvents    []recentEvent     `json:"recentEvents"`
	NamespaceHealth []namespaceHealth `json:"namespaceHealth"`
}

type summaryStats struct {
	Synced   int `json:"synced"`
	Error    int `json:"error"`
	Stale    int `json:"stale"`
	Expiring int `json:"expiring"`
}

type providerStat struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
	Color string `json:"color"`
}

type recentEvent struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	Time      *time.Time `json:"time"`
	Target    string     `json:"target"`
	Message   string     `json:"message"`
	CommitSha string     `json:"commitSha,omitempty"`
}

type namespaceHealth struct {
	Name     string `json:"name"`
	Synced   int    `json:"synced"`
	Error    int    `json:"error"`
	Stale    int    `json:"stale"`
	Coverage int    `json:"coverage"`
}

type encryptedSecretResp struct {
	UID             string     `json:"uid"`
	Name            string     `json:"name"`
	Namespace       string     `json:"namespace"`
	Store           string     `json:"store"`
	AgeKeyRef       string     `json:"ageKeyRef"`
	TargetSecret    string     `json:"targetSecret"`
	Status          string     `json:"status"`
	Message         string     `json:"message,omitempty"`
	LastSync        *time.Time `json:"lastSync"`
	NextSync        *time.Time `json:"nextSync"`
	CommitSha       string     `json:"commitSha,omitempty"`
	Sources         []string   `json:"sources"`
	RefreshInterval string     `json:"refreshInterval"`
}

type encryptedSecretDetailResp struct {
	UID             string       `json:"uid"`
	Name            string       `json:"name"`
	Namespace       string       `json:"namespace"`
	Store           string       `json:"store"`
	AgeKeyRef       string       `json:"ageKeyRef"`
	TargetSecret    string       `json:"targetSecret"`
	Status          string       `json:"status"`
	Message         string       `json:"message,omitempty"`
	LastSync        *time.Time   `json:"lastSync"`
	NextSync        *time.Time   `json:"nextSync"`
	CommitSha       string       `json:"commitSha,omitempty"`
	Sources         []string     `json:"sources"`
	RefreshInterval string       `json:"refreshInterval"`
	Events          []eventEntry `json:"events"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (s *Server) handleConfigStores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var stores syncv1alpha1.ConfigStoreList
	if err := s.client.List(ctx, &stores); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var clusterStores syncv1alpha1.ClusterConfigStoreList
	if err := s.client.List(ctx, &clusterStores); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Count ExternalConfig references per store.
	var ecList syncv1alpha1.ExternalConfigList
	if err := s.client.List(ctx, &ecList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	usage := map[string]int{}
	for _, ec := range ecList.Items {
		kind := ec.Spec.StoreRef.Kind
		if kind == "" {
			kind = storeKindConfigStore
		}
		usage[kind+"/"+ec.Namespace+"/"+ec.Spec.StoreRef.Name]++
	}

	result := make([]configStoreResp, 0, len(stores.Items)+len(clusterStores.Items))

	for i := range stores.Items {
		st := &stores.Items[i]
		result = append(result, configStoreResp{
			UID:        string(st.UID),
			ID:         st.Namespace + "--" + st.Name,
			Name:       st.Name,
			Type:       storeKindConfigStore,
			Provider:   string(st.Spec.Provider),
			Namespace:  st.Namespace,
			UsageCount: usage[storeKindConfigStore+"/"+st.Namespace+"/"+st.Name],
		})
	}

	for i := range clusterStores.Items {
		cs := &clusterStores.Items[i]
		result = append(result, configStoreResp{
			UID:        string(cs.UID),
			ID:         cs.Name,
			Name:       cs.Name,
			Type:       storeKindClusterConfigStore,
			Provider:   string(cs.Spec.Provider),
			UsageCount: usage[storeKindClusterConfigStore+"//"+cs.Name],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		if result[i].Namespace != result[j].Namespace {
			return result[i].Namespace < result[j].Namespace
		}
		return result[i].Name < result[j].Name
	})

	writeJSON(w, result)
}

func (s *Server) handleConfigStoreDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := r.PathValue("uid")

	var spec *syncv1alpha1.ConfigStoreSpec
	var resp configStoreDetailResp

	// Search ConfigStores first, then ClusterConfigStores.
	var stores syncv1alpha1.ConfigStoreList
	if err := s.client.List(ctx, &stores); err == nil {
		for i := range stores.Items {
			st := &stores.Items[i]
			if string(st.UID) == uid {
				spec = &st.Spec
				resp = configStoreDetailResp{
					ID:        st.Namespace + "--" + st.Name,
					Name:      st.Name,
					Type:      storeKindConfigStore,
					Provider:  string(st.Spec.Provider),
					Namespace: st.Namespace,
				}
				break
			}
		}
	}
	if spec == nil {
		var clusterStores syncv1alpha1.ClusterConfigStoreList
		if err := s.client.List(ctx, &clusterStores); err == nil {
			for i := range clusterStores.Items {
				cs := &clusterStores.Items[i]
				if string(cs.UID) == uid {
					spec = &cs.Spec
					resp = configStoreDetailResp{
						ID:       cs.Name,
						Name:     cs.Name,
						Type:     storeKindClusterConfigStore,
						Provider: string(cs.Spec.Provider),
					}
					break
				}
			}
		}
	}
	if spec == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("ConfigStore with uid %q not found", uid))
		return
	}

	if spec.Git != nil {
		resp.Git = &gitInfo{URL: spec.Git.URL, Branch: spec.Git.Branch}
	}

	// Find all ExternalConfigs that reference this store.
	var ecList syncv1alpha1.ExternalConfigList
	if err := s.client.List(ctx, &ecList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	refs := make([]ecRef, 0)
	for i := range ecList.Items {
		ec := &ecList.Items[i]
		kind := ec.Spec.StoreRef.Kind
		if kind == "" {
			kind = storeKindConfigStore
		}
		var matches bool
		if resp.Type == storeKindClusterConfigStore {
			matches = kind == storeKindClusterConfigStore && ec.Spec.StoreRef.Name == resp.Name
		} else {
			matches = kind == storeKindConfigStore && ec.Spec.StoreRef.Name == resp.Name && ec.Namespace == resp.Namespace
		}
		if matches {
			refs = append(refs, ecRef{
				UID:       string(ec.UID),
				ID:        ec.Namespace + "--" + ec.Name,
				Name:      ec.Name,
				Namespace: ec.Namespace,
				Status:    deriveStatus(ec),
				LastSync:  timePtr(ec.Status.SyncedAt),
			})
		}
	}
	resp.ExternalConfigs = refs

	writeJSON(w, resp)
}

func (s *Server) handleExternalConfigs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var ecList syncv1alpha1.ExternalConfigList
	if err := s.client.List(ctx, &ecList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]externalConfigResp, 0, len(ecList.Items))
	for i := range ecList.Items {
		ec := &ecList.Items[i]
		storeKind := ec.Spec.StoreRef.Kind
		if storeKind == "" {
			storeKind = storeKindConfigStore
		}
		resp := externalConfigResp{
			UID:         string(ec.UID),
			ID:          ec.Namespace + "--" + ec.Name,
			Name:        ec.Name,
			Namespace:   ec.Namespace,
			ConfigStore: ec.Spec.StoreRef.Name,
			StoreType:   storeKind,
			Status:      deriveStatus(ec),
			Message:     deriveMessage(ec),
			LastSync:    timePtr(ec.Status.SyncedAt),
			NextSync:    nextSyncTime(ec.Status.SyncedAt, ec.Spec.RefreshInterval),
			CommitSha:   shortSHA(ec.Status.ObservedVersion),
		}
		result = append(result, resp)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace != result[j].Namespace {
			return result[i].Namespace < result[j].Namespace
		}
		return result[i].Name < result[j].Name
	})

	writeJSON(w, result)
}

func (s *Server) handleExternalConfigDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := r.PathValue("uid")

	var ecList syncv1alpha1.ExternalConfigList
	if err := s.client.List(ctx, &ecList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var ec *syncv1alpha1.ExternalConfig
	for i := range ecList.Items {
		if string(ecList.Items[i].UID) == uid {
			ec = &ecList.Items[i]
			break
		}
	}
	if ec == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("ExternalConfig with uid %q not found", uid))
		return
	}

	storeKind := ec.Spec.StoreRef.Kind
	if storeKind == "" {
		storeKind = "ConfigStore"
	}

	cmName := ec.Name
	if ec.Spec.Target != nil && ec.Spec.Target.Name != "" {
		cmName = ec.Spec.Target.Name
	}

	cmData := s.readConfigMapData(ctx, ec.Namespace, cmName)
	entries := buildDataEntries(ec, cmData)

	labels := ec.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	writeJSON(w, configDetailResp{
		ID:              ec.Namespace + "--" + ec.Name,
		Name:            ec.Name,
		Namespace:       ec.Namespace,
		Status:          deriveStatus(ec),
		Message:         deriveMessage(ec),
		LastSync:        timePtr(ec.Status.SyncedAt),
		NextSync:        nextSyncTime(ec.Status.SyncedAt, ec.Spec.RefreshInterval),
		RefreshInterval: ec.Spec.RefreshInterval.Duration.String(),
		Store: storeInfo{
			Name:     ec.Spec.StoreRef.Name,
			Type:     storeKind,
			Provider: s.resolveProvider(ctx, ec),
		},
		Data:   entries,
		Events: s.readEvents(ctx, ec),
		Labels: labels,
	})
}

func (s *Server) handleConfigDashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var ecList syncv1alpha1.ExternalConfigList
	if err := s.client.List(ctx, &ecList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var esList syncv1alpha1.EncryptedSecretList
	if err := s.client.List(ctx, &esList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	summary := summaryStats{}
	providerCount := map[string]int{}
	nsMap := map[string]*namespaceHealth{}
	recent := make([]recentEvent, 0)

	// Process ExternalConfigs
	for i := range ecList.Items {
		ec := &ecList.Items[i]
		status := deriveStatus(ec)
		provider := s.resolveProvider(ctx, ec)

		switch status {
		case statusSynced:
			summary.Synced++
		case statusError:
			summary.Error++
		default:
			summary.Stale++
		}
		providerCount[provider]++

		ns := ec.Namespace
		if nsMap[ns] == nil {
			nsMap[ns] = &namespaceHealth{Name: ns}
		}
		switch status {
		case statusSynced:
			nsMap[ns].Synced++
		case statusError:
			nsMap[ns].Error++
		default:
			nsMap[ns].Stale++
		}

		evStatus := strings.ToLower(status)
		if status == statusStale {
			evStatus = "warning"
		}
		re := recentEvent{
			ID:     "ec-" + string(ec.UID),
			Status: evStatus,
			Time:   timePtr(ec.Status.SyncedAt),
			Target: ec.Namespace + "/" + ec.Name,
		}
		if msg := deriveMessage(ec); msg != "" {
			re.Message = msg
		} else if ec.Status.ObservedVersion != "" {
			re.Message = "synced from " + shortSHA(ec.Status.ObservedVersion)
			re.CommitSha = shortSHA(ec.Status.ObservedVersion)
		}
		recent = append(recent, re)
	}

	// Process EncryptedSecrets
	for i := range esList.Items {
		es := &esList.Items[i]
		status := deriveEncSecretStatus(es)
		provider := "Git" // EncryptedSecrets are always from Git stores currently

		switch status {
		case statusSynced:
			summary.Synced++
		case statusError:
			summary.Error++
		default:
			summary.Stale++
		}
		providerCount[provider]++

		ns := es.Namespace
		if nsMap[ns] == nil {
			nsMap[ns] = &namespaceHealth{Name: ns}
		}
		switch status {
		case statusSynced:
			nsMap[ns].Synced++
		case statusError:
			nsMap[ns].Error++
		default:
			nsMap[ns].Stale++
		}

		evStatus := strings.ToLower(status)
		if status == statusStale {
			evStatus = "warning"
		}
		re := recentEvent{
			ID:     "es-" + string(es.UID),
			Status: evStatus,
			Time:   timePtr(es.Status.SyncedAt),
			Target: es.Namespace + "/" + es.Name,
		}
		if msg := deriveEncSecretMessage(es); msg != "" {
			re.Message = msg
		} else if es.Status.ObservedVersion != "" {
			re.Message = "synced from " + shortSHA(es.Status.ObservedVersion)
			re.CommitSha = shortSHA(es.Status.ObservedVersion)
		}
		recent = append(recent, re)
	}

	// Sort recent events by time descending
	sort.Slice(recent, func(i, j int) bool {
		if recent[i].Time == nil {
			return false
		}
		if recent[j].Time == nil {
			return true
		}
		return recent[i].Time.After(*recent[j].Time)
	})

	// Limit to top 10 recent events
	if len(recent) > 10 {
		recent = recent[:10]
	}

	providerColors := map[string]string{
		"Git": "oklch(0.62 0.14 39.15)",
	}
	providers := make([]providerStat, 0, len(providerCount))
	for p, count := range providerCount {
		color, ok := providerColors[p]
		if !ok {
			color = "oklch(0.5 0.2 300)"
		}
		providers = append(providers, providerStat{Name: p, Value: count, Color: color})
	}

	nsHealth := make([]namespaceHealth, 0, len(nsMap))
	for _, h := range nsMap {
		total := h.Synced + h.Error + h.Stale
		if total > 0 {
			h.Coverage = (h.Synced * 100) / total
		}
		nsHealth = append(nsHealth, *h)
	}
	sort.Slice(nsHealth, func(i, j int) bool {
		return nsHealth[i].Name < nsHealth[j].Name
	})

	writeJSON(w, configDashboardStats{
		Summary:         summary,
		Providers:       providers,
		RecentEvents:    recent,
		NamespaceHealth: nsHealth,
	})
}

func (s *Server) handleEncryptedSecrets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var esList syncv1alpha1.EncryptedSecretList
	if err := s.client.List(ctx, &esList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]encryptedSecretResp, 0, len(esList.Items))
	for i := range esList.Items {
		es := &esList.Items[i]

		targetName := es.Name
		if es.Spec.Target != nil && es.Spec.Target.Name != "" {
			targetName = es.Spec.Target.Name
		}

		sources := make([]string, len(es.Spec.Data))
		for j, d := range es.Spec.Data {
			sources[j] = d.Source
		}

		result = append(result, encryptedSecretResp{
			UID:             string(es.UID),
			Name:            es.Name,
			Namespace:       es.Namespace,
			Store:           es.Spec.StoreRef.Name,
			AgeKeyRef:       es.Spec.AgeKeyRef.Name,
			TargetSecret:    targetName,
			Status:          deriveEncSecretStatus(es),
			Message:         deriveEncSecretMessage(es),
			LastSync:        timePtr(es.Status.SyncedAt),
			NextSync:        nextSyncTime(es.Status.SyncedAt, es.Spec.RefreshInterval),
			CommitSha:       shortSHA(es.Status.ObservedVersion),
			Sources:         sources,
			RefreshInterval: es.Spec.RefreshInterval.Duration.String(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace != result[j].Namespace {
			return result[i].Namespace < result[j].Namespace
		}
		return result[i].Name < result[j].Name
	})

	writeJSON(w, result)
}

func (s *Server) handleEncryptedSecretDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := r.PathValue("uid")

	var esList syncv1alpha1.EncryptedSecretList
	if err := s.client.List(ctx, &esList); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var es *syncv1alpha1.EncryptedSecret
	for i := range esList.Items {
		if string(esList.Items[i].UID) == uid {
			es = &esList.Items[i]
			break
		}
	}
	if es == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("EncryptedSecret with uid %q not found", uid))
		return
	}

	targetName := es.Name
	if es.Spec.Target != nil && es.Spec.Target.Name != "" {
		targetName = es.Spec.Target.Name
	}

	sources := make([]string, len(es.Spec.Data))
	for i, d := range es.Spec.Data {
		sources[i] = d.Source
	}

	// Fetch K8s Events for this EncryptedSecret.
	var eventList corev1.EventList
	events := []eventEntry{}
	if err := s.client.List(ctx, &eventList,
		client.InNamespace(es.Namespace),
		client.MatchingFields{"involvedObject.name": es.Name},
	); err == nil {
		for i := range eventList.Items {
			ev := &eventList.Items[i]
			if ev.InvolvedObject.Kind != "EncryptedSecret" {
				continue
			}
			t := ev.LastTimestamp.Time
			events = append(events, eventEntry{
				ID:      string(ev.UID),
				Type:    ev.Type,
				Reason:  ev.Reason,
				Message: ev.Message,
				Time:    &t,
			})
		}
	}

	writeJSON(w, encryptedSecretDetailResp{
		UID:             string(es.UID),
		Name:            es.Name,
		Namespace:       es.Namespace,
		Store:           es.Spec.StoreRef.Name,
		AgeKeyRef:       es.Spec.AgeKeyRef.Name,
		TargetSecret:    targetName,
		Status:          deriveEncSecretStatus(es),
		Message:         deriveEncSecretMessage(es),
		LastSync:        timePtr(es.Status.SyncedAt),
		NextSync:        nextSyncTime(es.Status.SyncedAt, es.Spec.RefreshInterval),
		CommitSha:       shortSHA(es.Status.ObservedVersion),
		Sources:         sources,
		RefreshInterval: es.Spec.RefreshInterval.Duration.String(),
		Events:          events,
	})
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// buildDataEntries builds the data entries for the detail view.
// If KeyMappings are available in status, each entry shows the correct
// remoteKey alongside its localKey(s). Otherwise it falls back to using the
// ConfigMap keys directly (remoteKey == localKey).
func buildDataEntries(ec *syncv1alpha1.ExternalConfig, cmData map[string]string) []dataEntry {
	// Create a map for quick format lookup from Spec
	formatMap := make(map[string]string)
	for _, d := range ec.Spec.Data {
		format := string(d.Format)
		if format == "" {
			format = "Raw"
		}
		formatMap[d.Source] = format
	}

	if len(ec.Status.KeyMappings) > 0 {
		entries := make([]dataEntry, 0, len(cmData))
		for _, m := range ec.Status.KeyMappings {
			format := formatMap[m.Source]
			for _, key := range m.Keys {
				entries = append(entries, dataEntry{
					Source:      m.Source,
					Key:         key,
					Value:       cmData[key],
					Format:      format,
					CommitSha:   shortSHA(ec.Status.ObservedVersion),
					LastChanged: timePtr(ec.Status.SyncedAt),
				})
			}
		}
		return entries
	}
	// Fallback: no mapping info yet (first sync pending).
	entries := make([]dataEntry, 0, len(cmData))
	fallbackKeys := make([]string, 0, len(cmData))
	for k := range cmData {
		fallbackKeys = append(fallbackKeys, k)
	}
	sort.Strings(fallbackKeys)

	for _, k := range fallbackKeys {
		v := cmData[k]
		entries = append(entries, dataEntry{
			Source:      k,
			Key:         k,
			Value:       v,
			Format:      "Raw", // Fallback assumes Raw
			CommitSha:   shortSHA(ec.Status.ObservedVersion),
			LastChanged: timePtr(ec.Status.SyncedAt),
		})
	}
	return entries
}

// readEvents lists the most recent Kubernetes Events for the given ExternalConfig.
func (s *Server) readEvents(ctx context.Context, ec *syncv1alpha1.ExternalConfig) []eventEntry {
	var eventList corev1.EventList
	if err := s.client.List(ctx, &eventList,
		client.InNamespace(ec.Namespace),
		client.MatchingFields{"involvedObject.name": ec.Name},
	); err != nil {
		return []eventEntry{}
	}

	entries := make([]eventEntry, 0, len(eventList.Items))
	for i := range eventList.Items {
		ev := &eventList.Items[i]
		if ev.InvolvedObject.Kind != "ExternalConfig" {
			continue
		}
		t := ev.LastTimestamp.Time
		entries = append(entries, eventEntry{
			ID:      string(ev.UID),
			Type:    ev.Type,
			Reason:  ev.Reason,
			Message: ev.Message,
			Time:    &t,
		})
	}
	return entries
}

func (s *Server) resolveProvider(ctx context.Context, ec *syncv1alpha1.ExternalConfig) string {
	kind := ec.Spec.StoreRef.Kind
	if kind == "" {
		kind = storeKindConfigStore
	}
	switch kind {
	case storeKindConfigStore:
		var store syncv1alpha1.ConfigStore
		if err := s.client.Get(ctx, client.ObjectKey{Namespace: ec.Namespace, Name: ec.Spec.StoreRef.Name}, &store); err == nil {
			return string(store.Spec.Provider)
		}
	case storeKindClusterConfigStore:
		var store syncv1alpha1.ClusterConfigStore
		if err := s.client.Get(ctx, client.ObjectKey{Name: ec.Spec.StoreRef.Name}, &store); err == nil {
			return string(store.Spec.Provider)
		}
	}
	return "Unknown"
}

func (s *Server) readConfigMapData(ctx context.Context, namespace, name string) map[string]string {
	var cm corev1.ConfigMap
	if err := s.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &cm); err != nil {
		return map[string]string{}
	}
	if cm.Data == nil {
		return map[string]string{}
	}
	return cm.Data
}

// ── Utility ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		log.Printf("writeError encode error: %v", err)
	}
}

func timePtr(t *metav1.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := t.Time
	return &v
}

func nextSyncTime(syncedAt *metav1.Time, interval metav1.Duration) *time.Time {
	if syncedAt == nil {
		return nil
	}
	v := syncedAt.Add(interval.Duration)
	return &v
}

func deriveStatus(ec *syncv1alpha1.ExternalConfig) string {
	for _, cond := range ec.Status.Conditions {
		if cond.Type == conditionTypeReady {
			if cond.Status == metav1.ConditionTrue {
				return statusSynced
			}
			return statusError
		}
	}
	return statusStale
}

func deriveMessage(ec *syncv1alpha1.ExternalConfig) string {
	for _, cond := range ec.Status.Conditions {
		if cond.Type == conditionTypeReady && cond.Status == metav1.ConditionFalse {
			return cond.Message
		}
	}
	return ""
}

func deriveEncSecretStatus(es *syncv1alpha1.EncryptedSecret) string {
	for _, cond := range es.Status.Conditions {
		if cond.Type == conditionTypeReady {
			if cond.Status == metav1.ConditionTrue {
				return statusSynced
			}
			return statusError
		}
	}
	return statusStale
}

func deriveEncSecretMessage(es *syncv1alpha1.EncryptedSecret) string {
	for _, cond := range es.Status.Conditions {
		if cond.Type == conditionTypeReady && cond.Status == metav1.ConditionFalse {
			return cond.Message
		}
	}
	return ""
}

func shortSHA(sha string) string {
	if len(sha) >= 7 {
		return sha[:7]
	}
	return sha
}
