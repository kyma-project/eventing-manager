package eventing

import (
	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	ecsubscriptionmanager "github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager"
	"go.uber.org/zap"
)

func (r *Reconciler) reconcileNATSSubManager(eventing *v1alpha1.Eventing, log *zap.SugaredLogger) error {
	if r.natsSubManager == nil {
		// create instance of NATS subscription manager
		natsSubManager := r.subManagerFactory.NewJetStreamManager(*eventing, r.eventingManager.GetNATSConfig())

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
