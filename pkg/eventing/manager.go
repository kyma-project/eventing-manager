package eventing

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	kappsv1 "k8s.io/api/apps/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha1"
	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	"github.com/kyma-project/eventing-manager/pkg/object"
)

const (
	cpuUtilization    = 60
	memoryUtilization = 60
)

// allowedAnnotations are the publisher proxy deployment spec template annotations
// which should be preserved during reconciliation.
var allowedAnnotations = map[string]string{
	"kubectl.kubernetes.io/restartedAt": "",
}

var (
	ErrUnknownBackendType = errors.New("unknown backend type")
	ErrEPPDeployFailed    = errors.New("failed to apply Publisher Proxy deployment")
)

//go:generate go run github.com/vektra/mockery/v2 --name=Manager --outpkg=mocks --case=underscore
type Manager interface {
	IsNATSAvailable(ctx context.Context, namespace string) (bool, error)
	DeployPublisherProxy(
		ctx context.Context,
		eventing *v1alpha1.Eventing,
		natsConfig *env.NATSConfig,
		backendType v1alpha1.BackendType) (*kappsv1.Deployment, error)
	DeployPublisherProxyResources(context.Context, *v1alpha1.Eventing, *kappsv1.Deployment) error
	DeletePublisherProxyResources(ctx context.Context, eventing *v1alpha1.Eventing) error
	GetBackendConfig() *env.BackendConfig
	SetBackendConfig(env.BackendConfig)
	SubscriptionExists(ctx context.Context) (bool, error)
}

type EventingManager struct {
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
	backendType v1alpha1.BackendType,
) (*kappsv1.Deployment, error) {
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
	backendType v1alpha1.BackendType,
) (*kappsv1.Deployment, error) {
	var desiredPublisher *kappsv1.Deployment

	switch backendType {
	case v1alpha1.NatsBackendType:
		desiredPublisher = newNATSPublisherDeployment(eventing, *natsConfig, em.backendConfig.PublisherConfig)
	case v1alpha1.EventMeshBackendType:
		desiredPublisher = newEventMeshPublisherDeployment(eventing, em.backendConfig.PublisherConfig)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownBackendType, backendType)
	}

	if err := controllerutil.SetControllerReference(eventing, desiredPublisher, em.Scheme()); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	currentPublisher, err := em.kubeClient.GetDeployment(ctx, GetPublisherDeploymentName(*eventing), eventing.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get Event Publisher deployment: %w", err)
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
			return nil, fmt.Errorf("failed to migrate publisher: %w", err)
		}
	}

	if object.Semantic.DeepEqual(currentPublisher, desiredPublisher) {
		em.logger.WithContext().Debug(
			"skip updating the EPP deployment because its desired and actual states are equal",
		)
		return currentPublisher, nil
	}

	// Update publisher proxy deployment
	if err := em.kubeClient.PatchApply(ctx, desiredPublisher); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEPPDeployFailed, err)
	}

	return desiredPublisher, nil
}

func (em *EventingManager) migratePublisherDeploymentFromEC(
	ctx context.Context, eventing *v1alpha1.Eventing,
	currentPublisher kappsv1.Deployment, desiredPublisher kappsv1.Deployment,
) error {
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
		return fmt.Errorf("failed to set controller reference: %w", err)
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
		if nats.Status.State == v1alpha1.StateReady || nats.Status.State == v1alpha1.StateWarning {
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
	publisherDeployment *kappsv1.Deployment,
) error {
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
		newHorizontalPodAutoscaler(publisherDeployment.Name, publisherDeployment.Namespace, int32(eventing.Spec.Publisher.Min),
			int32(eventing.Spec.Publisher.Max), cpuUtilization, memoryUtilization, publisherDeployment.Labels),
	}

	// create the resources on k8s.
	for _, obj := range resources {
		// add owner reference.
		if err := controllerutil.SetControllerReference(eventing, obj, em.Scheme()); err != nil {
			return err
		}

		// patch apply the object.
		if err := em.kubeClient.PatchApply(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func (em EventingManager) DeletePublisherProxyResources(ctx context.Context, eventing *v1alpha1.Eventing) error {
	// define list of resources to delete for EPP.
	publisherDeployment := &kappsv1.Deployment{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      GetPublisherDeploymentName(*eventing),
			Namespace: eventing.Namespace,
		},
	}

	resources := []client.Object{
		// Deployment
		publisherDeployment,
		// ServiceAccount
		newPublisherProxyServiceAccount(GetPublisherServiceAccountName(*eventing), eventing.Namespace, map[string]string{}),
		// Service to expose event publishing endpoint of EPP.
		newPublisherProxyService(GetPublisherPublishServiceName(*eventing), eventing.Namespace, map[string]string{}, map[string]string{}),
		// Service to expose metrics endpoint of EPP.
		newPublisherProxyMetricsService(GetPublisherMetricsServiceName(*eventing), eventing.Namespace, map[string]string{}, map[string]string{}),
		// Service to expose health endpoint of EPP.
		newPublisherProxyHealthService(GetPublisherHealthServiceName(*eventing), eventing.Namespace, map[string]string{}, map[string]string{}),
		// HPA to auto-scale publisher proxy.
		newHorizontalPodAutoscaler(publisherDeployment.Name, eventing.Namespace, 0, 0, 0, 0, map[string]string{}),
	}

	// delete the resources on k8s.
	for _, obj := range resources {
		// delete the object.
		if err := em.kubeClient.DeleteResource(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func (em *EventingManager) SubscriptionExists(ctx context.Context) (bool, error) {
	subscriptionList, err := em.kubeClient.GetSubscriptions(ctx)
	if err != nil {
		return false, err
	}
	if len(subscriptionList.Items) > 0 {
		return true, nil
	}
	return false, nil
}

func convertECBackendType(backendType v1alpha1.BackendType) (eventingv1alpha1.BackendType, error) {
	switch backendType {
	case v1alpha1.EventMeshBackendType:
		return eventingv1alpha1.BEBBackendType, nil
	case v1alpha1.NatsBackendType:
		return eventingv1alpha1.NatsBackendType, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownBackendType, backendType)
	}
}
