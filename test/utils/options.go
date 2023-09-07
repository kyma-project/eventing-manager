package utils

import (
	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func WithEventingCRMinimal() EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec = v1alpha1.EventingSpec{
			Backend: v1alpha1.Backend{
				Type: v1alpha1.NatsBackendType,
			},
		}
		return nil
	}
}

func WithEventingCRName(name string) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Name = name
		return nil
	}
}

func WithEventingCRNamespace(namespace string) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Namespace = namespace
		return nil
	}
}

func WithEventingCRFinalizer(finalizer string) EventingOption {
	return func(eventing *v1alpha1.Eventing) error {
		controllerutil.AddFinalizer(eventing, finalizer)
		return nil
	}
}

func WithEventingStreamData(natsStorageType string, maxStreamSize string, natsStreamReplicas, maxMsgsPerTopic int) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec.Backend.Config = v1alpha1.BackendConfig{
			NATSStreamStorageType: natsStorageType,
			NATSStreamMaxSize:     resource.MustParse(maxStreamSize),
			NATSStreamReplicas:    natsStreamReplicas,
			NATSMaxMsgsPerTopic:   maxMsgsPerTopic,
		}
		return nil
	}
}

func WithEventingPublisherData(minReplicas, maxReplicas int, requestCPU, requestMemory, limitCPU, limitMemory string) EventingOption {
	return func(eventing *v1alpha1.Eventing) error {
		eventing.Spec.Publisher = v1alpha1.Publisher{
			Replicas: v1alpha1.Replicas{
				Min: minReplicas,
				Max: maxReplicas,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(requestCPU),
					corev1.ResourceMemory: resource.MustParse(requestMemory),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(limitCPU),    // "100m"
					corev1.ResourceMemory: resource.MustParse(limitMemory), // "128Mi"
				},
			},
		}
		return nil
	}
}

func WithEventingInvalidBackend() EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec = v1alpha1.EventingSpec{
			Backend: v1alpha1.Backend{
				Type: "invalid",
			},
		}
		return nil
	}
}

func WithEventingEventTypePrefix(eventTypePrefix string) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec.Backend.Config.EventTypePrefix = eventTypePrefix
		return nil
	}
}

func WithEventingLogLevel(logLevel string) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec.LogLevel = logLevel
		return nil
	}
}

func WithEventMeshBackend(eventMeshSecretName string) EventingOption {
	return func(eventing *v1alpha1.Eventing) error {
		eventing.Spec.Backend = v1alpha1.Backend{
			Type: v1alpha1.EventMeshBackendType,
			Config: v1alpha1.BackendConfig{
				EventMeshSecret: eventing.Namespace + "/" + eventMeshSecretName,
			},
		}
		return nil
	}
}

func WithNATSBackend() EventingOption {
	return func(eventing *v1alpha1.Eventing) error {
		eventing.Spec.Backend.Type = v1alpha1.NatsBackendType
		return nil
	}
}

func WithStatusActiveBackend(activeBackend v1alpha1.BackendType) EventingOption {
	return func(eventing *v1alpha1.Eventing) error {
		eventing.Status.ActiveBackend = activeBackend
		return nil
	}
}

func WithStatusState(state string) EventingOption {
	return func(eventing *v1alpha1.Eventing) error {
		eventing.Status.State = state
		return nil
	}
}

func WithStatusConditions(conditions []metav1.Condition) EventingOption {
	return func(eventing *v1alpha1.Eventing) error {
		eventing.Status.Conditions = conditions
		return nil
	}
}
