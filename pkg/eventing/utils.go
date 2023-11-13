package eventing

import (
	"fmt"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func newHorizontalPodAutoscaler(name, namespace string, min int32, max int32, cpuUtilization, memoryUtilization int32) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       name,
				APIVersion: "apps/v1",
			},
			MinReplicas: &min,
			MaxReplicas: max,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpuUtilization,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "memory",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &memoryUtilization,
						},
					},
				},
			},
		},
	}
}

func newPublisherProxyClusterRole(name, namespace string, labels map[string]string) *rbacv1.ClusterRole {
	// setting `TypeMeta` is important for patch apply to work.
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: []rbacv1.PolicyRule{
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

func newPublisherProxyServiceAccount(name, namespace string, labels map[string]string) *corev1.ServiceAccount {
	// setting `TypeMeta` is important for patch apply to work.
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func newPublisherProxyClusterRoleBinding(name, namespace string, labels map[string]string) *rbacv1.ClusterRoleBinding {
	// setting `TypeMeta` is important for patch apply to work.
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     name,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

func newPublisherProxyService(name, namespace string, labels map[string]string,
	selectorLabels map[string]string) *corev1.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
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
	selectorLabels map[string]string) *corev1.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "9090",
				"prometheus.io/scheme": "http",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
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
	selectorLabels map[string]string) *corev1.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
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

func getECBackendType(backendType v1alpha1.BackendType) ecv1alpha1.BackendType {
	if backendType == v1alpha1.EventMeshBackendType {
		return ecv1alpha1.BEBBackendType
	}
	return ecv1alpha1.NatsBackendType
}
