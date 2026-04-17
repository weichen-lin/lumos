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

// EncryptedSecretSpec defines the desired state of EncryptedSecret.
type EncryptedSecretSpec struct {
	// StoreRef points to the ConfigStore (Git provider) that holds the encrypted files.
	StoreRef ConfigStoreRef `json:"storeRef"`

	// AgeKeyRef is the name of the K8s Secret in the same namespace that contains
	// the age private key under the "keys.txt" data key (AGE-SECRET-KEY-...).
	AgeKeyRef corev1.LocalObjectReference `json:"ageKeyRef"`

	// RefreshInterval defines how often to re-sync from the Git store.
	// +kubebuilder:default="5m"
	RefreshInterval metav1.Duration `json:"refreshInterval,omitempty"`

	// Data lists the SOPS-encrypted files to read from the Git store.
	// Each file is decrypted and its top-level keys are written into the target Secret.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=20
	Data []EncryptedSecretData `json:"data"`

	// Target defines the output K8s Secret. Defaults to the EncryptedSecret name.
	// +optional
	Target *EncryptedSecretTarget `json:"target,omitempty"`
}

// EncryptedSecretData references a single SOPS-encrypted file in the Git repository.
type EncryptedSecretData struct {
	// Source is the file path in the Git repository (e.g. "secrets/app.yaml").
	// The file must be SOPS-encrypted with the age key referenced by AgeKeyRef.
	Source string `json:"source"`
}

// EncryptedSecretTarget defines the output K8s Secret.
type EncryptedSecretTarget struct {
	// Name of the K8s Secret to create. Defaults to the EncryptedSecret name.
	// +optional
	Name string `json:"name,omitempty"`
}

// EncryptedSecretStatus defines the observed state of EncryptedSecret.
type EncryptedSecretStatus struct {
	// Conditions reflect the current state. The "Ready" condition tracks sync health.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// SyncedAt is the timestamp of the last successful sync.
	// +optional
	SyncedAt *metav1.Time `json:"syncedAt,omitempty"`

	// ObservedVersion is the Git commit SHA of the last successful sync.
	// +optional
	ObservedVersion string `json:"observedVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Store",type=string,JSONPath=".spec.storeRef.name"
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=".spec.target.name"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Synced At",type=date,JSONPath=".status.syncedAt"
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=".status.observedVersion"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// EncryptedSecret syncs SOPS-encrypted files from a Git repository and writes
// the decrypted key-value pairs into a Kubernetes Secret.
type EncryptedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EncryptedSecretSpec   `json:"spec"`
	Status EncryptedSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EncryptedSecretList contains a list of EncryptedSecret.
type EncryptedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EncryptedSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EncryptedSecret{}, &EncryptedSecretList{})
}
