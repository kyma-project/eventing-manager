package eventing

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	natsClientPort    = 4222
	cpuUtilization    = 60
	memoryUtilization = 60
)

var (
	// allowedAnnotations are the publisher proxy deployment spec template annotations
	// which should be preserved during reconciliation.
	allowedAnnotations = map[string]string{
		"kubectl.kubernetes.io/restartedAt": "",
	}
)

//go:generate mockery --name=Manager --outpkg=mocks --case=underscore
type Manager interface {
	IsNATSAvailable(ctx context.Context, namespace string) (bool, error)
	DeployPublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing, backendType v1alpha1.BackendType) (*appsv1.Deployment, error)
	DeployPublisherProxyResources(context.Context, *v1alpha1.Eventing, *appsv1.Deployment) error
}

type EventingManager struct {
	ctx context.Context
	client.Client
	natsConfig    env.NATSConfig
	backendConfig env.BackendConfig
	kubeClient    k8s.Client
	logger        *logger.Logger
	recorder      record.EventRecorder
}

func NewEventingManager(
	ctx context.Context,
	client client.Client,
	kubeClient k8s.Client,
	natsConfig env.NATSConfig,
	logger *logger.Logger,
	recorder record.EventRecorder,
) Manager {
	// create an instance of ecbackend Reconciler
	backendConfig := env.GetBackendConfig()
	return EventingManager{
		ctx:           ctx,
		Client:        client,
		natsConfig:    natsConfig,
		backendConfig: backendConfig,
		kubeClient:    kubeClient,
		logger:        logger,
		recorder:      recorder,
	}
}

