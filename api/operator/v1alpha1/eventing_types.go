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

//nolint:lll //this is annotation
package v1alpha1

import (
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionReason string

type ConditionType string

const (
	StateReady      string = "Ready"
	StateError      string = "Error"
	StateProcessing string = "Processing"
	StateWarning    string = "Warning"

	ConditionBackendAvailable         ConditionType = "BackendAvailable"
	ConditionPublisherProxyReady      ConditionType = "PublisherProxyReady"
	ConditionWebhookReady             ConditionType = "WebhookReady"
	ConditionSubscriptionManagerReady ConditionType = "SubscriptionManagerReady"
	ConditionDeleted                  ConditionType = "Deleted"

	// common reasons.
	ConditionReasonProcessing ConditionReason = "Processing"
	ConditionReasonDeleted    ConditionReason = "Deleted"
	ConditionReasonStopped    ConditionReason = "Stopped"

	// publisher proxy reasons.
	ConditionReasonDeployed                   ConditionReason = "Deployed"
	ConditionReasonDeployedFailed             ConditionReason = "DeployFailed"
	ConditionReasonDeploymentStatusSyncFailed ConditionReason = "DeploymentStatusSyncFailed"
	ConditionReasonNATSAvailable              ConditionReason = "NATSAvailable"
	ConditionReasonNATSNotAvailable           ConditionReason = "NATSUnavailable"
	ConditionReasonBackendNotSpecified        ConditionReason = "BackendNotSpecified"
	ConditionReasonForbidden                  ConditionReason = "Forbidden"
	ConditionReasonWebhookFailed              ConditionReason = "WebhookFailed"
	ConditionReasonWebhookReady               ConditionReason = "Ready"
	ConditionReasonDeletionError              ConditionReason = "DeletionError"

	// message for conditions.
	ConditionPublisherProxyReadyMessage        = "Publisher proxy is deployed"
	ConditionPublisherProxyDeletedMessage      = "Publisher proxy is deleted"
	ConditionNATSAvailableMessage              = "NATS is available"
	ConditionWebhookReadyMessage               = "Webhook is available"
	ConditionPublisherProxyProcessingMessage   = "Eventing publisher proxy deployment is in progress"
	ConditionSubscriptionManagerReadyMessage   = "Subscription manager is ready"
	ConditionSubscriptionManagerStoppedMessage = "Subscription manager is stopped"
	ConditionBackendNotSpecifiedMessage        = "Backend config is not provided. Please specify a backend."

	// subscription manager reasons.
	ConditionReasonEventMeshSubManagerReady      ConditionReason = "EventMeshSubscriptionManagerReady"
	ConditionReasonEventMeshSubManagerFailed     ConditionReason = "EventMeshSubscriptionManagerFailed"
	ConditionReasonEventMeshSubManagerStopFailed ConditionReason = "EventMeshSubscriptionManagerStopFailed"
)

// getSupportedConditionsTypes returns a map of supported condition types.
func getSupportedConditionsTypes() map[ConditionType]interface{} {
	return map[ConditionType]interface{}{
		ConditionBackendAvailable:         nil,
		ConditionPublisherProxyReady:      nil,
		ConditionWebhookReady:             nil,
		ConditionSubscriptionManagerReady: nil,
		ConditionDeleted:                  nil,
	}
}

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Eventing is the Schema for the eventing API.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of Eventing"
// +kubebuilder:printcolumn:name="Backend",type="string",JSONPath=".spec.backend.type",description="Type of Eventing backend, either NATS or EventMesh"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the resource"
type Eventing struct {
	kmetav1.TypeMeta   `json:",inline"`
	kmetav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:default:={logging:{logLevel:Info}, publisher:{replicas:{min:2,max:2}, resources:{limits:{cpu:"500m",memory:"512Mi"}, requests:{cpu:"40m",memory:"256Mi"}}}}
	// +kubebuilder:validation:XValidation:rule="!(oldSelf!=null && has(oldSelf.backend)) || has(self.backend)", message="backend config cannot be deleted"
	Spec   EventingSpec   `json:"spec,omitempty"`
	Status EventingStatus `json:"status,omitempty"`
}

// EventingStatus defines the observed state of Eventing.
type EventingStatus struct {
	ActiveBackend     BackendType `json:"activeBackend"`
	BackendConfigHash int64       `json:"specHash"`
	// Can have one of the following values: Ready, Error, Processing, Warning. Ready state is set
	// when all the resources are deployed successfully and backend is connected.
	// It gets Warning state in case backend is not specified, NATS module is not installed or EventMesh secret is missing in the cluster.
	// Error state is set when there is an error. Processing state is set if recources are being created or changed.
	State            string              `json:"state"`
	PublisherService string              `json:"publisherService,omitempty"`
	Conditions       []kmetav1.Condition `json:"conditions,omitempty"`
}

// EventingSpec defines the desired state of Eventing.
type EventingSpec struct {
	// Backend defines the active backend used by Eventing.
	// +kubebuilder:validation:XValidation:rule=" (self.type != 'EventMesh') || ((self.type == 'EventMesh') && (self.config.eventMeshSecret != ''))", message="secret cannot be empty if EventMesh backend is used"
	Backend *Backend `json:"backend,omitempty"`

	// Publisher defines the configurations for eventing-publisher-proxy.
	// +kubebuilder:default:={replicas:{min:2,max:2}, resources:{limits:{cpu:"500m",memory:"512Mi"}, requests:{cpu:"40m",memory:"256Mi"}}}
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

// EventingList contains a list of Eventing.
type EventingList struct {
	kmetav1.TypeMeta `json:",inline"`
	kmetav1.ListMeta `json:"metadata,omitempty"`
	Items            []Eventing `json:"items"`
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

	// +kubebuilder:default:="sap.kyma.custom"
	// +kubebuilder:validation:XValidation:rule="self!=''", message="eventTypePrefix cannot be empty"
	EventTypePrefix string `json:"eventTypePrefix,omitempty"`

	// Domain defines the cluster public domain used to configure the EventMesh Subscriptions
	// and their corresponding ApiRules.
	// +kubebuilder:validation:Pattern:="^(?:([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9])(\\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9]))*)?$"
	Domain string `json:"domain,omitempty"`
}

// Publisher defines the configurations for eventing-publisher-proxy.
type Publisher struct {
	// Replicas defines the scaling min/max for eventing-publisher-proxy.
	// +kubebuilder:default:={min:2,max:2}
	// +kubebuilder:validation:XValidation:rule="self.min <= self.max", message="min value must be smaller than the max value"
	Replicas `json:"replicas,omitempty"`

	// Resources defines resources for eventing-publisher-proxy.
	// +kubebuilder:default:={limits:{cpu:"500m",memory:"512Mi"}, requests:{cpu:"40m",memory:"256Mi"}}
	Resources kcorev1.ResourceRequirements `json:"resources,omitempty"`
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

//nolint:gochecknoinits // registers Eventing CRD at startup
func init() {
	SchemeBuilder.Register(&Eventing{}, &EventingList{})
}

func (e *Eventing) SyncStatusActiveBackend() {
	e.Status.ActiveBackend = e.Spec.Backend.Type
}

func (e *Eventing) IsPreviousBackendEmpty() bool {
	return e.Status.ActiveBackend == ""
}

func (e *Eventing) IsSpecBackendTypeChanged() bool {
	return e.Status.ActiveBackend != e.Spec.Backend.Type
}
