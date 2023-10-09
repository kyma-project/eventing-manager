package eventing

import (
	"context"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	ecenv "github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	"github.com/mitchellh/hashstructure/v2"
	v1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) containsFinalizer(eventing *eventingv1alpha1.Eventing) bool {
	return controllerutil.ContainsFinalizer(eventing, FinalizerName)
}

func (r *Reconciler) addFinalizer(ctx context.Context, eventing *eventingv1alpha1.Eventing) (ctrl.Result, error) {
	controllerutil.AddFinalizer(eventing, FinalizerName)
	if err := r.Update(ctx, eventing); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) removeFinalizer(ctx context.Context, eventing *eventingv1alpha1.Eventing) (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(eventing, FinalizerName)
	if err := r.Update(ctx, eventing); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) getNATSBackendConfigHash(defaultSubscriptionConfig ecenv.DefaultSubscriptionConfig, natsConfig env.NATSConfig) (uint64, error) {
	natsBackendConfig := struct {
		ecenv.DefaultSubscriptionConfig
		env.NATSConfig
	}{defaultSubscriptionConfig, natsConfig}
	hash, err := hashstructure.Hash(natsBackendConfig, hashstructure.FormatV2, nil)
	if err != nil {
		return 0, err
	}
	return hash, nil
}

func (r *Reconciler) getEventMeshBackendConfigHash(eventMeshSecret *v1.Secret) (uint64, error) {
	eventMeshBackendConfig := struct {
		*v1.Secret
		oauth2Credentials
	}{eventMeshSecret, r.oauth2credentials}

	hash, err := hashstructure.Hash(eventMeshBackendConfig, hashstructure.FormatV2, nil)
	if err != nil {
		return 0, err
	}
	return hash, nil
}

func (r *Reconciler) getDefaultSubscriptionConfig() ecenv.DefaultSubscriptionConfig {
	return r.eventingManager.GetBackendConfig().
		DefaultSubscriptionConfig.ToECENVDefaultSubscriptionConfig()
}
