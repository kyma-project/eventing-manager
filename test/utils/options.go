package utils

import (
	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func WithEventingCRMinimal() EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec = v1alpha1.EventingSpec{
			Backends: []v1alpha1.Backend{
				{
					Type: v1alpha1.NatsBackendType,
				},
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

func WithEventingStreamData(natsStorageType string, maxStreamSize string, natsStreamReplicas, maxMsgsPerTopic int) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec.Backends[0].Config = v1alpha1.BackendConfig{
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
			Backends: []v1alpha1.Backend{
				{
					Type: "invalid",
				},
			},
		}
		return nil
	}
}

func WithEventingEventTypePrefix(eventTypePrefix string) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec.Backends[0].Config.EventTypePrefix = eventTypePrefix
		return nil
	}
}

func WithEventingLogLevel(logLevel string) EventingOption {
	return func(nats *v1alpha1.Eventing) error {
		nats.Spec.LogLevel = logLevel
		return nil
	}
}