func (em EventingManager) DeployPublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing, backendType v1alpha1.BackendType) (*appsv1.Deployment, error) {
	// update EC reconciler NATS and public config from the data in the eventing CR
	if err := em.updateNatsConfig(ctx, eventing); err != nil {
		return nil, err
	}
	em.updatePublisherConfig(eventing)
	deployment, err := em.applyPublisherProxyDeployment(ctx, eventing, backendType)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (em *EventingManager) applyPublisherProxyDeployment(
	ctx context.Context,
	eventing *v1alpha1.Eventing,
	backendType v1alpha1.BackendType) (*appsv1.Deployment, error) {
	var desiredPublisher *appsv1.Deployment
	// update service account name.
	em.backendConfig.PublisherConfig.ServiceAccount = GetEPPServiceAccountName(*eventing)

	switch backendType {
	case v1alpha1.NatsBackendType:
		desiredPublisher = newNATSPublisherDeployment(GetEPPDeploymentName(*eventing), eventing.Namespace, em.natsConfig, em.backendConfig.PublisherConfig)
	case v1alpha1.EventMeshBackendType:
		desiredPublisher = newEventMeshPublisherDeployment(GetEPPDeploymentName(*eventing), eventing.Namespace, em.backendConfig.PublisherConfig)
	default:
		return nil, fmt.Errorf("unknown EventingBackend type %q", backendType)
	}

	if err := controllerutil.SetControllerReference(eventing, desiredPublisher, em.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %v", err)
	}

	currentPublisher, err := em.kubeClient.GetDeployment(ctx, eventing.Name, eventing.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get Event Publisher deployment: %v", err)
	}

	if currentPublisher != nil {
		// preserve only allowed annotations
		desiredPublisher.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		for k, v := range currentPublisher.Spec.Template.ObjectMeta.Annotations {
			if _, ok := allowedAnnotations[k]; ok {
				desiredPublisher.Spec.Template.ObjectMeta.Annotations[k] = v
			}
		}
	}
	// Update publisher proxy deployment
	if err := em.kubeClient.PatchApply(ctx, desiredPublisher); err != nil {
		return nil, fmt.Errorf("failed to apply Publisher Proxy deployment: %v", err)
	}

	return desiredPublisher, nil
}

func (em EventingManager) IsNATSAvailable(ctx context.Context, namespace string) (bool, error) {
	natsList, err := em.kubeClient.GetNATSResources(ctx, namespace)
	if err != nil {
		return false, err
	}
	for _, nats := range natsList.Items {
		if nats.Status.State == v1alpha1.StateReady {
			return true, nil
		}
	}
	return false, nil
}

func (em *EventingManager) getNATSUrl(ctx context.Context, namespace string) (string, error) {
	natsList, err := em.kubeClient.GetNATSResources(ctx, namespace)
	if err != nil {
		return "", err
	}
	for _, nats := range natsList.Items {
		return fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", nats.Name, nats.Namespace, natsClientPort), nil
	}
	return "", fmt.Errorf("NATS CR is not found to build NATS server URL")
}

func (em *EventingManager) updateNatsConfig(ctx context.Context, eventing *v1alpha1.Eventing) error {
	natsUrl, err := em.getNATSUrl(ctx, eventing.Namespace)
	if err != nil {
		return err
	}
	em.natsConfig.URL = natsUrl
	em.natsConfig.JSStreamStorageType = eventing.Spec.Backends[0].Config.NATSStreamStorageType
	em.natsConfig.JSStreamReplicas = eventing.Spec.Backends[0].Config.NATSStreamReplicas
	em.natsConfig.JSStreamMaxBytes = eventing.Spec.Backends[0].Config.NATSStreamMaxSize.String()
	em.natsConfig.JSStreamMaxMsgsPerTopic = int64(eventing.Spec.Backends[0].Config.NATSMaxMsgsPerTopic)
	em.natsConfig.EventTypePrefix = eventing.Spec.Backends[0].Config.EventTypePrefix
	return nil
}

func (em *EventingManager) updatePublisherConfig(eventing *v1alpha1.Eventing) {
	em.backendConfig.PublisherConfig.RequestsCPU = eventing.Spec.Publisher.Resources.Requests.Cpu().String()
	em.backendConfig.PublisherConfig.RequestsMemory = eventing.Spec.Publisher.Resources.Requests.Memory().String()
	em.backendConfig.PublisherConfig.LimitsCPU = eventing.Spec.Publisher.Resources.Limits.Cpu().String()
	em.backendConfig.PublisherConfig.LimitsMemory = eventing.Spec.Publisher.Resources.Limits.Memory().String()
	em.backendConfig.PublisherConfig.Replicas = int32(eventing.Spec.Min)
	em.backendConfig.PublisherConfig.AppLogLevel = strings.ToLower(eventing.Spec.LogLevel)
}

func (em *EventingManager) GetBackendConfig() *env.BackendConfig {
	return &em.backendConfig
}

func (em EventingManager) DeployPublisherProxyResources(
	ctx context.Context,
	eventing *v1alpha1.Eventing,
	eppDeployment *appsv1.Deployment) error {
	// define list of resources to create for EPP.
	resources := []client.Object{
		// ServiceAccount
		newPublisherProxyServiceAccount(GetEPPServiceAccountName(*eventing), eventing.Namespace, eppDeployment.Labels),
		// ClusterRole
		newPublisherProxyClusterRole(GetEPPClusterRoleName(*eventing), eventing.Namespace, eppDeployment.Labels),
		// ClusterRoleBinding
		newPublisherProxyClusterRoleBinding(GetEPPClusterRoleBindingName(*eventing), eventing.Namespace,
			eppDeployment.Labels),
		// Service to expose event publishing endpoint of EPP.
		newPublisherProxyService(GetEPPPublishServiceName(*eventing), eventing.Namespace, eppDeployment.Labels,
			eppDeployment.Spec.Template.Labels),
		// Service to expose metrics endpoint of EPP.
		newPublisherProxyMetricsService(GetEPPMetricsServiceName(*eventing), eventing.Namespace, eppDeployment.Labels,
			eppDeployment.Spec.Template.Labels),
		// Service to expose health endpoint of EPP.
		newPublisherProxyHealthService(GetEPPHealthServiceName(*eventing), eventing.Namespace, eppDeployment.Labels,
			eppDeployment.Spec.Template.Labels),
		// HPA to auto-scale publisher proxy.
		newHorizontalPodAutoscaler(eppDeployment, int32(eventing.Spec.Publisher.Min),
			int32(eventing.Spec.Publisher.Max), cpuUtilization, memoryUtilization),
	}

	// create the resources on k8s.
	for _, object := range resources {
		// add owner reference.
		if err := controllerutil.SetControllerReference(eventing, object, em.Scheme()); err != nil {
			return err
		}

		// patch apply the object.
		if err := em.kubeClient.PatchApply(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func convertECBackendType(backendType v1alpha1.BackendType) (ecv1alpha1.BackendType, error) {
	switch backendType {
	case v1alpha1.EventMeshBackendType:
		return ecv1alpha1.BEBBackendType, nil
	case v1alpha1.NatsBackendType:
		return ecv1alpha1.NatsBackendType, nil
	default:
		return "", fmt.Errorf("unknown backend type: %s", backendType)
	}
}
