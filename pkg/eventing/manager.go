package eventing

import (
	"context"
	"fmt"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
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
	DeployPublisherProxy(
		ctx context.Context,
		eventing *v1alpha1.Eventing,
		natsConfig *env.NATSConfig,
		backendType v1alpha1.BackendType) (*appsv1.Deployment, error)
	DeployPublisherProxyResources(context.Context, *v1alpha1.Eventing, *appsv1.Deployment) error
	GetBackendConfig() *env.BackendConfig
	SetBackendConfig(env.BackendConfig)
}

type EventingManager struct {
	ctx context.Context
	client.Client
	backendConfig env.BackendConfig
	kubeClient    k8s.Client
	logger        *logger.Logger
	recorder      record.EventRecorder
}

func NewEventingManager(
	ctx context.Context,
	client client.Client,
	kubeClient k8s.Client,
	backendConfig env.BackendConfig,
	logger *logger.Logger,
	recorder record.EventRecorder,
) Manager {
	return &EventingManager{
		ctx:           ctx,
		Client:        client,
		backendConfig: backendConfig,
		kubeClient:    kubeClient,
		logger:        logger,
		recorder:      recorder,
	}
}

func (em EventingManager) DeployPublisherProxy(
	ctx context.Context,
	eventing *v1alpha1.Eventing,
	natsConfig *env.NATSConfig,
	backendType v1alpha1.BackendType) (*appsv1.Deployment, error) {
	// update EC reconciler NATS and public config from the data in the eventing CR
	deployment, err := em.applyPublisherProxyDeployment(ctx, eventing, natsConfig, backendType)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (em *EventingManager) applyPublisherProxyDeployment(
	ctx context.Context,
	eventing *v1alpha1.Eventing,
	natsConfig *env.NATSConfig,
	backendType v1alpha1.BackendType) (*appsv1.Deployment, error) {
	var desiredPublisher *appsv1.Deployment

	switch backendType {
	case v1alpha1.NatsBackendType:
		desiredPublisher = newNATSPublisherDeployment(eventing, *natsConfig, em.backendConfig.PublisherConfig)
	case v1alpha1.EventMeshBackendType:
		desiredPublisher = newEventMeshPublisherDeployment(eventing, em.backendConfig.PublisherConfig)
	default:
		return nil, fmt.Errorf("unknown EventingBackend type %q", backendType)
	}

	if err := controllerutil.SetControllerReference(eventing, desiredPublisher, em.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %v", err)
	}

	currentPublisher, err := em.kubeClient.GetDeployment(ctx, GetPublisherDeploymentName(*eventing), eventing.Namespace)
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

		// if a publisher deploy from eventing-controller exists, then update it.
		if err := em.migratePublisherDeploymentFromEC(ctx, eventing, *currentPublisher, *desiredPublisher); err != nil {
			return nil, fmt.Errorf("failed to migrate publisher: %v", err)
		}
	}
	// Update publisher proxy deployment
	if err := em.kubeClient.PatchApply(ctx, desiredPublisher); err != nil {
		return nil, fmt.Errorf("failed to apply Publisher Proxy deployment: %v", err)
	}

	return desiredPublisher, nil
}

func (em *EventingManager) migratePublisherDeploymentFromEC(
	ctx context.Context, eventing *v1alpha1.Eventing,
	currentPublisher appsv1.Deployment, desiredPublisher appsv1.Deployment) error {
	// If Eventing CR is already owner of deployment, then it means that the publisher deployment
	// was already migrated.
	if len(currentPublisher.OwnerReferences) == 1 && currentPublisher.OwnerReferences[0].Name == eventing.Name {
		return nil
	}

	em.logger.WithContext().Info("migrating publisher deployment from eventing-controller to Eventing CR")
	updatedPublisher := currentPublisher.DeepCopy()
	// change OwnerReference to Eventing CR.
	updatedPublisher.OwnerReferences = nil
	if err := controllerutil.SetControllerReference(eventing, updatedPublisher, em.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference: %v", err)
	}
	// copy Spec from desired publisher
	// because some ENV variables conflicts with server-side patch apply.
	updatedPublisher.Spec = desiredPublisher.Spec

	// update the publisher deployment.
	return em.kubeClient.UpdateDeployment(ctx, updatedPublisher)
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

func (em EventingManager) GetBackendConfig() *env.BackendConfig {
	return &em.backendConfig
}

func (em *EventingManager) SetBackendConfig(config env.BackendConfig) {
	em.backendConfig = config
}

func (em EventingManager) DeployPublisherProxyResources(
	ctx context.Context,
	eventing *v1alpha1.Eventing,
	publisherDeployment *appsv1.Deployment) error {
	// define list of resources to create for EPP.
	resources := []client.Object{
		// ServiceAccount
		newPublisherProxyServiceAccount(GetPublisherServiceAccountName(*eventing), eventing.Namespace, publisherDeployment.Labels),
		// ClusterRole
		newPublisherProxyClusterRole(GetPublisherClusterRoleName(*eventing), eventing.Namespace, publisherDeployment.Labels),
		// ClusterRoleBinding
		newPublisherProxyClusterRoleBinding(GetPublisherClusterRoleBindingName(*eventing), eventing.Namespace,
			publisherDeployment.Labels),
		// Service to expose event publishing endpoint of EPP.
		newPublisherProxyService(GetPublisherPublishServiceName(*eventing), eventing.Namespace, publisherDeployment.Labels,
			publisherDeployment.Spec.Template.Labels),
		// Service to expose metrics endpoint of EPP.
		newPublisherProxyMetricsService(GetPublisherMetricsServiceName(*eventing), eventing.Namespace, publisherDeployment.Labels,
			publisherDeployment.Spec.Template.Labels),
		// Service to expose health endpoint of EPP.
		newPublisherProxyHealthService(GetPublisherHealthServiceName(*eventing), eventing.Namespace, publisherDeployment.Labels,
			publisherDeployment.Spec.Template.Labels),
		// HPA to auto-scale publisher proxy.
		newHorizontalPodAutoscaler(publisherDeployment, int32(eventing.Spec.Publisher.Min),
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
