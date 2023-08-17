package eventing

import (
	"context"
	"fmt"
	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	ecsubscriptionmanager "github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager"
	"go.uber.org/zap"
)

func (r *Reconciler) reconcileNATSSubManager(eventing *v1alpha1.Eventing, log *zap.SugaredLogger) error {
	if r.natsSubManager == nil {
		natsConfig, err := r.natsConfigHandler.GetNatsConfig(context.Background(), *eventing)
		if err != nil {
			return err
		}
		// create instance of NATS subscription manager
		natsSubManager := r.subManagerFactory.NewJetStreamManager(*eventing, *natsConfig)

		// init it
		if err := natsSubManager.Init(r.ctrlManager); err != nil {
			return err
		}

		log.Info("NATS subscription-manager initialized")
		// save instance only when init is successful.
		r.natsSubManager = natsSubManager
	}

	if r.isNATSSubManagerStarted {
		log.Info("NATS subscription-manager is already started")
		return nil
	}

	// start the subscription manager.
	defaultSubsConfig := r.eventingManager.GetBackendConfig().
		DefaultSubscriptionConfig.ToECENVDefaultSubscriptionConfig()

	if err := r.natsSubManager.Start(defaultSubsConfig, ecsubscriptionmanager.Params{}); err != nil {
		return err
	}

	log.Info("NATS subscription-manager started")
	// update flag so it do not try to start the manager again.
	r.isNATSSubManagerStarted = true

	return nil
}

func (r *Reconciler) stopNATSSubManager(runCleanup bool, log *zap.SugaredLogger) error {
	log.Debug("stopping NATS subscription-manager")
	if r.natsSubManager == nil || !r.isNATSSubManagerStarted {
		log.Info("NATS subscription-manager is already stopped!")
		return nil
	}

	// stop the subscription manager.
	if err := r.natsSubManager.Stop(runCleanup); err != nil {
		return err
	}

	log.Info("NATS subscription-manager stopped!")
	// update flags so it does not try to stop the manager again.
	r.isNATSSubManagerStarted = false
	r.natsSubManager = nil

	return nil
}

func NewNatsConfigHandler(
	kubeClient k8s.Client,
	opts *options.Options,
) NatsConfigHandler {
	return &NatsConfigHandlerImpl{
		kubeClient: kubeClient,
		opts:       opts,
	}
}

//go:generate mockery --name=NatsConfigHandler --outpkg=mocks --case=underscore
type NatsConfigHandler interface {
	GetNatsConfig(ctx context.Context, eventing v1alpha1.Eventing) (*env.NATSConfig, error)
}

type NatsConfigHandlerImpl struct {
	kubeClient k8s.Client
	opts       *options.Options
}

func (n *NatsConfigHandlerImpl) GetNatsConfig(ctx context.Context, eventing v1alpha1.Eventing) (*env.NATSConfig, error) {
	natsConfig, err := env.GetNATSConfig(n.opts.MaxReconnects, n.opts.ReconnectWait)
	if err != nil {
		return nil, err
	}
	// identifies the NATs server url and sets to natsConfig
	err = n.setUrlToNatsConfig(ctx, &eventing, &natsConfig)
	if err != nil {
		return nil, err
	}
	return &natsConfig, nil
}

func (n *NatsConfigHandlerImpl) setUrlToNatsConfig(ctx context.Context, eventing *v1alpha1.Eventing, natsConfig *env.NATSConfig) error {
	natsUrl, err := n.getNATSUrl(ctx, eventing.Namespace)
	if err != nil {
		return err
	}
	natsConfig.URL = natsUrl
	return nil
}

func (n *NatsConfigHandlerImpl) getNATSUrl(ctx context.Context, namespace string) (string, error) {
	natsList, err := n.kubeClient.GetNATSResources(ctx, namespace)
	if err != nil {
		return "", err
	}
	for _, nats := range natsList.Items {
		return fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", nats.Name, nats.Namespace, natsClientPort), nil
	}
	return "", fmt.Errorf("NATS CR is not found to build NATS server URL")
}
