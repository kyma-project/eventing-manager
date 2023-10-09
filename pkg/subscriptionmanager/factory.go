package subscriptionmanager

import (
	"time"

	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	ecmetrics "github.com/kyma-project/kyma/components/eventing-controller/pkg/backend/metrics"
	ecsubscriptionmanager "github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager"

	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/jetstream"
	eclogger "github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager/eventmesh"
	"k8s.io/client-go/rest"
)

// Perform a compile-time check.
var _ ManagerFactory = &Factory{}

//go:generate mockery --name=ManagerFactory --outpkg=mocks --case=underscore
//go:generate mockery --name=Manager --dir=../../vendor/github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager --outpkg=mocks --output=mocks/ec --case=underscore
type ManagerFactory interface {
	NewJetStreamManager(v1alpha1.Eventing, env.NATSConfig) manager.Manager
	NewEventMeshManager() (ecsubscriptionmanager.Manager, error)
}

type Factory struct {
	k8sRestCfg       *rest.Config
	metricsAddress   string
	metricsCollector *metrics.Collector
	resyncPeriod     time.Duration
	logger           *eclogger.Logger
}

func NewFactory(
	k8sRestCfg *rest.Config,
	metricsAddress string,
	metricsCollector *metrics.Collector,
	resyncPeriod time.Duration,
	logger *eclogger.Logger) *Factory {
	return &Factory{
		k8sRestCfg:       k8sRestCfg,
		metricsAddress:   metricsAddress,
		metricsCollector: metricsCollector,
		resyncPeriod:     resyncPeriod,
		logger:           logger,
	}
}

func (f Factory) NewJetStreamManager(eventing v1alpha1.Eventing, natsConfig env.NATSConfig) manager.Manager {
	return jetstream.NewSubscriptionManager(f.k8sRestCfg, natsConfig.GetNewNATSConfig(eventing),
		f.metricsAddress, f.metricsCollector, f.logger)
}

func (f Factory) NewEventMeshManager() (ecsubscriptionmanager.Manager, error) {
	// TODO: fix the metrics object from f.metricsCollector
	mc := ecmetrics.NewCollector()
	return eventmesh.NewSubscriptionManager(f.k8sRestCfg, f.metricsAddress, f.resyncPeriod, f.logger, mc), nil
}
