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

// nolint:lll //this is annotation
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionReason string

type ConditionType string

const (
	StateReady      string = "Ready"
	StateError      string = "Error"
	StateProcessing string = "Processing"

	ConditionNATSAvailable       ConditionType = "NATSAvailable"
	ConditionPublisherProxyReady ConditionType = "PublisherProxyReady"

	ConditionReasonProcessing                 ConditionReason = "Processing"
	ConditionReasonDeployed                   ConditionReason = "Deployed"
	ConditionReasonDeployedFailed             ConditionReason = "DeployFailed"
	ConditionReasonDeploymentStatusSyncFailed ConditionReason = "DeploymentStatusSyncFailed"
	ConditionReasonNATSAvailable              ConditionReason = "Available"
	ConditionReasonNATSNotAvailable           ConditionReason = "NotAvailable"

	ConditionPublisherProxyReadyMessage      = "Publisher proxy is deployed"
	ConditionNATSAvailableMessage            = "NATS is available"
	ConditionPublisherProxyProcessingMessage = "Eventing publisher proxy deployment is in progress"
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

	// +kubebuilder:default:={backends:{{type:""},{type:"NATS", config:{natsStreamStorageType:"File", natsStreamReplicas:3, natsStreamMaxSize:"700Mi", natsMaxMsgsPerTopic:1000000}}}, logging:{logLevel:Info}, publisher:{replicas:{min:2,max:2}, resources:{limits:{cpu:"500m",memory:"512Mi"}, requests:{cpu:"10m",memory:"256Mi"}}}}
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
	// +kubebuilder:default:={{type:""},{type:"NATS", config:{natsStreamStorageType:"File", natsStreamReplicas:3, natsStreamMaxSize:"700Mi", natsMaxMsgsPerTopic:1000000}}}
	Backends []Backend `json:"backends"`

	// Publisher defines the configurations for eventing-publisher-proxy.
	// +kubebuilder:default:={replicas:{min:2,max:2}, resources:{limits:{cpu:"500m",memory:"512Mi"}, requests:{cpu:"10m",memory:"256Mi"}}}
	Publisher `json:"publisher,omitempty"`

	// Logging defines the log level for eventing-manager.
	// +kubebuilder:default:={logLevel:Info}
	Logging `json:"logging,omitempty"`

	// Annotations allows to add annotations to resources.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels allows to add Labels to resources.
	Labels map[string]string `json:"labels,omitempty"`
}

// +kubebuilder:object:root=true

// EventingList contains a list of Eventing
type EventingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Eventing `json:"items"`
}

type BackendType string

const (
	EventMeshBackendType BackendType = "EventMesh"
	NatsBackendType      BackendType = "NATS"
)

// Backend defines eventing backend.
type Backend struct {
	// Type defines which backend to use. The value is either `EventMesh`, or `NATS`.
	// +kubebuilder:default:="NATS"
	// +kubebuilder:validation:XValidation:rule="self=='NATS' || self=='EventMesh' || self==''", message="backend type can only be set to NATS or EventMesh"
	Type BackendType `json:"type"`

	// Config defines configuration for eventing backend.
	// +kubebuilder:default:={natsStreamStorageType:"File", natsStreamReplicas:3, natsStreamMaxSize:"700Mi", natsMaxMsgsPerTopic:1000000}
	Config BackendConfig `json:"config,omitempty"`
}

// BackendConfig defines configuration for eventing backend.
type BackendConfig struct {
	// NATSStreamStorageType defines the storage type for stream data.
	// +kubebuilder:default:="File"
	// +kubebuilder:validation:XValidation:rule="self=='File' || self=='Memory'", message="storage type can only be set to File or Memory"
	NATSStreamStorageType string `json:"natsStreamStorageType,omitempty"`

	// NATSStreamReplicas defines the number of replicas for stream.
	// +kubebuilder:default:=3
	NATSStreamReplicas int `json:"natsStreamReplicas,omitempty"`

	// NATSStreamMaxSize defines the maximum storage size for stream data.
	// +kubebuilder:default:="700Mi"
	NATSStreamMaxSize resource.Quantity `json:"natsStreamMaxSize,omitempty"`

	// NATSMaxMsgsPerTopic limits how many messages in the NATS stream to retain per subject.
	// +kubebuilder:default:=1000000
	NATSMaxMsgsPerTopic int `json:"natsMaxMsgsPerTopic,omitempty"`

	// EventMeshSecret defines the namespaced name of K8s Secret containing EventMesh credentials. The format of name is "namespace/name".
	// +kubebuilder:validation:Pattern:="^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"
	EventMeshSecret string `json:"eventMeshSecret,omitempty"`
}

func (ev *Eventing) GetNATSBackend() *Backend {
	for _, backend := range ev.Spec.Backends {
		if backend.Type == NatsBackendType {
			return &backend
		}
	}
	return nil
}

func (ev *Eventing) GetEventMeshBackend() *Backend {
	for _, backend := range ev.Spec.Backends {
		if backend.Type == EventMeshBackendType {
			return &backend
		}
	}
	return nil
}

// Publisher defines the configurations for eventing-publisher-proxy.
type Publisher struct {
	// Replicas defines the scaling min/max for eventing-publisher-proxy.
	// +kubebuilder:default:={min:2,max:2}
	// +kubebuilder:validation:XValidation:rule="self.min <= self.max", message="min value must be smaller than the max value"
	Replicas `json:"replicas,omitempty"`

	// Resources defines resources for eventing-publisher-proxy.
	// +kubebuilder:default:={limits:{cpu:"500m",memory:"512Mi"}, requests:{cpu:"10m",memory:"256Mi"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// Replicas defines min/max replicas for a resource.
type Replicas struct {
	// Min defines minimum number of replicas.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Minimum:=0
	Min int `json:"min,omitempty"`

	// Max defines maximum number of replicas.
	// +kubebuilder:default:=2
	Max int `json:"max,omitempty"`
}

type Logging struct {
	// LogLevel defines the log level.
	// +kubebuilder:default:=Info
	// +kubebuilder:validation:XValidation:rule="self=='Info' || self=='Warn' || self=='Error' || self=='Debug'", message="logLevel can only be set to Debug, Info, Warn or Error"
	LogLevel string `json:"logLevel,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Eventing{}, &EventingList{})
}
