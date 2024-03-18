package matchers

import (
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/controller/operator/eventing"
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

func HaveStatusWarning() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.Eventing) string {
			return n.Status.State
		}, gomega.Equal(v1alpha1.StateWarning))
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

func HaveCondition(condition kmetav1.Condition) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.Eventing) []kmetav1.Condition {
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
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionPublisherProxyReady),
		Status:  kmetav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonDeployed),
		Message: v1alpha1.ConditionPublisherProxyReadyMessage,
	})
}

func HavePublisherProxyReadyConditionProcessing() gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionPublisherProxyReady),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonProcessing),
		Message: v1alpha1.ConditionPublisherProxyProcessingMessage,
	})
}

func HavePublisherProxyConditionForbiddenWithMsg(msg string) gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionPublisherProxyReady),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonForbidden),
		Message: msg,
	})
}

func HaveNATSAvailableCondition() gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionBackendAvailable),
		Status:  kmetav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonNATSAvailable),
		Message: v1alpha1.ConditionNATSAvailableMessage,
	})
}

func HaveBackendNotAvailableConditionWith(message string, reason v1alpha1.ConditionReason) gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionBackendAvailable),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(reason),
		Message: message,
	})
}

func HaveBackendAvailableConditionWith(message string, reason v1alpha1.ConditionReason) gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionBackendAvailable),
		Status:  kmetav1.ConditionTrue,
		Reason:  string(reason),
		Message: message,
	})
}

func HaveNATSNotAvailableConditionWith(message string) gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionBackendAvailable),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonNATSNotAvailable),
		Message: message,
	})
}

func HaveNATSNotAvailableCondition() gomegatypes.GomegaMatcher {
	return HaveNATSNotAvailableConditionWith(eventing.NatsServerNotAvailableMsg)
}

func HaveEventMeshSubManagerReadyCondition() gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionSubscriptionManagerReady),
		Status:  kmetav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonEventMeshSubManagerReady),
		Message: v1alpha1.ConditionSubscriptionManagerReadyMessage,
	})
}

func HaveEventMeshSubManagerNotReadyCondition(message string) gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionSubscriptionManagerReady),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonEventMeshSubManagerFailed),
		Message: message,
	})
}

func HaveDeletionErrorCondition(message string) gomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(v1alpha1.ConditionDeleted),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonDeletionError),
		Message: message,
	})
}

func HaveBackendTypeNats(config v1alpha1.BackendConfig) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return string(e.Spec.Backend.Type)
			}, gomega.Equal("NATS")),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return e.Spec.Backend.Config.NATSStreamStorageType
			}, gomega.Equal(config.NATSStreamStorageType)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Backend.Config.NATSStreamReplicas
			}, gomega.Equal(config.NATSStreamReplicas)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) resource.Quantity {
				return e.Spec.Backend.Config.NATSStreamMaxSize
			}, gomega.Equal(config.NATSStreamMaxSize)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Backend.Config.NATSMaxMsgsPerTopic
			}, gomega.Equal(config.NATSMaxMsgsPerTopic)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return e.Spec.Backend.Config.EventTypePrefix
			}, gomega.Equal(config.EventTypePrefix)),
	)
}

func HavePublisher(publisher v1alpha1.Publisher) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Publisher.Replicas.Min
			}, gomega.Equal(publisher.Replicas.Min)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Publisher.Replicas.Max
			}, gomega.Equal(publisher.Replicas.Max)))
}

func HavePublisherResources(res kcorev1.ResourceRequirements) gomegatypes.GomegaMatcher {
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
