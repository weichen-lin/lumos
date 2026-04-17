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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigStoreProvider defines which backend this store connects to.
// +kubebuilder:validation:Enum=Git
type ConfigStoreProvider string

const (
	ProviderGit ConfigStoreProvider = "Git"
)

// GitProvider holds connection details for a Git repository.
// +kubebuilder:validation:XValidation:rule="!self.url.startsWith('git@') || has(self.secretRef)",message="secretRef is required when using an SSH URL (git@...)"
type GitProvider struct {
	// url is the HTTPS or SSH URL of the repository.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// branch to track. Defaults to "main".
	// +kubebuilder:default=main
	// +optional
	Branch string `json:"branch,omitempty"`

	// secretRef points to a Secret containing credentials.
	// For HTTPS: keys "username" and "password" (or "token").
	// For SSH: key "sshPrivateKey".
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// ConfigStoreSpec defines the desired state of ConfigStore.
type ConfigStoreSpec struct {
	// provider selects the backend type for this store.
	// +kubebuilder:validation:Required
	Provider ConfigStoreProvider `json:"provider"`

	// git configures the Git backend. Required when provider is Git.
	// +optional
	Git *GitProvider `json:"git,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ConfigStore is the Schema for the configstores API.
type ConfigStore struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ConfigStoreSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// ConfigStoreList contains a list of ConfigStore.
type ConfigStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ConfigStore `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ClusterConfigStore is the Schema for the clusterconfigstores API.
// It is cluster-scoped and can be referenced from any namespace.
type ClusterConfigStore struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ConfigStoreSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// ClusterConfigStoreList contains a list of ClusterConfigStore.
type ClusterConfigStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ClusterConfigStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigStore{}, &ConfigStoreList{})
	SchemeBuilder.Register(&ClusterConfigStore{}, &ClusterConfigStoreList{})
}
