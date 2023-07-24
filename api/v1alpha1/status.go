package v1alpha1

import (
	"reflect"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (es *EventingStatus) UpdateConditionNATSAvailable(status metav1.ConditionStatus, reason ConditionReason,
	message string) {
	condition := metav1.Condition{
		Type:               string(ConditionNATSAvailable),
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) UpdateConditionPublisherProxyReady(status metav1.ConditionStatus, reason ConditionReason,
	message string) {
	condition := metav1.Condition{
		Type:               string(ConditionPublisherProxyReady),
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&es.Conditions, condition)
}

func (es *EventingStatus) SetStateReady() {
	es.State = StateReady
	es.UpdateConditionNATSAvailable(metav1.ConditionTrue, ConditionReasonNATSAvailable, ConditionNATSAvailableMessage)
	es.UpdateConditionPublisherProxyReady(metav1.ConditionTrue, ConditionReasonDeployed, ConditionPublisherProxyReadyMessage)
}

func (es *EventingStatus) SetNATSAvailableConditionToTrue() {
	es.UpdateConditionNATSAvailable(metav1.ConditionTrue, ConditionReasonNATSAvailable, ConditionNATSAvailableMessage)
}

func (es *EventingStatus) SetPublisherProxyReadyToTrue() {
	es.State = StateReady
	es.UpdateConditionPublisherProxyReady(metav1.ConditionTrue, ConditionReasonDeployed, ConditionPublisherProxyReadyMessage)
}

func (es *EventingStatus) SetStateProcessing() {
	es.State = StateProcessing
}

func (es *EventingStatus) SetStateError() {
	es.State = StateError
}

func (es *EventingStatus) IsEqual(status EventingStatus) bool {
	thisWithoutCond := es.DeepCopy()
	statusWithoutCond := status.DeepCopy()

	// remove conditions, so that we don't compare them
	thisWithoutCond.Conditions = []metav1.Condition{}
	statusWithoutCond.Conditions = []metav1.Condition{}

	return reflect.DeepEqual(thisWithoutCond, statusWithoutCond) &&
		natsv1alpha1.ConditionsEquals(es.Conditions, status.Conditions)
}
