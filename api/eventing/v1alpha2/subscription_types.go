package v1alpha2

import (
	"encoding/json"
	"strconv"

	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/utils"
)

type TypeMatching string

//nolint:gochecknoglobals // required for consistency
var Finalizer = GroupVersion.Group

// Defines the desired state of the Subscription.
type SubscriptionSpec struct {
	// Unique identifier of the Subscription, read-only.
	// +optional
	ID string `json:"id,omitempty"`

	// Kubernetes Service that should be used as a target for the events that match the Subscription.
	// Must exist in the same Namespace as the Subscription.
	Sink string `json:"sink"`

	// Defines how types should be handled.<br />
	// - `standard`: backend-specific logic will be applied to the configured source and types.<br />
	// - `exact`: no further processing will be applied to the configured source and types.
	// +kubebuilder:default:="standard"
	// +kubebuilder:validation:XValidation:rule="self=='standard' || self=='exact'", message="typeMatching can only be set to standard or exact"
	TypeMatching TypeMatching `json:"typeMatching,omitempty"`

	// Defines the origin of the event.
	Source string `json:"source"`

	// List of event types that will be used for subscribing on the backend.
	Types []string `json:"types"`

	// Map of configuration options that will be applied on the backend.
	// +optional
	// +kubebuilder:default={"maxInFlightMessages":"10"}
	Config map[string]string `json:"config,omitempty"`
}

// SubscriptionStatus defines the observed state of Subscription.
// +kubebuilder:subresource:status
type SubscriptionStatus struct {
	// Current state of the Subscription.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`

	// Overall readiness of the Subscription.
	Ready bool `json:"ready"`

	// List of event types after cleanup for use with the configured backend.
	Types []EventType `json:"types"`

	// Backend-specific status which is applicable to the active backend only.
	Backend Backend `json:"backend,omitempty"`
}

// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Subscription is the Schema for the subscriptions API.
type Subscription struct {
	kmetav1.TypeMeta   `json:",inline"`
	kmetav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubscriptionSpec   `json:"spec,omitempty"`
	Status SubscriptionStatus `json:"status,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// If the SubscriptionStatus.CleanEventTypes is nil, it will be initialized to an empty slice of stings.
// It is needed because the Kubernetes APIServer will reject requests containing null in the JSON payload.
func (s Subscription) MarshalJSON() ([]byte, error) {
	// Use type alias to copy the subscription without causing an infinite recursion when calling json.Marshal.
	type Alias Subscription
	a := Alias(s)
	if a.Status.Types == nil {
		a.Status.InitializeEventTypes()
	}
	return json.Marshal(a)
}

// GetMaxInFlightMessages tries to convert the string-type maxInFlight to the integer.
func (s *Subscription) GetMaxInFlightMessages(defaults *env.DefaultSubscriptionConfig) int {
	val, err := strconv.Atoi(s.Spec.Config[MaxInFlightMessages])
	if err != nil {
		return defaults.MaxInFlightMessages
	}
	return val
}

// InitializeEventTypes initializes the SubscriptionStatus.Types with an empty slice of EventType.
func (s *SubscriptionStatus) InitializeEventTypes() {
	s.Types = []EventType{}
}

// GetUniqueTypes returns the de-duplicated types from subscription spec.
func (s *Subscription) GetUniqueTypes() []string {
	result := make([]string, 0, len(s.Spec.Types))
	for _, t := range s.Spec.Types {
		if !utils.ContainsString(result, t) {
			result = append(result, t)
		}
	}

	return result
}

// GetDuplicateTypes returns the duplicate types from the Subscription spec.
func (s *Subscription) GetDuplicateTypes() []string {
	if len(s.Spec.Types) == 0 {
		return s.Spec.Types
	}
	const duplicatesCount = 2
	types := make(map[string]int, len(s.Spec.Types))
	duplicates := make([]string, 0, len(s.Spec.Types))
	for _, t := range s.Spec.Types {
		if types[t]++; types[t] == duplicatesCount {
			duplicates = append(duplicates, t)
		}
	}
	return duplicates
}

func (s *Subscription) DuplicateWithStatusDefaults() *Subscription {
	desiredSub := s.DeepCopy()
	desiredSub.Status = SubscriptionStatus{}
	return desiredSub
}

func (s *Subscription) ToUnstructuredSub() (*kunstructured.Unstructured, error) {
	object, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(&s)
	if err != nil {
		return nil, err
	}
	return &kunstructured.Unstructured{Object: object}, nil
}

// +kubebuilder:object:root=true

// SubscriptionList contains a list of Subscription.
type SubscriptionList struct {
	kmetav1.TypeMeta `json:",inline"`
	kmetav1.ListMeta `json:"metadata,omitempty"`
	Items            []Subscription `json:"items"`
}

func init() { //nolint:gochecknoinits
	SchemeBuilder.Register(&Subscription{}, &SubscriptionList{})
}
