package eventing

import (
	"context"
	"fmt"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	ecbackend "github.com/kyma-project/kyma/components/eventing-controller/controllers/backend"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	v1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const natsClientPort = 4222

type Manager interface {
	IsNATSAvailable(ctx context.Context, namespace string) (bool, error)
	CreateOrUpdatePublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing) (*v1.Deployment, error)
	CreateOrUpdateHPA(ctx context.Context, deployment *v1.Deployment, eventing *eventingv1alpha1.Eventing, cpuUtilization, memoryUtilization int32) error
}

type EventingManager struct {
	ctx context.Context
	client.Client
	natsConfig env.NATSConfig
	kubeClient k8s.Client
	logger     *logger.Logger
	recorder   record.EventRecorder
}

func NewEventingManager(
	ctx context.Context,
	client client.Client,
	natsConfig env.NATSConfig,
	logger *logger.Logger,
	recorder record.EventRecorder,
) Manager {
	return EventingManager{
		ctx:        ctx,
		Client:     client,
		natsConfig: natsConfig,
		kubeClient: k8s.NewKubeClient(client),
		logger:     logger,
		recorder:   recorder,
	}
}

func (em EventingManager) CreateOrUpdatePublisherProxy(ctx context.Context, eventing *v1alpha1.Eventing) (*v1.Deployment, error) {
	ecBackendType, err := convertECBackendType(eventing.GetNATSBackend().Type)
	if err != nil {
		return nil, fmt.Errorf("failed to convert eventing controller backend type: %s", err)
	}

	if err = em.updateNatsConfig(ctx, &em.natsConfig, eventing); err != nil {
		return nil, err
	}

	backendConfig := env.GetBackendConfig()
	updatePublisherConfig(&backendConfig, eventing)

	// create an instance of ecbackend Reconciler
	ecReconciler := ecbackend.NewReconciler(
		ctx,
		nil,
		em.natsConfig,
		env.GetConfig(),
		backendConfig,
		nil,
		em.Client,
		em.logger,
		em.recorder,
	)

	deployment, err := ecReconciler.CreateOrUpdatePublisherProxy(ctx, ecBackendType)
	if err != nil {
		return nil, err
	}

	// Overwrite owner reference for publisher proxy deployment as the EC sets its deployment as owner
	// and we want the Eventing CR to be the owner.
	err = em.setDeploymentOwnerReference(ctx, deployment, eventing)
	if err != nil {
		return deployment, fmt.Errorf("failed to set owner reference for publisher proxy deployment: %s", err)
	}

	return deployment, nil
}

// createOrUpdateHorizontalPodAutoscaler creates or updates the HPA for the given deployment.
func (em EventingManager) CreateOrUpdateHPA(ctx context.Context, deployment *v1.Deployment, eventing *eventingv1alpha1.Eventing, cpuUtilization, memoryUtilization int32) error {
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
		if nats.Status.State == eventingv1alpha1.StateReady {
			return true, nil
		}
	}
	return false, nil
}

func (em *EventingManager) setDeploymentOwnerReference(ctx context.Context, deployment *v1.Deployment, eventing *eventingv1alpha1.Eventing) error {
	// Update the deployment object
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := em.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, deployment)
		if err != nil {
			return err
		}
		// Set the controller reference to the parent object
		if err := controllerutil.SetControllerReference(eventing, deployment, em.Scheme()); err != nil {
			return fmt.Errorf("failed to set controller reference: %v", err)
		}
		err = em.Update(ctx, deployment)
		if err != nil {
			return err
		}
		return nil
	})
	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (em *EventingManager) getNATSUrl(ctx context.Context, namespace string) (string, error) {
	natsList, err := em.kubeClient.GetNATSResources(ctx, namespace)
	if err != nil {
		return "", err
	}
	for _, nats := range natsList.Items {
		return fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", nats.Name, nats.Namespace, natsClientPort), nil
	}
	return "", fmt.Errorf("no NATS CR found to build NATS server URL")
}

func (em *EventingManager) updateNatsConfig(ctx context.Context, natsConfig *env.NATSConfig, eventing *v1alpha1.Eventing) error {
	natsUrl, err := em.getNATSUrl(ctx, eventing.Namespace)
	if err != nil {
		return err
	}
	natsConfig.URL = natsUrl
	natsConfig.JSStreamStorageType = eventing.Spec.Backends[0].Config.NATSStorageType
	natsConfig.JSStreamReplicas = eventing.Spec.Backends[0].Config.NATSStreamReplicas
	natsConfig.JSStreamMaxBytes = eventing.Spec.Backends[0].Config.MaxStreamSize.String()
	natsConfig.JSStreamMaxMsgsPerTopic = eventing.Spec.Backends[0].Config.MaxMsgsPerTopic
	return nil
}

func updatePublisherConfig(backendConfig *env.BackendConfig, eventing *v1alpha1.Eventing) {
	backendConfig.PublisherConfig.RequestsCPU = eventing.Spec.Publisher.Resources.Requests.Cpu().String()
	backendConfig.PublisherConfig.RequestsMemory = eventing.Spec.Publisher.Resources.Requests.Memory().String()
	backendConfig.PublisherConfig.LimitsCPU = eventing.Spec.Publisher.Resources.Limits.Cpu().String()
	backendConfig.PublisherConfig.LimitsMemory = eventing.Spec.Publisher.Resources.Limits.Memory().String()
	backendConfig.PublisherConfig.Replicas = int32(eventing.Spec.Min)
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
