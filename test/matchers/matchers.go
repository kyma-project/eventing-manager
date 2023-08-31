package matchers

import (
	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func HaveFinalizer() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.Eventing) bool {
			return controllerutil.ContainsFinalizer(n, eventing.FinalizerName)
		}, gomega.BeTrue())
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
		Message: eventing.NatsServerNotAvailableMsg,
	})
}

func HaveEventMeshSubManagerReadyCondition() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionSubscriptionManagerReady),
		Status:  metav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonEventMeshSubManagerReady),
		Message: v1alpha1.ConditionSubscriptionManagerReadyMessage,
	})
}

func HaveEventMeshSubManagerNotReadyCondition(message string) gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionSubscriptionManagerReady),
		Status:  metav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonEventMeshSubManagerFailed),
		Message: message,
	})
}

func HaveEventMeshSubManagerStopFailedCondition(message string) gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionSubscriptionManagerReady),
		Status:  metav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonEventMeshSubManagerStopFailed),
		Message: message,
	})
}

func HaveBackendTypeNats(bc v1alpha1.BackendConfig) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return string(e.Spec.Backend.Type)
			}, gomega.Equal("NATS")),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return e.Spec.Backend.Config.NATSStreamStorageType
			}, gomega.Equal(bc.NATSStreamStorageType)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Backend.Config.NATSStreamReplicas
			}, gomega.Equal(bc.NATSStreamReplicas)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) resource.Quantity {
				return e.Spec.Backend.Config.NATSStreamMaxSize
			}, gomega.Equal(bc.NATSStreamMaxSize)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Backend.Config.NATSMaxMsgsPerTopic
			}, gomega.Equal(bc.NATSMaxMsgsPerTopic)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return e.Spec.Backend.Config.EventTypePrefix
			}, gomega.Equal(bc.EventTypePrefix)),
	)
}

func HavePublisher(p v1alpha1.Publisher) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Publisher.Replicas.Min
			}, gomega.Equal(p.Replicas.Min)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Publisher.Replicas.Max
			}, gomega.Equal(p.Replicas.Max)))
}

func HavePublisherResources(res corev1.ResourceRequirements) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) *resource.Quantity {
				return e.Spec.Publisher.Resources.Limits.Cpu()
			}, gomega.Equal(res.Limits.Cpu())),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) *resource.Quantity {
				return e.Spec.Publisher.Resources.Limits.Memory()
			}, gomega.Equal(res.Limits.Memory())),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) *resource.Quantity {
				return e.Spec.Publisher.Resources.Requests.Cpu()
			}, gomega.Equal(res.Requests.Cpu())),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) *resource.Quantity {
				return e.Spec.Publisher.Resources.Requests.Memory()
			}, gomega.Equal(res.Requests.Memory())),
	)
}

func HaveLogging(logging v1alpha1.Logging) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return e.Spec.Logging.LogLevel
			}, gomega.Equal(logging.LogLevel)),
	)
}
