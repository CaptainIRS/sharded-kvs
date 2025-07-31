/*
Copyright 2025.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KVStoreSpec defines the desired state of KVStore
type KVStoreSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// The number of shards to be used for the KVStore
	// +required
	// +kubebuilder:validation:Minimum=1
	Shards int32 `json:"shards"`

	// The number of replicas per shard in the Raft cluster
	// +required
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// Ingress domain
	// +required
	IngressDomain string `json:"ingressDomain"`

	// PodSpec of the replicas
	// +required
	PodSpec corev1.PodSpec `json:"podSpec"`
}

type RedistributionPhase string

const (
	PhaseNormal          RedistributionPhase = ""
	PhasePreparingScale  RedistributionPhase = "PreparingScale"
	PhaseStoppingWrites  RedistributionPhase = "StoppingWrites"
	PhaseSendingKeys     RedistributionPhase = "SendingKeys"
	PhasePurgingKeys     RedistributionPhase = "PurgingKeys"
	PhaseResumingWrites  RedistributionPhase = "ResumingWrites"
	PhaseFinalizingScale RedistributionPhase = "FinalizingScale"
	PhaseScaleComplete   RedistributionPhase = "ScaleComplete"
)

// KVStoreStatus defines the observed state of KVStore.
type KVStoreStatus struct {
	Phase                RedistributionPhase `json:"phase,omitempty"`
	CurrentShards        int                 `json:"currentShards"`
	TargetShards         int                 `json:"targetShards,omitempty"`
	ShardsStoppedWrites  []int               `json:"shardsStoppedWrites,omitempty"`
	ShardsCompletedSend  []int               `json:"shardsCompletedSend,omitempty"`
	ShardsCompletedPurge []int               `json:"shardsCompletedPurge,omitempty"`
	ShardsResumedWrites  []int               `json:"shardsResumedWrites,omitempty"`
	Message              string              `json:"message,omitempty"`
	LastTransition       metav1.Time         `json:"lastTransition,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KVStore is the Schema for the kvstores API
type KVStore struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of KVStore
	// +required
	Spec KVStoreSpec `json:"spec"`

	// status defines the observed state of KVStore
	// +optional
	Status KVStoreStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// KVStoreList contains a list of KVStore
type KVStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KVStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KVStore{}, &KVStoreList{})
}
