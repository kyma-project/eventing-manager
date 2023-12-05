package fixtures

import (
	"errors"
	"fmt"
	"strings"

	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/eventing"
)

const (
	FieldManager                = "eventing-tests"
	NamespaceName               = "kyma-system"
	ManagerDeploymentName       = "eventing-manager"
	CRName                      = "eventing"
	ManagerContainerName        = "manager"
	PublisherContainerName      = "eventing-publisher-proxy"
	WebhookServerCertSecretName = "eventing-manager-webhook-server-cert"
	WebhookServerCertJobName    = "eventing-manager-cert-handler"
	EventMeshSecretNamespace    = "kyma-system"
	EventMeshSecretName         = "eventing-backend"
	EventOriginalTypeHeader     = "originaltype"
)

type SubscriptionCRVersion string

const (
	V1Alpha1SubscriptionCRVersion SubscriptionCRVersion = "v1alpha1"
	V1Alpha2SubscriptionCRVersion SubscriptionCRVersion = "v1alpha2"
)

func EventingCR(backendType operatorv1alpha1.BackendType) *operatorv1alpha1.Eventing {
	if backendType == operatorv1alpha1.EventMeshBackendType {
		return EventingEventMeshCR()
	}
	return EventingNATSCR()
}

func PeerAuthenticationGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "security.istio.io",
		Version:  "v1beta1",
		Resource: "peerauthentications",
	}
}

func EventingNATSCR() *operatorv1alpha1.Eventing {
	return &operatorv1alpha1.Eventing{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Eventing",
			APIVersion: "operator.kyma-project.io/v1alpha1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      CRName,
			Namespace: NamespaceName,
		},
		Spec: operatorv1alpha1.EventingSpec{
			Backend: operatorv1alpha1.Backend{
				Type: "NATS",
				Config: operatorv1alpha1.BackendConfig{
					NATSStreamStorageType: "File",
					NATSStreamReplicas:    3,
					NATSStreamMaxSize:     resource.MustParse("700Mi"),
					NATSMaxMsgsPerTopic:   1000000,
				},
			},
			Publisher: PublisherSpec(),
		},
	}
}

func EventingEventMeshCR() *operatorv1alpha1.Eventing {
	return &operatorv1alpha1.Eventing{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Eventing",
			APIVersion: "operator.kyma-project.io/v1alpha1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      CRName,
			Namespace: NamespaceName,
		},
		Spec: operatorv1alpha1.EventingSpec{
			Backend: operatorv1alpha1.Backend{
				Type: "EventMesh",
				Config: operatorv1alpha1.BackendConfig{
					EventMeshSecret: fmt.Sprintf("%s/%s", EventMeshSecretNamespace, EventMeshSecretName),
				},
			},
			Publisher: PublisherSpec(),
		},
	}
}

func PublisherSpec() operatorv1alpha1.Publisher {
	return operatorv1alpha1.Publisher{
		Replicas: operatorv1alpha1.Replicas{
			Min: 2,
			Max: 2,
		},
		Resources: kcorev1.ResourceRequirements{
			Limits: kcorev1.ResourceList{
				"cpu":    resource.MustParse("300m"),
				"memory": resource.MustParse("312Mi"),
			},
			Requests: kcorev1.ResourceList{
				"cpu":    resource.MustParse("100m"),
				"memory": resource.MustParse("156Mi"),
			},
		},
	}
}

func V1Alpha1SubscriptionsToTest() []eventing.TestSubscriptionInfo {
	return []eventing.TestSubscriptionInfo{
		{
			Name:        "test-sub-1-v1alpha1",
			Description: "event type and source without any alpha-numeric characters",
			Types:       []string{"sap.kyma.custom.noapp.order.tested.v1"},
		},
		{
			Name:        "test-sub-2-v1alpha1",
			Description: "event type and source with alpha-numeric characters",
			Types:       []string{"sap.kyma.custom.test-app.order-$.second.R-e-c-e-i-v-e-d.v1"},
		},
		{
			Name:        "test-sub-3-with-multiple-types-v1alpha1",
			Description: "multiple types in same subscription",
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
			Description: "event type and source without any alpha-numeric characters",
			Source:      "noapp",
			Types:       []string{"order.modified.v1"},
		},
		{
			Name:        "test-sub-2-v1alpha2",
			Description: "event type and source with alpha-numeric characters",
			Source:      "test-app",
			Types:       []string{"Order-$.third.R-e-c-e-i-v-e-d.v1"},
		},
		{
			Name:        "test-sub-3-with-multiple-types-v1alpha2",
			Description: "multiple types in same subscription",
			Source:      "test-evnt",
			Types: []string{
				"or-der.crea-ted.one.two.three.four.v4",
				"order.testing.v1",
			},
		},
	}
}

func Namespace(name string) *kcorev1.Namespace {
	return &kcorev1.Namespace{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewSinkDeployment(name, namespace, image string) *kappsv1.Deployment {
	labels := map[string]string{
		"source": "eventing-tests",
		"name":   name,
	}
	return &kappsv1.Deployment{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: kappsv1.DeploymentSpec{
			Selector: kmetav1.SetAsLabelSelector(labels),
			Template: kcorev1.PodTemplateSpec{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:   name,
					Labels: labels,
				},
				Spec: kcorev1.PodSpec{
					RestartPolicy: kcorev1.RestartPolicyAlways,
					Containers: []kcorev1.Container{
						{
							Name:  name,
							Image: image,
							Args: []string{
								"subscriber",
								"--listen-port=8080",
							},
							Ports: []kcorev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
								},
							},
							ImagePullPolicy: kcorev1.PullAlways,
							Resources: kcorev1.ResourceRequirements{
								Limits: kcorev1.ResourceList{
									"cpu":    resource.MustParse("300m"),
									"memory": resource.MustParse("312Mi"),
								},
								Requests: kcorev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("156Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewSinkService(name, namespace string) *kcorev1.Service {
	labels := map[string]string{
		"source": "eventing-tests",
		"name":   name,
	}
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
			Selector: labels,
			Ports: []kcorev1.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
}

func FindContainerInPod(pod kcorev1.Pod, name string) *kcorev1.Container {
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

func AppendMsgToError(err error, msg string) error {
	return errors.Join(err, fmt.Errorf("\n==> %s", msg))
}
