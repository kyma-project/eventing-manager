package eventing

import (
	"context"
	"fmt"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	ecbackend "github.com/kyma-project/kyma/components/eventing-controller/controllers/backend"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	v1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const natsClientPort = 4222

type Manager interface {
	IsNATSAvailable(ctx context.Context, namespace string) (bool, error)
	CreateOrUpdatePublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing) (*v1.Deployment, error)
	CreateOrUpdateHPA(ctx context.Context, deployment *v1.Deployment, eventing *v1alpha1.Eventing, cpuUtilization, memoryUtilization int32) error
}

type EventingManager struct {
	ctx context.Context
	client.Client
	natsConfig         env.NATSConfig
	backendConfig      env.BackendConfig
	kubeClient         k8s.Client
	ecReconcilerClient ECReconcilerClient
	logger             *logger.Logger
	recorder           record.EventRecorder
}

func NewEventingManager(
	ctx context.Context,
	client client.Client,
	natsConfig env.NATSConfig,
	logger *logger.Logger,
	recorder record.EventRecorder,
) Manager {
	// create an instance of ecbackend Reconciler
	backendConfig := env.GetBackendConfig()
	ecReconciler := ecbackend.NewReconciler(
		ctx,
		nil,
		natsConfig,
		env.GetConfig(),
		backendConfig,
		nil,
		client,
		logger,
		recorder,
	)
	return EventingManager{
		ctx:                ctx,
		Client:             client,
		natsConfig:         natsConfig,
		backendConfig:      backendConfig,
		kubeClient:         k8s.NewKubeClient(client),
		ecReconcilerClient: &ECReconcilerEventingClient{ecReconciler},
		logger:             logger,
		recorder:           recorder,
	}
}

type ECReconcilerClient interface {
	CreateOrUpdatePublisherProxy(
		ctx context.Context,
		backend ecv1alpha1.BackendType,
		natsConfig env.NATSConfig,
		backendConfig env.BackendConfig) (*v1.Deployment, error)
}

type ECReconcilerEventingClient struct {
	ecReconciler *ecbackend.Reconciler
}

func (e *ECReconcilerEventingClient) CreateOrUpdatePublisherProxy(
	ctx context.Context,
	backendType ecv1alpha1.BackendType,
	natsConfig env.NATSConfig,
	backendConfig env.BackendConfig) (*v1.Deployment, error) {
	e.ecReconciler.SetNatsConfig(natsConfig)
	e.ecReconciler.SetBackendConfig(backendConfig)
	return e.ecReconciler.CreateOrUpdatePublisherProxyDeployment(ctx, backendType, false)
}

func (em EventingManager) CreateOrUpdatePublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing) (*v1.Deployment, error) {
	natsBackend := eventing.GetNATSBackend()
	var ecBackendType ecv1alpha1.BackendType
	if natsBackend == nil {
		return nil, fmt.Errorf("NATs backend is not specified in the eventing CR")
	}

	ecBackendType, err := convertECBackendType(natsBackend.Type)
	if err != nil {
		return nil, err
	}

	// update EC reconciler NATS and public config from the data in the eventing CR
	if err := em.updateNatsConfig(ctx, eventing); err != nil {
		return nil, err
	}
	em.updatePublisherConfig(eventing)
	deployment, err := em.ecReconcilerClient.CreateOrUpdatePublisherProxy(ctx, ecBackendType, em.natsConfig, em.backendConfig)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

// createOrUpdateHorizontalPodAutoscaler creates or updates the HPA for the given deployment.
func (em EventingManager) CreateOrUpdateHPA(ctx context.Context, deployment *v1.Deployment, eventing *v1alpha1.Eventing, cpuUtilization, memoryUtilization int32) error {
	// try to get the existing horizontal pod autoscaler object
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err := em.Client.Get(ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, hpa)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get horizontal pod autoscaler: %v", err)
	}
	min := int32(eventing.Spec.Publisher.Min)
	max := int32(eventing.Spec.Publisher.Max)
	hpa = createNewHorizontalPodAutoscaler(deployment, min, max, cpuUtilization, memoryUtilization)
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
	em.natsConfig.JSStreamStorageType = eventing.Spec.Backends[0].Config.NATSStorageType
	em.natsConfig.JSStreamReplicas = eventing.Spec.Backends[0].Config.NATSStreamReplicas
	em.natsConfig.JSStreamMaxBytes = eventing.Spec.Backends[0].Config.MaxStreamSize.String()
	em.natsConfig.JSStreamMaxMsgsPerTopic = eventing.Spec.Backends[0].Config.MaxMsgsPerTopic
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
