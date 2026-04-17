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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigStoreRef references either a ConfigStore (namespace-scoped) or a
// ClusterConfigStore (cluster-scoped).
type ConfigStoreRef struct {
	// name of the ConfigStore or ClusterConfigStore.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// kind of the store. Either ConfigStore or ClusterConfigStore.
	// Defaults to ConfigStore.
	// +kubebuilder:default=ConfigStore
	// +kubebuilder:validation:Enum=ConfigStore;ClusterConfigStore
	// +optional
	Kind string `json:"kind,omitempty"`
}

// DataFormat controls how a fetched value is written into the ConfigMap.
// +kubebuilder:validation:Enum=Raw;Env
type DataFormat string

const (
	// FormatRaw stores the entire fetched value under key. This is the default.
	FormatRaw DataFormat = "Raw"
	// FormatEnv parses flat key/value pairs from JSON or .env and stores each
	// as a separate ConfigMap entry.
	FormatEnv DataFormat = "Env"
)

// ExternalConfigData maps one remote path/key to one or more local ConfigMap keys.
// +kubebuilder:validation:XValidation:rule="self.format != 'Raw' || has(self.key)",message="key is required when format is Raw"
// +kubebuilder:validation:XValidation:rule="self.format != 'Env' || !has(self.key)",message="key must be empty when format is Env"
// +kubebuilder:validation:XValidation:rule="!has(self.key) || self.key.matches('^[-._a-zA-Z0-9]+$')",message="key must consist of alphanumeric characters, '-', '_' or '.'"
type ExternalConfigData struct {
	// source is the path or key in the remote source.
	// For Git: relative file path, e.g. "config/app.yaml".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=2048
	Source string `json:"source"`

	// key is the ConfigMap key name. Required when format=Raw.
	// Must be empty when format=Env.
	// +optional
	// +kubebuilder:validation:MaxLength=253
	Key string `json:"key,omitempty"`

	// format controls how the fetched value is written to the ConfigMap.
	// Raw (default): the entire value is stored under key.
	// Env: the value is parsed as flat JSON or .env and each entry is stored
	// as a separate ConfigMap key.
	// +kubebuilder:default=Raw
	// +optional
	Format DataFormat `json:"format,omitempty"`
}

// ExternalConfigTarget defines where the synced data is written.
type ExternalConfigTarget struct {
	// name of the ConfigMap to create or update.
	// Defaults to the ExternalConfig name if omitted.
	// +optional
	Name string `json:"name,omitempty"`
}

// ExternalConfigSpec defines the desired state of ExternalConfig.
type ExternalConfigSpec struct {
	// storeRef references the ConfigStore to pull config from.
	// +kubebuilder:validation:Required
	StoreRef ConfigStoreRef `json:"storeRef"`

	// refreshInterval controls how often the operator re-syncs.
	// Must be a valid Go duration string, e.g. "5m", "1h".
	// +kubebuilder:default="5m"
	RefreshInterval metav1.Duration `json:"refreshInterval"`

	// data lists the remote keys to sync and their local names.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=100
	Data []ExternalConfigData `json:"data"`

	// target defines where the synced config is written.
	// Defaults to a ConfigMap with the same name as this ExternalConfig.
	// +optional
	Target *ExternalConfigTarget `json:"target,omitempty"`
}

// KeyMapping records which ConfigMap keys were produced by one remote data entry.
type KeyMapping struct {
	// source is the source path or KV key.
	Source string `json:"source"`
	// keys are the ConfigMap keys written for this entry.
	Keys []string `json:"keys"`
}

// ExternalConfigStatus defines the observed state of ExternalConfig.
type ExternalConfigStatus struct {
	// conditions represent the current sync state.
	// "Ready" = True means the last sync succeeded.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// syncedAt is the timestamp of the last successful sync.
	// +optional
	SyncedAt *metav1.Time `json:"syncedAt,omitempty"`

	// observedVersion is a provider-specific version string
	// (commit SHA for Git).
	// +optional
	ObservedVersion string `json:"observedVersion,omitempty"`

	// keyMappings records which remote key produced which ConfigMap keys.
	// Populated after each successful sync.
	// +optional
	KeyMappings []KeyMapping `json:"keyMappings,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Store",type=string,JSONPath=`.spec.storeRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Synced At",type=date,JSONPath=`.status.syncedAt`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.observedVersion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ExternalConfig is the Schema for the externalconfigs API.
type ExternalConfig struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ExternalConfigSpec `json:"spec"`

	// +optional
	Status ExternalConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ExternalConfigList contains a list of ExternalConfig.
type ExternalConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ExternalConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalConfig{}, &ExternalConfigList{})
}
