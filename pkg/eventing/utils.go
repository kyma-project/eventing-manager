package eventing

import (
	"fmt"

	kautoscalingv2 "k8s.io/api/autoscaling/v2"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

const publisherProxySuffix = "publisher-proxy"

func GetPublisherDeploymentName(eventing v1alpha1.Eventing) string {
	return fmt.Sprintf("%s-%s", eventing.GetName(), publisherProxySuffix)
}

func GetPublisherPublishServiceName(eventing v1alpha1.Eventing) string {
	return fmt.Sprintf("%s-%s", eventing.GetName(), publisherProxySuffix)
}

func GetPublisherMetricsServiceName(eventing v1alpha1.Eventing) string {
	return fmt.Sprintf("%s-%s-metrics", eventing.GetName(), publisherProxySuffix)
}

func GetPublisherHealthServiceName(eventing v1alpha1.Eventing) string {
	return fmt.Sprintf("%s-%s-health", eventing.GetName(), publisherProxySuffix)
}

func GetPublisherServiceAccountName(eventing v1alpha1.Eventing) string {
	return fmt.Sprintf("%s-%s", eventing.GetName(), publisherProxySuffix)
}

func GetPublisherClusterRoleName(eventing v1alpha1.Eventing) string {
	return fmt.Sprintf("%s-%s", eventing.GetName(), publisherProxySuffix)
}

func GetPublisherClusterRoleBindingName(eventing v1alpha1.Eventing) string {
	return fmt.Sprintf("%s-%s", eventing.GetName(), publisherProxySuffix)
}

func newHorizontalPodAutoscaler(name, namespace string, min, max, cpuUtilization, memoryUtilization int32,
	labels map[string]string,
) *kautoscalingv2.HorizontalPodAutoscaler {
	return &kautoscalingv2.HorizontalPodAutoscaler{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: kautoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       name,
				APIVersion: "apps/v1",
			},
			MinReplicas: &min,
			MaxReplicas: max,
			Metrics: []kautoscalingv2.MetricSpec{
				{
					Type: kautoscalingv2.ResourceMetricSourceType,
					Resource: &kautoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: kautoscalingv2.MetricTarget{
							Type:               kautoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpuUtilization,
						},
					},
				},
				{
					Type: kautoscalingv2.ResourceMetricSourceType,
					Resource: &kautoscalingv2.ResourceMetricSource{
						Name: "memory",
						Target: kautoscalingv2.MetricTarget{
							Type:               kautoscalingv2.UtilizationMetricType,
							AverageUtilization: &memoryUtilization,
						},
					},
				},
			},
		},
	}
}

func newPublisherProxyClusterRole(name, namespace string, labels map[string]string) *krbacv1.ClusterRole {
	// setting `TypeMeta` is important for patch apply to work.
	return &krbacv1.ClusterRole{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: []krbacv1.PolicyRule{
			{
				APIGroups: []string{"eventing.kyma-project.io"},
				Resources: []string{"subscriptions"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"applicationconnector.kyma-project.io"},
				Resources: []string{"applications"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func newPublisherProxyServiceAccount(name, namespace string, labels map[string]string) *kcorev1.ServiceAccount {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcorev1.ServiceAccount{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func newPublisherProxyClusterRoleBinding(name, namespace string, labels map[string]string) *krbacv1.ClusterRoleBinding {
	// setting `TypeMeta` is important for patch apply to work.
	return &krbacv1.ClusterRoleBinding{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		RoleRef: krbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     name,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []krbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

func newPublisherProxyService(name, namespace string, labels map[string]string,
	selectorLabels map[string]string,
) *kcorev1.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcorev1.Service{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: kcorev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []kcorev1.ServicePort{
				{
					Name:       "http-client",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
}

func newPublisherProxyMetricsService(name, namespace string, labels map[string]string,
	selectorLabels map[string]string,
) *kcorev1.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcorev1.Service{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "9090",
				"prometheus.io/scheme": "http",
			},
		},
		Spec: kcorev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []kcorev1.ServicePort{
				{
					Name:       "http-metrics",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromInt(9090),
				},
			},
		},
	}
}

func newPublisherProxyHealthService(name, namespace string, labels map[string]string,
	selectorLabels map[string]string,
) *kcorev1.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcorev1.Service{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: kcorev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []kcorev1.ServicePort{
				{
					Name:       "http-status",
					Protocol:   "TCP",
					Port:       15020,
					TargetPort: intstr.FromInt(15020),
				},
			},
		},
	}
}

func getECBackendType(backendType v1alpha1.BackendType) v1alpha1.BackendType {
	if backendType == v1alpha1.EventMeshBackendType {
		return v1alpha1.EventMeshBackendType
	}
	return v1alpha1.NatsBackendType
}
