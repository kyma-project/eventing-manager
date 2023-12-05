package eventing

import (
	"fmt"
	"strconv"
	"strings"

	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/label"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/utils"
)

const (
	livenessInitialDelaySecs = int32(5)
	livenessTimeoutSecs      = int32(1)
	livenessPeriodSecs       = int32(2)
	eventMeshNamespacePrefix = "/"
	publisherPortName        = "http"
	publisherPortNum         = int32(8080)
	publisherMetricsPortName = "http-metrics"
	publisherMetricsPortNum  = int32(9090)
	PublisherName            = "eventing-publisher-proxy"

	PublisherSecretClientIDKey      = "client-id"
	PublisherSecretClientSecretKey  = "client-secret"
	PublisherSecretTokenEndpointKey = "token-endpoint"

	PublisherSecretEMSURLKey       = "ems-publish-url"
	PublisherSecretBEBNamespaceKey = "beb-namespace"

	PriorityClassName = "eventing-manager-priority-class"
)

var TerminationGracePeriodSeconds = int64(30)

func newNATSPublisherDeployment(
	eventing *v1alpha1.Eventing,
	natsConfig env.NATSConfig,
	publisherConfig env.PublisherConfig,
) *kappsv1.Deployment {
	return newDeployment(
		eventing,
		publisherConfig,
		WithLabels(GetPublisherDeploymentName(*eventing), v1alpha1.NatsBackendType),
		WithSelector(GetPublisherDeploymentName(*eventing)),
		WithContainers(publisherConfig, eventing),
		WithNATSEnvVars(natsConfig, publisherConfig, eventing),
		WithLogEnvVars(publisherConfig, eventing),
		WithAffinity(GetPublisherDeploymentName(*eventing)),
		WithPriorityClassName(PriorityClassName),
	)
}

func newEventMeshPublisherDeployment(
	eventing *v1alpha1.Eventing,
	publisherConfig env.PublisherConfig,
) *kappsv1.Deployment {
	return newDeployment(
		eventing,
		publisherConfig,
		WithLabels(GetPublisherDeploymentName(*eventing), v1alpha1.EventMeshBackendType),
		WithSelector(GetPublisherDeploymentName(*eventing)),
		WithContainers(publisherConfig, eventing),
		WithBEBEnvVars(GetPublisherDeploymentName(*eventing), publisherConfig, eventing),
		WithLogEnvVars(publisherConfig, eventing),
		WithPriorityClassName(PriorityClassName),
	)
}

type DeployOpt func(deployment *kappsv1.Deployment)

func newDeployment(eventing *v1alpha1.Eventing, publisherConfig env.PublisherConfig, opts ...DeployOpt) *kappsv1.Deployment {
	newDeployment := &kappsv1.Deployment{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      GetPublisherDeploymentName(*eventing),
			Namespace: eventing.Namespace,
		},
		Spec: kappsv1.DeploymentSpec{
			Template: kcorev1.PodTemplateSpec{
				ObjectMeta: kmetav1.ObjectMeta{
					Name: GetPublisherDeploymentName(*eventing),
				},
				Spec: kcorev1.PodSpec{
					RestartPolicy:                 kcorev1.RestartPolicyAlways,
					ServiceAccountName:            GetPublisherServiceAccountName(*eventing),
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					PriorityClassName:             publisherConfig.PriorityClassName,
					SecurityContext:               getPodSecurityContext(),
				},
			},
		},
		Status: kappsv1.DeploymentStatus{},
	}
	for _, o := range opts {
		o(newDeployment)
	}
	return newDeployment
}

