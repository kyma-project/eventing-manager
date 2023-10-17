package subscriptionmanager

import (
	"time"

	"github.com/kyma-project/eventing-manager/pkg/apigateway"

	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	eclogger "github.com/kyma-project/eventing-manager/pkg/logger"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/eventmesh"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/jetstream"
	"k8s.io/client-go/rest"
)

// Perform a compile-time check.
var _ ManagerFactory = &Factory{}

//go:generate mockery --name=ManagerFactory --outpkg=mocks --case=underscore
type ManagerFactory interface {
	NewJetStreamManager(v1alpha1.Eventing, env.NATSConfig) manager.Manager
	NewEventMeshManager() (manager.Manager, error)
}

type Factory struct {
	k8sRestCfg       *rest.Config
	metricsAddress   string
	metricsCollector *metrics.Collector
	resyncPeriod     time.Duration
	apiGateway       apigateway.APIGateway
	logger           *eclogger.Logger
}

func NewFactory(
	k8sRestCfg *rest.Config,
	metricsAddress string,
	metricsCollector *metrics.Collector,
	resyncPeriod time.Duration,
	apiGateway apigateway.APIGateway,
	logger *eclogger.Logger) *Factory {
	return &Factory{
		k8sRestCfg:       k8sRestCfg,
		metricsAddress:   metricsAddress,
		metricsCollector: metricsCollector,
		resyncPeriod:     resyncPeriod,
		apiGateway:       apiGateway,
		logger:           logger,
	}
}

func (f Factory) NewJetStreamManager(eventing v1alpha1.Eventing, natsConfig env.NATSConfig) manager.Manager {
	return jetstream.NewSubscriptionManager(f.k8sRestCfg, natsConfig.GetNewNATSConfig(eventing),
		f.metricsAddress, f.metricsCollector, f.logger)
}

func (f Factory) NewEventMeshManager() (manager.Manager, error) {
	return eventmesh.NewSubscriptionManager(f.k8sRestCfg, f.metricsAddress, f.resyncPeriod,
		f.logger, f.metricsCollector, f.apiGateway), nil
}
