package utils

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	charset       = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomNameLen = 5

	NameFormat           = "name-%s"
	NamespaceFormat      = "namespace-%s"
	PublisherProxySuffix = "publisher-proxy"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec,gochecknoglobals // used in tests

func GetRandString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type EventingOption func(*v1alpha1.Eventing) error

func NewNamespace(name string) *v1.Namespace {
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

func NewEventingCR(opts ...EventingOption) *v1alpha1.Eventing {
	name := fmt.Sprintf(NameFormat, GetRandString(randomNameLen))
	namespace := fmt.Sprintf(NamespaceFormat, GetRandString(randomNameLen))

	eventing := &v1alpha1.Eventing{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Eventing",
			APIVersion: "operator.kyma-project.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "1234-5678-1234-5678",
		},
		Spec: v1alpha1.EventingSpec{
			Backends: []v1alpha1.Backend{
				{
					Type: v1alpha1.NatsBackendType,
				},
			},
		},
	}

	for _, opt := range opts {
		if err := opt(eventing); err != nil {
			panic(err)
		}
	}

	return eventing
}

func HasOwnerReference(object client.Object, eventingCR v1alpha1.Eventing) bool {
	ownerReferences := object.GetOwnerReferences()

	return len(ownerReferences) > 0 && ownerReferences[0].Name == eventingCR.Name &&
		ownerReferences[0].Kind == "Eventing" &&
		ownerReferences[0].UID == eventingCR.UID
}

func IsEPPPublishServiceCorrect(svc v1.Service, eppDeployment appsv1.Deployment) bool {
	wantSpec := v1.ServiceSpec{
		Selector: eppDeployment.Spec.Template.Labels,
		Ports: []v1.ServicePort{
			{
				Name:       "http-client",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			},
		},
	}

	return reflect.DeepEqual(wantSpec.Selector, svc.Spec.Selector) &&
		reflect.DeepEqual(wantSpec.Ports, svc.Spec.Ports)
}

func IsEPPMetricsServiceCorrect(svc v1.Service, eppDeployment appsv1.Deployment) bool {
	wantAnnotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "9090",
		"prometheus.io/scheme": "http",
	}

	wantSpec := v1.ServiceSpec{
		Selector: eppDeployment.Spec.Template.Labels,
		Ports: []v1.ServicePort{
			{
				Name:       "http-metrics",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(9090),
			},
		},
	}

	return reflect.DeepEqual(wantSpec.Selector, svc.Spec.Selector) &&
		reflect.DeepEqual(wantSpec.Ports, svc.Spec.Ports) &&
		reflect.DeepEqual(wantAnnotations, svc.Annotations)
}

func IsEPPHealthServiceCorrect(svc v1.Service, eppDeployment appsv1.Deployment) bool {
	wantSpec := v1.ServiceSpec{
		Selector: eppDeployment.Spec.Template.Labels,
		Ports: []v1.ServicePort{
			{
				Name:       "http-status",
				Protocol:   "TCP",
				Port:       15020,
				TargetPort: intstr.FromInt(15020),
			},
		},
	}

	return reflect.DeepEqual(wantSpec.Selector, svc.Spec.Selector) &&
		reflect.DeepEqual(wantSpec.Ports, svc.Spec.Ports)
}

func IsEPPClusterRoleCorrect(clusterRole rbacv1.ClusterRole) bool {
	wantRules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"eventing.kyma-project.io"},
			Resources: []string{"subscriptions"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"operator.kyma-project.io"},
			Resources: []string{"subscriptions"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"applicationconnector.kyma-project.io"},
			Resources: []string{"applications"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}
	return reflect.DeepEqual(wantRules, clusterRole.Rules)
}

func IsEPPClusterRoleBindingCorrect(clusterRoleBinding rbacv1.ClusterRoleBinding, eventingCR v1alpha1.Eventing) bool {
	wantRoleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     fmt.Sprintf("%s-%s", eventingCR.GetName(), PublisherProxySuffix),
		APIGroup: "rbac.authorization.k8s.io",
	}
	wantSubjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      fmt.Sprintf("%s-%s", eventingCR.GetName(), PublisherProxySuffix),
			Namespace: eventingCR.Namespace,
		},
	}

	return reflect.DeepEqual(wantRoleRef, clusterRoleBinding.RoleRef) &&
		reflect.DeepEqual(wantSubjects, clusterRoleBinding.Subjects)
}
