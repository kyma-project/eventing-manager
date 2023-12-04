package v1alpha1

import (
	"fmt"

	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	ConditionSubscribed         ConditionType = "Subscribed"
	ConditionSubscriptionActive ConditionType = "Subscription active"
	ConditionAPIRuleStatus      ConditionType = "APIRule status"
	ConditionWebhookCallStatus  ConditionType = "Webhook call status"

	ConditionPublisherProxyReady ConditionType = "Publisher Proxy Ready"
	ConditionControllerReady     ConditionType = "Subscription Controller Ready"
)

var allSubscriptionConditions = MakeSubscriptionConditions()

type Condition struct {
	// Short description of the condition.
	Type ConditionType `json:"type,omitempty"`
	// Status of the condition. The value is either `True`, `False`, or `Unknown`.
	Status kcorev1.ConditionStatus `json:"status"`
	// Defines the date of the last condition status change.
	LastTransitionTime kmetav1.Time `json:"lastTransitionTime,omitempty"`
	// Defines the reason for the condition status change.
	Reason ConditionReason `json:"reason,omitempty"`
	// Provides more details about the condition status change.
	Message string `json:"message,omitempty"`
}

type ConditionReason string

const (
	// BEB Conditions.
	ConditionReasonSubscriptionActive    ConditionReason = "BEB Subscription active"
	ConditionReasonSubscriptionNotActive ConditionReason = "BEB Subscription not active"
	ConditionReasonSubscriptionDeleted   ConditionReason = "BEB Subscription deleted"
	ConditionReasonAPIRuleStatusReady    ConditionReason = "APIRule status ready"
	ConditionReasonAPIRuleStatusNotReady ConditionReason = "APIRule status not ready"

	// Common backend Conditions.
	ConditionReasonSubscriptionControllerReady    ConditionReason = "Subscription controller started"
	ConditionReasonSubscriptionControllerNotReady ConditionReason = "Subscription controller not ready"
	ConditionReasonPublisherDeploymentReady       ConditionReason = "Publisher proxy deployment ready"
	ConditionReasonPublisherDeploymentNotReady    ConditionReason = "Publisher proxy deployment not ready"
)

// initializeConditions sets unset conditions to Unknown.
func initializeConditions(initialConditions, currentConditions []Condition) []Condition {
	givenConditions := make(map[ConditionType]Condition)

	// create map of Condition per ConditionType
	for _, condition := range currentConditions {
		givenConditions[condition.Type] = condition
	}

	finalConditions := currentConditions
	// check if every Condition is present in the current Conditions
	for _, expectedCondition := range initialConditions {
		if _, ok := givenConditions[expectedCondition.Type]; !ok {
			// and add it if it is missing
			finalConditions = append(finalConditions, expectedCondition)
		}
	}
	return finalConditions
}

// InitializeConditions sets unset Subscription conditions to Unknown.
func (s *SubscriptionStatus) InitializeConditions() {
	initialConditions := MakeSubscriptionConditions()
	s.Conditions = initializeConditions(initialConditions, s.Conditions)
}

func (s SubscriptionStatus) IsReady() bool {
	if !ContainSameConditionTypes(allSubscriptionConditions, s.Conditions) {
		return false
	}

	// the subscription is ready if all its conditions are evaluated to true
	for _, c := range s.Conditions {
		if c.Status != kcorev1.ConditionTrue {
			return false
		}
	}
	return true
}

func (s SubscriptionStatus) FindCondition(conditionType ConditionType) *Condition {
	for _, condition := range s.Conditions {
		if conditionType == condition.Type {
			return &condition
		}
	}
	return nil
}

// ShouldUpdateReadyStatus checks if there is a mismatch between the
// subscription Ready Status and the Ready status of all the conditions.
func (s SubscriptionStatus) ShouldUpdateReadyStatus() bool {
	if !s.Ready && s.IsReady() || s.Ready && !s.IsReady() {
		return true
	}
	return false
}

