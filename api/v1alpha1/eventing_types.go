/*
Copyright 2023.

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

// +kubebuilder:validation:Optional // This sets 'required' as the default behaviour.
//
//nolint:lll //this is annotation
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Eventing is the Schema for the eventing API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of Eventing"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the resource"
type Eventing struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:default:={backends:{type:"NATS"}, publisher:{replicas:{min:2,max:2}}}
	Spec   EventingSpec   `json:"spec,omitempty"`
	Status EventingStatus `json:"status,omitempty"`
}

// EventingStatus defines the observed state of Eventing
type EventingStatus struct {
	State      string             `json:"state"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// EventingSpec defines the desired state of Eventing
type EventingSpec struct {
	// Backends defines the list of eventing backends to provision.
	Backends Backend `json:"backends"`

	// Publisher defines the configurations for eventing-publisher-proxy.
	// +optional
	Publisher `json:"publisher,omitempty"`

	// Logging defines the log level for eventing-manager.
	// +optional
	Logging `json:"logging,omitempty"`

	// Annotations allows to add annotations to resources.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels allows to add Labels to resources.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// EventingList contains a list of Eventing
// +kubebuilder:object:root=true
type EventingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Eventing `json:"items"`
}

// Backend defines eventing backend.
type Backend struct {
	// Type defines which backend to use. The value is either `EventMesh`, or `NATS`.
	// +kubebuilder:validation:Enum=EventMesh;NATS
	Type string `json:"type"`

	// Config defines configuration for eventing backend.
	// +optional
	Config BackendConfig `json:"config"`
}

// BackendConfig defines configuration for eventing backend.
type BackendConfig struct {
	// NatsStorageType defines the storage type for stream data.
	// +optional
	// +kubebuilder:validation:Enum=File;Memory
	NATSStreamStorageType string `json:"natsStreamStorageType"`

	// NatsStreamReplicas defines the number of replicas for stream.
	// +optional
	NATSStreamReplicas int `json:"natsStreamReplicas"`

	// NatsMaxStreamSize defines the maximum storage size for stream data.
	// +optional
	NATSMaxStreamSize resource.Quantity `json:"natsMaxStreamSize"`

	// NatsMaxMsgsPerTopic limits how many messages in the NATS stream to retain per subject.
	// +optional
	NATSMaxMsgsPerTopic int `json:"natsMaxMsgsPerTopic"`

	// EventMeshSecret defines the namespaced name of K8s Secret containing EventMesh credentials. The format of name is "namespace/name".
	// +optional
	EventMeshSecret string `json:"eventMeshSecret"`
}

// Publisher defines the configurations for eventing-publisher-proxy.
type Publisher struct {
	// Replicas defines the scaling min/max for eventing-publisher-proxy.
	// +optional
	Replicas `json:"replicas,omitempty"`

	// Resources defines resources for eventing-publisher-proxy.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// Replicas defines min/max replicas for a resource.
type Replicas struct {
	// Min defines minimum number of replicas.
	// +optional
	Min int `json:"min,omitempty"`

	// Max defines maximum number of replicas.
	// +optional
	Max int `json:"max,omitempty"`
}

type Logging struct {
	// LogLevel defines the log level.
	// +optional
	// +kubebuilder:validation:Enum=Info;Warn;Error;debug
	LogLevel string `json:"logLevel,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Eventing{}, &EventingList{})
}
