package eventing

import (
	"context"
	"fmt"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const natsClientPort = 4222

var (
	// allowedAnnotations are the publisher proxy deployment spec template annotations
	// which should be preserved during reconciliation.
	allowedAnnotations = map[string]string{
		"kubectl.kubernetes.io/restartedAt": "",
	}
)

type Manager interface {
	IsNATSAvailable(ctx context.Context, namespace string) (bool, error)
	CreateOrUpdatePublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing, backendType v1alpha1.BackendType) (*appsv1.Deployment, error)
	CreateOrUpdateHPA(ctx context.Context, deployment *appsv1.Deployment, eventing *v1alpha1.Eventing, cpuUtilization, memoryUtilization int32) error
	DeployPublisherProxyResources(context.Context, *v1alpha1.Eventing, *appsv1.Deployment, *runtime.Scheme) error
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

type ECReconcilerClient interface {
	CreateOrUpdatePublisherProxy(
		ctx context.Context,
		backend ecv1alpha1.BackendType,
		natsConfig env.NATSConfig,
		backendConfig env.BackendConfig) (*appsv1.Deployment, error)
}

func (em EventingManager) CreateOrUpdatePublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing, backendType v1alpha1.BackendType) (*appsv1.Deployment, error) {
	// update EC reconciler NATS and public config from the data in the eventing CR
	if err := em.updateNatsConfig(ctx, eventing); err != nil {
		return nil, err
	}
	em.updatePublisherConfig(eventing)
	// deployment, err := em.ecReconcilerClient.CreateOrUpdatePublisherProxy(ctx, ecBackendType, em.natsConfig, em.backendConfig)
	deployment, err := em.applyPublisherProxyDeployment(ctx, eventing, backendType)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (em *EventingManager) applyPublisherProxyDeployment(
	ctx context.Context,
	eventing *v1alpha1.Eventing,
	backend v1alpha1.BackendType) (*appsv1.Deployment, error) {
	var desiredPublisher *appsv1.Deployment

	switch backend {
	case v1alpha1.NatsBackendType:
		desiredPublisher = newNATSPublisherDeployment(eventing.Name, eventing.Namespace, em.natsConfig, em.backendConfig.PublisherConfig)
	case v1alpha1.EventMeshBackendType:
		desiredPublisher = newBEBPublisherDeployment(eventing.Name, eventing.Namespace, em.backendConfig.PublisherConfig)
	default:
		return nil, fmt.Errorf("unknown EventingBackend type %q", backend)
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
		return nil, fmt.Errorf("update Event Publisher deployment failed: %v", err)
	}

	return desiredPublisher, nil
}

// CreateOrUpdateHPA creates or updates the HPA for the given deployment.
func (em EventingManager) CreateOrUpdateHPA(ctx context.Context, deployment *appsv1.Deployment, eventing *v1alpha1.Eventing, cpuUtilization, memoryUtilization int32) error {
	// try to get the existing horizontal pod autoscaler object
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err := em.Client.Get(ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, hpa)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get horizontal pod autoscaler: %v", err)
	}
	min := int32(eventing.Spec.Publisher.Min)
	max := int32(eventing.Spec.Publisher.Max)
	hpa = newHorizontalPodAutoscaler(deployment, min, max, cpuUtilization, memoryUtilization)
	if err := controllerutil.SetControllerReference(eventing, hpa, em.Scheme()); err != nil {
		return err
	}
	// if the horizontal pod autoscaler object does not exist, create it
	if errors.IsNotFound(err) {
		// create a new horizontal pod autoscaler object
		err = em.Client.Create(ctx, hpa)
		if err != nil {
			return fmt.Errorf("failed to create horizontal pod autoscaler: %v", err)
		}
		return nil
	}

	// if the horizontal pod autoscaler object exists, update it
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err = em.Client.Update(ctx, hpa)
		if err != nil {
			return fmt.Errorf("failed to update horizontal pod autoscaler: %v", err)
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("failed to update horizontal pod autoscaler: %v", retryErr)
	}

	return nil
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
	return nil
}

func (em *EventingManager) updatePublisherConfig(eventing *v1alpha1.Eventing) {
	em.backendConfig.PublisherConfig.RequestsCPU = eventing.Spec.Publisher.Resources.Requests.Cpu().String()
	em.backendConfig.PublisherConfig.RequestsMemory = eventing.Spec.Publisher.Resources.Requests.Memory().String()
	em.backendConfig.PublisherConfig.LimitsCPU = eventing.Spec.Publisher.Resources.Limits.Cpu().String()
	em.backendConfig.PublisherConfig.LimitsMemory = eventing.Spec.Publisher.Resources.Limits.Memory().String()
	em.backendConfig.PublisherConfig.Replicas = int32(eventing.Spec.Min)
}

func (em *EventingManager) GetBackendConfig() *env.BackendConfig {
	return &em.backendConfig
}

func (em EventingManager) DeployPublisherProxyResources(
	ctx context.Context,
	eventing *v1alpha1.Eventing,
	eppDeployment *appsv1.Deployment,
	scheme *runtime.Scheme) error {
	// define list of resources to create for EPP.
	resources := []client.Object{
		// ServiceAccount
		newPublisherProxyServiceAccount(eppDeployment.Name, eppDeployment.Namespace, eppDeployment.Labels),
		// ClusterRole
		newPublisherProxyClusterRole(eppDeployment.Name, eppDeployment.Namespace, eppDeployment.Labels),
		// ClusterRoleBinding
		newPublisherProxyClusterRoleBinding(eppDeployment.Name, eppDeployment.Namespace, eppDeployment.Labels),
		// Service to expose event publishing endpoint of EPP.
		newPublisherProxyService(eppDeployment.Name, eppDeployment.Namespace, eppDeployment.Labels,
			eppDeployment.Spec.Template.Labels),
		// Service to expose metrics endpoint of EPP.
		newPublisherProxyMetricsService(eppDeployment.Name, eppDeployment.Namespace, eppDeployment.Labels,
			eppDeployment.Spec.Template.Labels),
		// Service to expose health endpoint of EPP.
		newPublisherProxyHealthService(eppDeployment.Name, eppDeployment.Namespace, eppDeployment.Labels,
			eppDeployment.Spec.Template.Labels),
	}

	// add PeerAuthentication for istio.
	paExists, err := em.kubeClient.PeerAuthenticationCRDExists(ctx)
	if err != nil {
		return err
	}
	if paExists {
		resources = append(resources, newPublisherProxyPeerAuthentication(eppDeployment.Name, eppDeployment.Namespace,
			eppDeployment.Labels,
			eppDeployment.Spec.Template.Labels),
		)
	}

	// create the resources on k8s.
	for _, object := range resources {
		// add owner reference.
		if err = controllerutil.SetControllerReference(eventing, object, scheme); err != nil {
			return err
		}

		// patch apply the object.
		if err = em.kubeClient.PatchApply(ctx, object); err != nil {
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
