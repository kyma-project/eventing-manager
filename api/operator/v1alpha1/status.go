package v1alpha1

import (
	"fmt"
	"reflect"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (es *EventingStatus) UpdateConditionBackendAvailable(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionBackendAvailable),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	// meta.SetStatusCondition will update LastTransitionTime only when `Status` is changed.
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) UpdateConditionPublisherProxyReady(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionPublisherProxyReady),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	// LastTransitionTime will only be updated when `Status` is changed.
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) UpdateConditionSubscriptionManagerReady(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionSubscriptionManagerReady),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	// meta.SetStatusCondition will update LastTransitionTime only when `Status` is changed.
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) UpdateConditionDeletion(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionDeleted),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	// meta.SetStatusCondition will update LastTransitionTime only when `Status` is changed.
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) SetSubscriptionManagerReadyConditionToTrue() {
	es.UpdateConditionSubscriptionManagerReady(kmetav1.ConditionTrue, ConditionReasonEventMeshSubManagerReady,
		ConditionSubscriptionManagerReadyMessage)
}

func (es *EventingStatus) SetStateReady() {
	es.State = StateReady
	es.UpdateConditionBackendAvailable(kmetav1.ConditionTrue, ConditionReasonNATSAvailable, ConditionNATSAvailableMessage)
	es.UpdateConditionPublisherProxyReady(kmetav1.ConditionTrue, ConditionReasonDeployed, ConditionPublisherProxyReadyMessage)
}

func (es *EventingStatus) SetStateWarning() {
	es.State = StateWarning
}

func (es *EventingStatus) SetNATSAvailableConditionToTrue() {
	es.UpdateConditionBackendAvailable(kmetav1.ConditionTrue, ConditionReasonNATSAvailable, ConditionNATSAvailableMessage)
}

func (es *EventingStatus) SetEventMeshAvailableConditionToTrue() {
	es.UpdateConditionBackendAvailable(
		kmetav1.ConditionTrue, ConditionReasonEventMeshConfigAvailable, ConditionEventMeshConfigAvailableMessage,
	)
}

func (es *EventingStatus) SetSubscriptionManagerReadyConditionToFalse(reason ConditionReason, message string) {
	es.UpdateConditionSubscriptionManagerReady(kmetav1.ConditionFalse, reason,
		message)
}

func (es *EventingStatus) SetPublisherProxyConditionToFalse(reason ConditionReason, message string) {
	es.UpdateConditionPublisherProxyReady(kmetav1.ConditionFalse, reason,
		message)
}

func (es *EventingStatus) SetPublisherProxyReadyToTrue() {
	es.State = StateReady
	es.UpdateConditionPublisherProxyReady(kmetav1.ConditionTrue, ConditionReasonDeployed, ConditionPublisherProxyReadyMessage)
}

func (es *EventingStatus) SetStateProcessing() {
	es.State = StateProcessing
}

func (es *EventingStatus) SetStateError() {
	es.State = StateError
}

func (es *EventingStatus) ClearConditions() {
	es.Conditions = []kmetav1.Condition{}
}

func (es *EventingStatus) IsEqual(status EventingStatus) bool {
	thisWithoutCond := es.DeepCopy()
	statusWithoutCond := status.DeepCopy()

	// remove conditions, so that we don't compare them
	thisWithoutCond.Conditions = []kmetav1.Condition{}
	statusWithoutCond.Conditions = []kmetav1.Condition{}

	return reflect.DeepEqual(thisWithoutCond, statusWithoutCond) &&
		natsv1alpha1.ConditionsEquals(es.Conditions, status.Conditions)
}

// ClearPublisherService clears the PublisherService.
func (es *EventingStatus) ClearPublisherService() {
	es.PublisherService = ""
}

// SetPublisherService sets the PublisherService from the given service name and namespace.
func (es *EventingStatus) SetPublisherService(name, namespace string) {
	es.PublisherService = fmt.Sprintf("%s.%s", name, namespace)
}

// RemoveUnsupportedConditions removes unsupported conditions from the status and keeps only the supported ones.
func (es *EventingStatus) RemoveUnsupportedConditions() {
	if len(es.Conditions) == 0 {
		return
	}

	supportedConditionsTypes := getSupportedConditionsTypes()
	supportedConditions := make([]kmetav1.Condition, 0, len(es.Conditions))
	for _, condition := range es.Conditions {
		if _, ok := supportedConditionsTypes[ConditionType(condition.Type)]; ok {
			supportedConditions = append(supportedConditions, condition)
		}
	}
	es.Conditions = supportedConditions
}
