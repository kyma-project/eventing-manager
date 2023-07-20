package matchers

import (
	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func HaveBackendTypeNats(b v1alpha1.Backend) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return e.Spec.Backends.Type
			}, gomega.Equal(b.Type)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) string {
				return e.Spec.Backends.Config.NATSStreamStorageType
			}, gomega.Equal(b.Config.NATSStreamStorageType)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Backends.Config.NATSStreamReplicas
			}, gomega.Equal(b.Config.NATSStreamReplicas)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) resource.Quantity {
				return e.Spec.Backends.Config.NATSStreamMaxSize
			}, gomega.Equal(b.Config.NATSStreamMaxSize)),
		gomega.WithTransform(
			func(e *v1alpha1.Eventing) int {
				return e.Spec.Backends.Config.NATSMaxMsgsPerTopic
			}, gomega.Equal(b.Config.NATSMaxMsgsPerTopic)),
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
