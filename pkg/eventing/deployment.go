package eventing

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/kyma/components/eventing-controller/utils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	livenessInitialDelaySecs = int32(5)
	livenessTimeoutSecs      = int32(1)
	livenessPeriodSecs       = int32(2)
	eventMeshNamespacePrefix = "/"
	InstanceLabelKey         = "app.kubernetes.io/instance"
	InstanceLabelValue       = "eventing"
	DashboardLabelKey        = "kyma-project.io/dashboard"
	DashboardLabelValue      = "eventing"
	BackendLabelKey          = "eventing.kyma-project.io/backend"
	publisherPortName        = "http"
	publisherPortNum         = int32(8080)
	publisherMetricsPortName = "http-metrics"
	publisherMetricsPortNum  = int32(9090)

	AppLabelKey                     = "app.kubernetes.io/name"
	PublisherSecretClientIDKey      = "client-id"
	PublisherSecretClientSecretKey  = "client-secret"
	PublisherSecretTokenEndpointKey = "token-endpoint"

	PublisherSecretEMSURLKey       = "ems-publish-url"
	PublisherSecretBEBNamespaceKey = "beb-namespace"
)

var (
	TerminationGracePeriodSeconds = int64(30)
)

func newNATSPublisherDeployment(
	eventing *v1alpha1.Eventing,
	natsConfig env.NATSConfig,
	publisherConfig env.PublisherConfig) *appsv1.Deployment {
	return newDeployment(
		eventing,
		publisherConfig,
		WithLabels(GetPublisherDeploymentName(*eventing), v1alpha1.NatsBackendType),
		WithContainers(publisherConfig, eventing),
		WithNATSEnvVars(natsConfig, publisherConfig, eventing),
		WithLogEnvVars(publisherConfig, eventing),
		WithAffinity(GetPublisherDeploymentName(*eventing)),
	)
}

func newEventMeshPublisherDeployment(
	eventing *v1alpha1.Eventing,
	publisherConfig env.PublisherConfig) *appsv1.Deployment {
	return newDeployment(
		eventing,
		publisherConfig,
		WithLabels(GetPublisherDeploymentName(*eventing), v1alpha1.EventMeshBackendType),
		WithContainers(publisherConfig, eventing),
		WithBEBEnvVars(GetPublisherDeploymentName(*eventing), publisherConfig, eventing),
		WithLogEnvVars(publisherConfig, eventing),
	)
}

type DeployOpt func(deployment *appsv1.Deployment)

func newDeployment(eventing *v1alpha1.Eventing, publisherConfig env.PublisherConfig, opts ...DeployOpt) *appsv1.Deployment {
	newDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetPublisherDeploymentName(*eventing),
			Namespace: eventing.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: GetPublisherDeploymentName(*eventing),
				},
				Spec: v1.PodSpec{
					RestartPolicy:                 v1.RestartPolicyAlways,
					ServiceAccountName:            GetPublisherServiceAccountName(*eventing),
					TerminationGracePeriodSeconds: &TerminationGracePeriodSeconds,
					PriorityClassName:             publisherConfig.PriorityClassName,
					SecurityContext:               getPodSecurityContext(),
				},
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	for _, o := range opts {
		o(newDeployment)
	}
	return newDeployment
}

