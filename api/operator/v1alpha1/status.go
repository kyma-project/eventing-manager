package v1alpha1

import (
	"reflect"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (es *EventingStatus) UpdateConditionNATSAvailable(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionNATSAvailable),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
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
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) UpdateConditionWebhookReady(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionWebhookReady),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (sm *EventingStatus) UpdateConditionSubscriptionManagerReady(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionSubscriptionManagerReady),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&sm.Conditions, condition)
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
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) SetSubscriptionManagerReadyConditionToTrue() {
	es.UpdateConditionSubscriptionManagerReady(kmetav1.ConditionTrue, ConditionReasonEventMeshSubManagerReady,
		ConditionSubscriptionManagerReadyMessage)
}

func (es *EventingStatus) SetStateReady() {
	es.State = StateReady
	es.UpdateConditionNATSAvailable(kmetav1.ConditionTrue, ConditionReasonNATSAvailable, ConditionNATSAvailableMessage)
	es.UpdateConditionPublisherProxyReady(kmetav1.ConditionTrue, ConditionReasonDeployed, ConditionPublisherProxyReadyMessage)
}

func (ns *EventingStatus) SetStateWarning() {
	ns.State = StateWarning
}

func (es *EventingStatus) SetNATSAvailableConditionToTrue() {
	es.UpdateConditionNATSAvailable(kmetav1.ConditionTrue, ConditionReasonNATSAvailable, ConditionNATSAvailableMessage)
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

func (es *EventingStatus) SetWebhookReadyConditionToTrue() {
	es.UpdateConditionWebhookReady(kmetav1.ConditionTrue, ConditionReasonWebhookReady, ConditionWebhookReadyMessage)
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
