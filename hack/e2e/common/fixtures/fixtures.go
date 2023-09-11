package fixtures

import (
	"fmt"
	"strings"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/eventing"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NamespaceName               = "kyma-system"
	ManagerDeploymentName       = "eventing-manager"
	CRName                      = "eventing"
	ManagerContainerName        = "manager"
	PublisherContainerName      = "eventing-publisher-proxy"
	WebhookServerCertSecretName = "eventing-manager-webhook-server-cert" //nolint:gosec // This is used for test purposes only.
	WebhookServerCertJobName    = "eventing-manager-cert-handler"
	EventMeshSecretNamespace    = "kyma-system"
	EventMeshSecretName         = "eventing-backend"
)

func EventingCR(backendType eventingv1alpha1.BackendType) *eventingv1alpha1.Eventing {
	if backendType == eventingv1alpha1.EventMeshBackendType {
		return EventingEventMeshCR()
	}
	return EventingNATSCR()
}

func EventingNATSCR() *eventingv1alpha1.Eventing {
	return &eventingv1alpha1.Eventing{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Eventing",
			APIVersion: "operator.kyma-project.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CRName,
			Namespace: NamespaceName,
		},
		Spec: eventingv1alpha1.EventingSpec{
			Backend: eventingv1alpha1.Backend{
				Type: "NATS",
				Config: eventingv1alpha1.BackendConfig{
					NATSStreamStorageType: "File",
					NATSStreamReplicas:    3,
					NATSStreamMaxSize:     resource.MustParse("700m"),
					NATSMaxMsgsPerTopic:   1000000,
				},
			},
			Publisher: PublisherSpec(),
		},
	}
}

func EventingEventMeshCR() *eventingv1alpha1.Eventing {
	return &eventingv1alpha1.Eventing{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Eventing",
			APIVersion: "operator.kyma-project.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CRName,
			Namespace: NamespaceName,
		},
		Spec: eventingv1alpha1.EventingSpec{
			Backend: eventingv1alpha1.Backend{
				Type: "EventMesh",
				Config: eventingv1alpha1.BackendConfig{
					EventMeshSecret: fmt.Sprintf("%s/%s", EventMeshSecretNamespace, EventMeshSecretName),
				},
			},
			Publisher: PublisherSpec(),
		},
	}
}

func PublisherSpec() eventingv1alpha1.Publisher {
	return eventingv1alpha1.Publisher{
		Replicas: eventingv1alpha1.Replicas{
			Min: 2,
			Max: 2,
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("300m"),
				"memory": resource.MustParse("312Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("100m"),
				"memory": resource.MustParse("156Mi"),
			},
		},
	}
}

func V1Alpha1SubscriptionsToTest() []eventing.TestSubscriptionInfo {
	return []eventing.TestSubscriptionInfo{
		{
			Name:  "test-sub-1-v1alpha1",
			Types: []string{"sap.kyma.custom.noapp.order.tested.v1"},
		},
		{
			Name:  "test-sub-2-v1alpha1",
			Types: []string{"sap.kyma.custom.test-app.order-$.second.R-e-c-e-i-v-e-d.v1"},
		},
		{
			Name: "test-sub-3-with-multiple-types-v1alpha1",
			Types: []string{
				"sap.kyma.custom.connected-app.order.tested.v1",
				"sap.kyma.custom.connected-app2.or-der.crea-ted.one.two.three.v4",
			},
		},
	}
}

func V1Alpha2SubscriptionsToTest() []eventing.TestSubscriptionInfo {
	return []eventing.TestSubscriptionInfo{
		{
			Name:        "test-sub-1-v1alpha2",
			Description: "Test event type and source without any alpha-numeric characters",
			Source:      "noapp",
			Types:       []string{"order.modified.v1"},
		},
		{
			Name:        "test-sub-2-v1alpha2",
			Description: "Test event type and source with any alpha-numeric characters",
			Source:      "test-app",
			Types:       []string{"Order-$.third.R-e-c-e-i-v-e-d.v1"},
		},
		{
			Name:   "test-sub-3-with-multiple-types-v1alpha2",
			Source: "test-evnt",
			Types: []string{
				"or-der.crea-ted.one.two.three.four.v4",
				"order.testing.v1",
			},
		},
	}
}

func Namespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func FindContainerInPod(pod corev1.Pod, name string) *corev1.Container {
	for _, container := range pod.Spec.Containers {
		if container.Name == name {
			return &container
		}
	}
	return nil
}

func ConvertSelectorLabelsToString(labels map[string]string) string {
	var result []string
	for k, v := range labels {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(result, ",")
}
