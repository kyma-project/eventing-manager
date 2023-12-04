package eventing

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	kcore "k8s.io/api/core/v1"
	krbac "k8s.io/api/rbac/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	labels map[string]string) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
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

func newPublisherProxyClusterRole(name, namespace string, labels map[string]string) *krbac.ClusterRole {
	// setting `TypeMeta` is important for patch apply to work.
	return &krbac.ClusterRole{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: []krbac.PolicyRule{
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

func newPublisherProxyServiceAccount(name, namespace string, labels map[string]string) *kcore.ServiceAccount {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcore.ServiceAccount{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func newPublisherProxyClusterRoleBinding(name, namespace string, labels map[string]string) *krbac.ClusterRoleBinding {
	// setting `TypeMeta` is important for patch apply to work.
	return &krbac.ClusterRoleBinding{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		RoleRef: krbac.RoleRef{
			Kind:     "ClusterRole",
			Name:     name,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []krbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

func newPublisherProxyService(name, namespace string, labels map[string]string,
	selectorLabels map[string]string) *kcore.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcore.Service{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: kcore.ServiceSpec{
			Selector: selectorLabels,
			Ports: []kcore.ServicePort{
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
	selectorLabels map[string]string) *kcore.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcore.Service{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "9090",
				"prometheus.io/scheme": "http",
			},
		},
		Spec: kcore.ServiceSpec{
			Selector: selectorLabels,
			Ports: []kcore.ServicePort{
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
	selectorLabels map[string]string) *kcore.Service {
	// setting `TypeMeta` is important for patch apply to work.
	return &kcore.Service{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: kcore.ServiceSpec{
			Selector: selectorLabels,
			Ports: []kcore.ServicePort{
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
