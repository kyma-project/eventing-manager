package matchers

import (
	"fmt"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HaveStatusReady() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.Eventing) string {
			return n.Status.State
		}, gomega.Equal(v1alpha1.StateReady))
}

func HaveStatusProcessing() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.Eventing) string {
			return n.Status.State
		}, gomega.Equal(v1alpha1.StateProcessing))
}

func HaveStatusError() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.Eventing) string {
			return n.Status.State
		}, gomega.Equal(v1alpha1.StateError))
}

func HaveCondition(condition metav1.Condition) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.Eventing) []metav1.Condition {
			return n.Status.Conditions
		},
		gomega.ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras|gstruct.IgnoreMissing, gstruct.Fields{
			"Type":    gomega.Equal(condition.Type),
			"Reason":  gomega.Equal(condition.Reason),
			"Message": gomega.Equal(condition.Message),
			"Status":  gomega.Equal(condition.Status),
		})))
}

func HavePublisherProxyReadyConditionDeployed() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionPublisherProxyReady),
		Status:  metav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonDeployed),
		Message: v1alpha1.ConditionPublisherProxyReadyMessage,
	})
}

func HavePublisherProxyReadyConditionProcessing() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionPublisherProxyReady),
		Status:  metav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonProcessing),
		Message: v1alpha1.ConditionPublisherProxyProcessingMessage,
	})
}

func HaveNATSAvailableConditionAvailable() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionNATSAvailable),
		Status:  metav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonNATSAvailable),
		Message: v1alpha1.ConditionNATSAvailableMessage,
	})
}

func HaveNATSAvailableConditionNotAvailable() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionNATSAvailable),
		Status:  metav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonNATSNotAvailable),
		Message: fmt.Sprintf(eventing.NatsServerNotAvailableMsg, "kyma-system"),
	})
}