func getPodSecurityContext() *v1.PodSecurityContext {
	const id = 10001
	return &v1.PodSecurityContext{
		FSGroup:      utils.Int64Ptr(id),
		RunAsUser:    utils.Int64Ptr(id),
		RunAsGroup:   utils.Int64Ptr(id),
		RunAsNonRoot: utils.BoolPtr(true),
		SeccompProfile: &v1.SeccompProfile{
			Type: v1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func WithLabels(publisherName string, backendType v1alpha1.BackendType) DeployOpt {
	labels := map[string]string{
		AppLabelKey:       publisherName,
		InstanceLabelKey:  InstanceLabelValue,
		DashboardLabelKey: DashboardLabelValue,
	}
	return func(d *appsv1.Deployment) {
		d.Spec.Selector = metav1.SetAsLabelSelector(labels)
		d.Spec.Template.ObjectMeta.Labels = labels

		// label the event-publisher proxy with the backendType label
		labels[BackendLabelKey] = fmt.Sprint(getECBackendType(backendType))
		d.ObjectMeta.Labels = labels
	}
}

func WithAffinity(publisherName string) DeployOpt {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Affinity = &v1.Affinity{
			PodAntiAffinity: &v1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
					{
						Weight: 100,
						PodAffinityTerm: v1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{AppLabelKey: publisherName},
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
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers = []v1.Container{
			{
				Name:            GetPublisherDeploymentName(*eventing),
				Image:           publisherConfig.Image,
				Ports:           getContainerPorts(),
				LivenessProbe:   getLivenessProbe(),
				ReadinessProbe:  getReadinessProbe(),
				ImagePullPolicy: getImagePullPolicy(publisherConfig.ImagePullPolicy),
				SecurityContext: getContainerSecurityContext(),
				Env:             getCommonEnvVars(publisherConfig),
				Resources: getResources(eventing.Spec.Publisher.Resources.Requests.Cpu().String(),
					eventing.Spec.Publisher.Resources.Requests.Memory().String(),
					eventing.Spec.Publisher.Resources.Limits.Cpu().String(),
					eventing.Spec.Publisher.Resources.Limits.Memory().String()),
			},
		}
	}
}

func WithLogEnvVars(publisherConfig env.PublisherConfig, eventing *v1alpha1.Eventing) DeployOpt {
	return func(d *appsv1.Deployment) {
		for i, container := range d.Spec.Template.Spec.Containers {
			if strings.EqualFold(container.Name, GetPublisherDeploymentName(*eventing)) {
				d.Spec.Template.Spec.Containers[i].Env = append(d.Spec.Template.Spec.Containers[i].Env, getLogEnvVars(publisherConfig, eventing)...)
			}
		}
	}
}

func WithNATSEnvVars(natsConfig env.NATSConfig, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing) DeployOpt {
	return func(d *appsv1.Deployment) {
		for i, container := range d.Spec.Template.Spec.Containers {
			if strings.EqualFold(container.Name, GetPublisherDeploymentName(*eventing)) {
				d.Spec.Template.Spec.Containers[i].Env = getNATSEnvVars(natsConfig, publisherConfig, eventing)
			}
		}
	}
}

func getNATSEnvVars(natsConfig env.NATSConfig, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing) []v1.EnvVar {
	return []v1.EnvVar{
		{Name: "BACKEND", Value: "nats"},
		{Name: "PORT", Value: strconv.Itoa(int(publisherPortNum))},
		{Name: "NATS_URL", Value: natsConfig.URL},
		{Name: "REQUEST_TIMEOUT", Value: publisherConfig.RequestTimeout},
		{Name: "LEGACY_NAMESPACE", Value: "kyma"},
		{Name: "EVENT_TYPE_PREFIX", Value: eventing.Spec.Backend.Config.EventTypePrefix},
		// JetStream-specific config
		{Name: "JS_STREAM_NAME", Value: natsConfig.JSStreamName},
	}
}

func getImagePullPolicy(imagePullPolicy string) v1.PullPolicy {
	switch imagePullPolicy {
	case "IfNotPresent":
		return v1.PullIfNotPresent
	case "Always":
		return v1.PullAlways
	case "Never":
		return v1.PullNever
	default:
		return v1.PullIfNotPresent
	}
}

func getContainerSecurityContext() *v1.SecurityContext {
	return &v1.SecurityContext{
		Privileged:               utils.BoolPtr(false),
		AllowPrivilegeEscalation: utils.BoolPtr(false),
		RunAsNonRoot:             utils.BoolPtr(true),
		Capabilities: &v1.Capabilities{
			Drop: []v1.Capability{"ALL"},
		},
	}
}

func getReadinessProbe() *v1.Probe {
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path:   "/readyz",
				Port:   intstr.FromInt(8080),
				Scheme: v1.URISchemeHTTP,
			},
		},
		FailureThreshold: 3,
	}
}

func getLivenessProbe() *v1.Probe {
	return &v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.FromInt(8080),
				Scheme: v1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: livenessInitialDelaySecs,
		TimeoutSeconds:      livenessTimeoutSecs,
		PeriodSeconds:       livenessPeriodSecs,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
}

func getContainerPorts() []v1.ContainerPort {
	return []v1.ContainerPort{
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

func getCommonEnvVars(publisherConfig env.PublisherConfig) []v1.EnvVar {
	return []v1.EnvVar{
		{Name: "APPLICATION_CRD_ENABLED", Value: strconv.FormatBool(publisherConfig.ApplicationCRDEnabled)},
	}
}

func getLogEnvVars(publisherConfig env.PublisherConfig, eventing *v1alpha1.Eventing) []v1.EnvVar {
	return []v1.EnvVar{
		{Name: "APP_LOG_FORMAT", Value: publisherConfig.AppLogFormat},
		{Name: "APP_LOG_LEVEL", Value: strings.ToLower(eventing.Spec.LogLevel)},
	}
}

func getResources(requestsCPU, requestsMemory, limitsCPU, limitsMemory string) v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(requestsCPU),
			v1.ResourceMemory: resource.MustParse(requestsMemory),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(limitsCPU),
			v1.ResourceMemory: resource.MustParse(limitsMemory),
		},
	}
}

func WithBEBEnvVars(publisherName string, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing) DeployOpt {
	return func(d *appsv1.Deployment) {
		for i, container := range d.Spec.Template.Spec.Containers {
			if strings.EqualFold(container.Name, publisherName) {
				d.Spec.Template.Spec.Containers[i].Env = getEventMeshEnvVars(publisherName, publisherConfig, eventing)
			}
		}
	}
}

func getEventMeshEnvVars(publisherName string, publisherConfig env.PublisherConfig,
	eventing *v1alpha1.Eventing) []v1.EnvVar {
	return []v1.EnvVar{
		{Name: "BACKEND", Value: "beb"},
		{Name: "PORT", Value: strconv.Itoa(int(publisherPortNum))},
		{Name: "EVENT_TYPE_PREFIX", Value: eventing.Spec.Backend.Config.EventTypePrefix},
		{Name: "REQUEST_TIMEOUT", Value: publisherConfig.RequestTimeout},
		{
			Name: "CLIENT_ID",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretClientIDKey,
				}},
		},
		{
			Name: "CLIENT_SECRET",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretClientSecretKey,
				}},
		},
		{
			Name: "TOKEN_ENDPOINT",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretTokenEndpointKey,
				}},
		},
		{
			Name: "EMS_PUBLISH_URL",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretEMSURLKey,
				}},
		},
		{
			Name: "BEB_NAMESPACE_VALUE",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: publisherName},
					Key:                  PublisherSecretBEBNamespaceKey,
				}},
		},
		{
			Name:  "BEB_NAMESPACE",
			Value: fmt.Sprintf("%s$(BEB_NAMESPACE_VALUE)", eventMeshNamespacePrefix),
		},
	}
}