// MakeSubscriptionConditions creates a map of all conditions which the Subscription should have.
func MakeSubscriptionConditions() []Condition {
	conditions := []Condition{
		{
			Type:               ConditionAPIRuleStatus,
			LastTransitionTime: kmetav1.Now(),
			Status:             kcorev1.ConditionUnknown,
		},
		{
			Type:               ConditionSubscribed,
			LastTransitionTime: kmetav1.Now(),
			Status:             kcorev1.ConditionUnknown,
		},
		{
			Type:               ConditionSubscriptionActive,
			LastTransitionTime: kmetav1.Now(),
			Status:             kcorev1.ConditionUnknown,
		},
		{
			Type:               ConditionWebhookCallStatus,
			LastTransitionTime: kmetav1.Now(),
			Status:             kcorev1.ConditionUnknown,
		},
	}
	return conditions
}

func ContainSameConditionTypes(conditions1, conditions2 []Condition) bool {
	if len(conditions1) != len(conditions2) {
		return false
	}

	for _, condition := range conditions1 {
		if !containConditionType(conditions2, condition.Type) {
			return false
		}
	}

	return true
}

func containConditionType(conditions []Condition, conditionType ConditionType) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return true
		}
	}

	return false
}

func MakeCondition(conditionType ConditionType, reason ConditionReason, status kcorev1.ConditionStatus, message string) Condition {
	return Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             reason,
		// TODO: https://github.com/kyma-project/kyma/issues/9770
		Message: message,
	}
}

func (s *SubscriptionStatus) IsConditionSubscribed() bool {
	for _, condition := range s.Conditions {
		if condition.Type == ConditionSubscribed && condition.Status == kcorev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (s *SubscriptionStatus) IsConditionWebhookCall() bool {
	for _, condition := range s.Conditions {
		if condition.Type == ConditionWebhookCallStatus &&
			(condition.Status == kcorev1.ConditionTrue || condition.Status == kcorev1.ConditionUnknown) {
			return true
		}
	}
	return false
}

func (s *SubscriptionStatus) GetConditionAPIRuleStatus() kcorev1.ConditionStatus {
	for _, condition := range s.Conditions {
		if condition.Type == ConditionAPIRuleStatus {
			return condition.Status
		}
	}
	return kcorev1.ConditionUnknown
}

func (s *SubscriptionStatus) SetConditionAPIRuleStatus(err error) {
	reason := ConditionReasonAPIRuleStatusReady
	status := kcorev1.ConditionTrue
	message := ""
	if err != nil {
		reason = ConditionReasonAPIRuleStatusNotReady
		status = kcorev1.ConditionFalse
		message = err.Error()
	}

	newConditions := []Condition{MakeCondition(ConditionAPIRuleStatus, reason, status, message)}
	for _, condition := range s.Conditions {
		if condition.Type == ConditionAPIRuleStatus {
			continue
		}
		newConditions = append(newConditions, condition)
	}
	s.Conditions = newConditions
}

func CreateMessageForConditionReasonSubscriptionCreated(bebName string) string {
	return fmt.Sprintf("BEB-subscription-name=%s", bebName)
}

// ConditionsEquals checks if two list of conditions are equal.
func ConditionsEquals(existing, expected []Condition) bool {
	// not equal if length is different
	if len(existing) != len(expected) {
		return false
	}

	// compile map of Conditions per ConditionType
	existingMap := make(map[ConditionType]Condition, len(existing))
	for _, value := range existing {
		existingMap[value.Type] = value
	}

	for _, value := range expected {
		if !ConditionEquals(existingMap[value.Type], value) {
			return false
		}
	}

	return true
}

// ConditionEquals checks if two conditions are equal.
func ConditionEquals(existing, expected Condition) bool {
	isTypeEqual := existing.Type == expected.Type
	isStatusEqual := existing.Status == expected.Status
	isReasonEqual := existing.Reason == expected.Reason
	isMessageEqual := existing.Message == expected.Message

	if !isStatusEqual || !isReasonEqual || !isMessageEqual || !isTypeEqual {
		return false
	}

	return true
}
