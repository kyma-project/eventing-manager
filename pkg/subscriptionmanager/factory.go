package subscriptionmanager

import (
	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	eclogger "github.com/kyma-project/kyma/components/eventing-controller/logger"
	ecbackendmetrics "github.com/kyma-project/kyma/components/eventing-controller/pkg/backend/metrics"
	ecsubscriptionmanager "github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager/eventmesh"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager/jetstream"
	"k8s.io/client-go/rest"
	"os"
	"time"
)

// Perform a compile-time check.
var _ ManagerFactory = &Factory{}

//go:generate mockery --name=ManagerFactory --outpkg=mocks --case=underscore
//go:generate mockery --name=Manager --dir=../../vendor/github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager --outpkg=mocks --output=mocks/ec --case=underscore
type ManagerFactory interface {
	NewJetStreamManager(v1alpha1.Eventing, env.NATSConfig) ecsubscriptionmanager.Manager
	NewEventMeshManager(v1alpha1.Eventing) (ecsubscriptionmanager.Manager, error)
}

type Factory struct {
	k8sRestCfg       *rest.Config
	metricsAddress   string
	metricsCollector *ecbackendmetrics.Collector
	resyncPeriod     time.Duration
	logger           *eclogger.Logger
}

func NewFactory(
	k8sRestCfg *rest.Config,
	metricsAddress string,
	metricsCollector *ecbackendmetrics.Collector,
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

func (f Factory) NewJetStreamManager(eventing v1alpha1.Eventing, natsConfig env.NATSConfig) ecsubscriptionmanager.Manager {
	return jetstream.NewSubscriptionManager(f.k8sRestCfg, natsConfig.ToECENVNATSConfig(eventing),
		f.metricsAddress, f.metricsCollector, f.logger)
}

func (f Factory) NewEventMeshManager(eventing v1alpha1.Eventing) (ecsubscriptionmanager.Manager, error) {
	if err := setUpEventMeshSubManagerEnvironment(eventing); err != nil {
		return nil, err
	}
	return eventmesh.NewSubscriptionManager(f.k8sRestCfg, f.metricsAddress, f.resyncPeriod, f.logger), nil
}

func setUpEventMeshSubManagerEnvironment(eventing v1alpha1.Eventing) error {
	// Set the EVENT_TYPE_PREFIX env variable as it is read in the NewSubscriptionManager() function.
	if err := os.Setenv("EVENT_TYPE_PREFIX", eventing.Spec.Backend.Config.EventTypePrefix); err != nil {
		return err
	}
	return nil
}