func getPodSecurityContext() *kcorev1.PodSecurityContext {
	const id = 10001
	return &kcorev1.PodSecurityContext{
		FSGroup:      utils.Int64Ptr(id),
		RunAsUser:    utils.Int64Ptr(id),
		RunAsGroup:   utils.Int64Ptr(id),
		RunAsNonRoot: utils.BoolPtr(true),
		SeccompProfile: &kcorev1.SeccompProfile{
			Type: kcorev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func getLabels(publisherName string, backendType v1alpha1.BackendType) map[string]string {
	return map[string]string{
		label.KeyComponent: label.ValueEventingManager,
		label.KeyCreatedBy: label.ValueEventingManager,
		label.KeyInstance:  label.ValueEventing,
		label.KeyManagedBy: label.ValueEventingManager,
		label.KeyName:      publisherName,
		label.KeyPartOf:    label.ValueEventingManager,
		label.KeyBackend:   fmt.Sprint(getECBackendType(backendType)),
		label.KeyDashboard: label.ValueEventing,
	}
}

func WithLabels(publisherName string, backendType v1alpha1.BackendType) DeployOpt {
	return func(d *kappsv1.Deployment) {
		labels := getLabels(publisherName, backendType)
		d.ObjectMeta.Labels = labels
		d.Spec.Template.ObjectMeta.Labels = labels
	}
}

func getSelector(publisherName string) *kmetav1.LabelSelector {
	labels := map[string]string{
		label.KeyInstance:  label.ValueEventing,
		label.KeyName:      publisherName,
		label.KeyDashboard: label.ValueEventing,
	}
	return kmetav1.SetAsLabelSelector(labels)
}

func WithSelector(publisherName string) DeployOpt {
	return func(d *kappsv1.Deployment) {
		d.Spec.Selector = getSelector(publisherName)
	}
}

func WithPriorityClassName(name string) DeployOpt {
	return func(deployment *kappsv1.Deployment) {
		deployment.Spec.Template.Spec.PriorityClassName = name
	}
}

func WithAffinity(publisherName string) DeployOpt {
	return func(d *kappsv1.Deployment) {
		d.Spec.Template.Spec.Affinity = &kcorev1.Affinity{
			PodAntiAffinity: &kcorev1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []kcorev1.WeightedPodAffinityTerm{
					{
						Weight: 100,
						PodAffinityTerm: kcorev1.PodAffinityTerm{
							LabelSelector: &kmetav1.LabelSelector{
								MatchLabels: map[string]string{label.KeyName: publisherName},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
		}
	}
}

func WithContainers(publisherConfig env.PublisherConfig, eventing *v1alpha1.Eventing) DeployOpt {
	return func(d *kappsv1.Deployment) {
		d.Spec.Template.Spec.Containers = []kcorev1.Container{
			{
				Name:            GetPublisherDeploymentName(*eventing),
				Image:           publisherConfig.Image,
				Ports:           getContainerPorts(),
				LivenessProbe:   getLivenessProbe(),
				ReadinessProbe:  getReadinessProbe(),
				ImagePullPolicy: getImagePullPolicy(publisherConfig.ImagePullPolicy),
				SecurityContext: getContainerSecurityContext(),
				Resources: getResources(eventing.Spec.Publisher.Resources.Requests.Cpu().String(),
					eventing.Spec.Publisher.Resources.Requests.Memory().String(),
					eventing.Spec.Publisher.Resources.Limits.Cpu().String(),
					eventing.Spec.Publisher.Resources.Limits.Memory().String()),
			},
		}
	}
}

func WithLogEnvVars(publisherConfig env.PublisherConfig, eventing *v1alpha1.Eventing) DeployOpt {
	return func(d *kappsv1.Deployment) {
		for i, container := range d.Spec.Template.Spec.Containers {
			if strings.EqualFold(container.Name, GetPublisherDeploymentName(*eventing)) {
				d.Spec.Template.Spec.Containers[i].Env = append(d.Spec.Template.Spec.Containers[i].Env, getLogEnvVars(publisherConfig, eventing)...)
			}
		}
	}
}

func WithNATSEnvVars(natsConfig env.NATSConfig, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing,
) DeployOpt {
	return func(d *kappsv1.Deployment) {
		for i, container := range d.Spec.Template.Spec.Containers {
			if strings.EqualFold(container.Name, GetPublisherDeploymentName(*eventing)) {
				d.Spec.Template.Spec.Containers[i].Env = getNATSEnvVars(natsConfig, publisherConfig, eventing)
			}
		}
	}
}

func getNATSEnvVars(natsConfig env.NATSConfig, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing,
) []kcorev1.EnvVar {
	return []kcorev1.EnvVar{
		{Name: "BACKEND", Value: "nats"},
		{Name: "PORT", Value: strconv.Itoa(int(publisherPortNum))},
		{Name: "NATS_URL", Value: natsConfig.URL},
		{Name: "REQUEST_TIMEOUT", Value: publisherConfig.RequestTimeout},
		{Name: "LEGACY_NAMESPACE", Value: "kyma"},
		{Name: "EVENT_TYPE_PREFIX", Value: eventing.Spec.Backend.Config.EventTypePrefix},
		{Name: "APPLICATION_CRD_ENABLED", Value: strconv.FormatBool(publisherConfig.ApplicationCRDEnabled)},
		// JetStream-specific config
		{Name: "JS_STREAM_NAME", Value: natsConfig.JSStreamName},
	}
}

func getImagePullPolicy(imagePullPolicy string) kcorev1.PullPolicy {
	switch imagePullPolicy {
	case "IfNotPresent":
		return kcorev1.PullIfNotPresent
	case "Always":
		return kcorev1.PullAlways
	case "Never":
		return kcorev1.PullNever
	default:
		return kcorev1.PullIfNotPresent
	}
}

func getContainerSecurityContext() *kcorev1.SecurityContext {
	return &kcorev1.SecurityContext{
		Privileged:               utils.BoolPtr(false),
		AllowPrivilegeEscalation: utils.BoolPtr(false),
		RunAsNonRoot:             utils.BoolPtr(true),
		Capabilities: &kcorev1.Capabilities{
			Drop: []kcorev1.Capability{"ALL"},
		},
	}
}

func getReadinessProbe() *kcorev1.Probe {
	return &kcorev1.Probe{
		ProbeHandler: kcorev1.ProbeHandler{
			HTTPGet: &kcorev1.HTTPGetAction{
				Path:   "/readyz",
				Port:   intstr.FromInt32(8080),
				Scheme: kcorev1.URISchemeHTTP,
			},
		},
		FailureThreshold: 3,
	}
}

func getLivenessProbe() *kcorev1.Probe {
	return &kcorev1.Probe{
		ProbeHandler: kcorev1.ProbeHandler{
			HTTPGet: &kcorev1.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.FromInt32(8080),
				Scheme: kcorev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: livenessInitialDelaySecs,
		TimeoutSeconds:      livenessTimeoutSecs,
		PeriodSeconds:       livenessPeriodSecs,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
}

func getContainerPorts() []kcorev1.ContainerPort {
	return []kcorev1.ContainerPort{
		{
			Name:          publisherPortName,
			ContainerPort: publisherPortNum,
		},
		{
			Name:          publisherMetricsPortName,
			ContainerPort: publisherMetricsPortNum,
		},
	}
}

func getLogEnvVars(publisherConfig env.PublisherConfig, eventing *v1alpha1.Eventing) []kcorev1.EnvVar {
	return []kcorev1.EnvVar{
		{Name: "APP_LOG_FORMAT", Value: publisherConfig.AppLogFormat},
		{Name: "APP_LOG_LEVEL", Value: strings.ToLower(eventing.Spec.LogLevel)},
	}
}

func getResources(requestsCPU, requestsMemory, limitsCPU, limitsMemory string) kcorev1.ResourceRequirements {
	return kcorev1.ResourceRequirements{
		Requests: kcorev1.ResourceList{
			kcorev1.ResourceCPU:    resource.MustParse(requestsCPU),
			kcorev1.ResourceMemory: resource.MustParse(requestsMemory),
		},
		Limits: kcorev1.ResourceList{
			kcorev1.ResourceCPU:    resource.MustParse(limitsCPU),
			kcorev1.ResourceMemory: resource.MustParse(limitsMemory),
		},
	}
}

func WithBEBEnvVars(publisherName string, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing,
) DeployOpt {
	return func(d *kappsv1.Deployment) {
		for i, container := range d.Spec.Template.Spec.Containers {
			if strings.EqualFold(container.Name, publisherName) {
				d.Spec.Template.Spec.Containers[i].Env = getEventMeshEnvVars(publisherName, publisherConfig, eventing)
			}
		}
	}
}

func getEventMeshEnvVars(publisherName string, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing,
) []kcorev1.EnvVar {
	return []kcorev1.EnvVar{
		{Name: "BACKEND", Value: "beb"},
		{Name: "PORT", Value: strconv.Itoa(int(publisherPortNum))},
		{Name: "EVENT_TYPE_PREFIX", Value: eventing.Spec.Backend.Config.EventTypePrefix},
		{Name: "APPLICATION_CRD_ENABLED", Value: strconv.FormatBool(publisherConfig.ApplicationCRDEnabled)},
		{Name: "REQUEST_TIMEOUT", Value: publisherConfig.RequestTimeout},
		{
			Name: "CLIENT_ID",
			ValueFrom: &kcorev1.EnvVarSource{
				SecretKeyRef: &kcorev1.SecretKeySelector{
					LocalObjectReference: kcorev1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretClientIDKey,
				},
			},
		},
		{
			Name: "CLIENT_SECRET",
			ValueFrom: &kcorev1.EnvVarSource{
				SecretKeyRef: &kcorev1.SecretKeySelector{
					LocalObjectReference: kcorev1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretClientSecretKey,
				},
			},
		},
		{
			Name: "TOKEN_ENDPOINT",
			ValueFrom: &kcorev1.EnvVarSource{
				SecretKeyRef: &kcorev1.SecretKeySelector{
					LocalObjectReference: kcorev1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretTokenEndpointKey,
				},
			},
		},
		{
			Name: "EMS_PUBLISH_URL",
			ValueFrom: &kcorev1.EnvVarSource{
				SecretKeyRef: &kcorev1.SecretKeySelector{
					LocalObjectReference: kcorev1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretEMSURLKey,
				},
			},
		},
		{
			Name: "BEB_NAMESPACE_VALUE",
			ValueFrom: &kcorev1.EnvVarSource{
				SecretKeyRef: &kcorev1.SecretKeySelector{
					LocalObjectReference: kcorev1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretBEBNamespaceKey,
				},
			},
		},
		{
			Name:  "BEB_NAMESPACE",
			Value: fmt.Sprintf("%s$(BEB_NAMESPACE_VALUE)", eventMeshNamespacePrefix),
		},
	}
}
