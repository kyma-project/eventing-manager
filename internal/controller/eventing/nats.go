package eventing

import (
	"context"
	"fmt"

	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/options"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"go.uber.org/zap"
)

func (r *Reconciler) reconcileNATSSubManager(ctx context.Context, eventing *v1alpha1.Eventing, log *zap.SugaredLogger) error {
	// get the subscription config
	defaultSubsConfig := r.getDefaultSubscriptionConfig()
	// get the nats config
	natsConfig, err := r.natsConfigHandler.GetNatsConfig(context.Background(), *eventing)
	if err != nil {
		return err
	}
	// get the hash of current config
	specHash, err := r.getNATSBackendConfigHash(defaultSubsConfig, *natsConfig)
	if err != nil {
		return err
	}

	// update the config if hashes differ
	if eventing.Status.BackendConfigHash != specHash && r.isNATSSubManagerStarted {
		log.Infof("specHash does not match, old hash: %v, new hash: %v", eventing.Status.BackendConfigHash, specHash)
		// stop the subsManager without cleanup
		if err := r.stopNATSSubManager(false, log); err != nil {
			return err
		}
	}

	if r.natsSubManager == nil {
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

	err = r.startNATSSubManager(defaultSubsConfig, log)
	if err != nil {
		return err
	}

	// update the hash of the current config only once subManager is started
	eventing.Status.BackendConfigHash = specHash
	log.Info(fmt.Sprintf("NATS subscription-manager has been updated, new hash: %d", specHash))

	return nil
}

func (r *Reconciler) startNATSSubManager(defaultSubsConfig env.DefaultSubscriptionConfig, log *zap.SugaredLogger) error {
	if err := r.natsSubManager.Start(defaultSubsConfig, manager.Params{}); err != nil {
		return err
	}

	log.Info("NATS subscription-manager started")
	// update flag so it do not try to start the manager again.
	r.isNATSSubManagerStarted = true
	return nil
}

func (r *Reconciler) getDefaultSubscriptionConfig() env.DefaultSubscriptionConfig {
	return r.eventingManager.GetBackendConfig().
		DefaultSubscriptionConfig
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

//go:generate go run github.com/vektra/mockery/v2 --name=NatsConfigHandler --outpkg=mocks --case=underscore
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
	natsConfig.JSStreamStorageType = eventing.Spec.Backend.Config.NATSStreamStorageType
	natsConfig.JSStreamReplicas = eventing.Spec.Backend.Config.NATSStreamReplicas
	natsConfig.JSStreamMaxBytes = eventing.Spec.Backend.Config.NATSStreamMaxSize.String()
	natsConfig.JSStreamMaxMsgsPerTopic = int64(eventing.Spec.Backend.Config.NATSMaxMsgsPerTopic)
	natsConfig.EventTypePrefix = eventing.Spec.Backend.Config.EventTypePrefix
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
