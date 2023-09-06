package fixtures

import (
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NamespaceName               = "kyma-system"
	ManagerDeploymentName       = "eventing-manager"
	CRName                      = "eventing"
	ContainerName               = "manager"              // TODO: check
	SecretName                  = "eventing-nats-secret" //nolint:gosec // This is used for test purposes only.
	True                        = "true"
	podLabel                    = "nats_cluster=eventing-nats"           // TODO: check
	WebhookServerCertSecretName = "eventing-manager-webhook-server-cert" //nolint:gosec // This is used for test purposes only.
	WebhookServerCertJobName    = "eventing-manager-cert-handler"
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
	// TODO: define spec
	return &eventingv1alpha1.Eventing{}
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

func Namespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: NamespaceName,
		},
	}
}

func PodListOpts() metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: podLabel}
}
