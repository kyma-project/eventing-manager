package utils

import (
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

func WithEventingCRMinimal() EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec = v1alpha1.EventingSpec{
			Backend: &v1alpha1.Backend{
				Type: v1alpha1.NatsBackendType,
			},
		}
		return nil
	}
}

func WithEventingEmptyBackend() EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec = v1alpha1.EventingSpec{}
		return nil
	}
}

func WithEventingCRName(name string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Name = name
		return nil
	}
}

func WithEventingCRNamespace(namespace string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Namespace = namespace
		return nil
	}
}

func WithEventingCRFinalizer(finalizer string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		controllerutil.AddFinalizer(e, finalizer)
		return nil
	}
}

func WithEventingStreamData(natsStorageType string, maxStreamSize string, natsStreamReplicas, maxMsgsPerTopic int) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec.Backend.Config = v1alpha1.BackendConfig{
			NATSStreamStorageType: natsStorageType,
			NATSStreamMaxSize:     resource.MustParse(maxStreamSize),
			NATSStreamReplicas:    natsStreamReplicas,
			NATSMaxMsgsPerTopic:   maxMsgsPerTopic,
		}
		return nil
	}
}

func WithEventingPublisherData(minReplicas, maxReplicas int, requestCPU, requestMemory, limitCPU, limitMemory string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec.Publisher = v1alpha1.Publisher{
			Replicas: v1alpha1.Replicas{
				Min: minReplicas,
				Max: maxReplicas,
			},
			Resources: kcorev1.ResourceRequirements{
				Requests: kcorev1.ResourceList{
					kcorev1.ResourceCPU:    resource.MustParse(requestCPU),
					kcorev1.ResourceMemory: resource.MustParse(requestMemory),
				},
				Limits: kcorev1.ResourceList{
					kcorev1.ResourceCPU:    resource.MustParse(limitCPU),    // "100m"
					kcorev1.ResourceMemory: resource.MustParse(limitMemory), // "128Mi"
				},
			},
		}
		return nil
	}
}

func WithEventingInvalidBackend() EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec = v1alpha1.EventingSpec{
			Backend: &v1alpha1.Backend{
				Type: "invalid",
			},
		}
		return nil
	}
}

func WithEventingEventTypePrefix(eventTypePrefix string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec.Backend.Config.EventTypePrefix = eventTypePrefix
		return nil
	}
}

func WithEventingDomain(domain string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec.Backend.Config.Domain = domain
		return nil
	}
}

func WithEventingLogLevel(logLevel string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec.LogLevel = logLevel
		return nil
	}
}

func WithEventMeshBackend(eventMeshSecretName string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		if e.Spec.Backend == nil {
			e.Spec.Backend = &v1alpha1.Backend{}
		}
		e.Spec.Backend.Type = v1alpha1.EventMeshBackendType
		e.Spec.Backend.Config.EventMeshSecret = e.Namespace + "/" + eventMeshSecretName
		return nil
	}
}

func WithEmptyBackend() EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec.Backend = &v1alpha1.Backend{}
		return nil
	}
}

func WithNATSBackend() EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Spec.Backend.Type = v1alpha1.NatsBackendType
		return nil
	}
}

func WithStatusActiveBackend(activeBackend v1alpha1.BackendType) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Status.ActiveBackend = activeBackend
		return nil
	}
}

func WithStatusState(state string) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Status.State = state
		return nil
	}
}

func WithStatusConditions(conditions []kmetav1.Condition) EventingOption {
	return func(e *v1alpha1.Eventing) error {
		e.Status.Conditions = conditions
		return nil
	}
}
