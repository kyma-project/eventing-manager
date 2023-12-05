package eventing

import (
	"context"
	"fmt"

	"github.com/mitchellh/hashstructure/v2"
	kctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
)

func (r *Reconciler) containsFinalizer(eventing *operatorv1alpha1.Eventing) bool {
	return controllerutil.ContainsFinalizer(eventing, FinalizerName)
}

func (r *Reconciler) addFinalizer(ctx context.Context, eventing *operatorv1alpha1.Eventing) (kctrl.Result, error) {
	controllerutil.AddFinalizer(eventing, FinalizerName)
	if err := r.Update(ctx, eventing); err != nil {
		return kctrl.Result{}, err
	}
	return kctrl.Result{}, nil
}

func (r *Reconciler) removeFinalizer(ctx context.Context, eventing *operatorv1alpha1.Eventing) (kctrl.Result, error) {
	controllerutil.RemoveFinalizer(eventing, FinalizerName)
	if err := r.Update(ctx, eventing); err != nil {
		return kctrl.Result{}, err
	}

	return kctrl.Result{}, nil
}

func (r *Reconciler) getNATSBackendConfigHash(defaultSubscriptionConfig env.DefaultSubscriptionConfig, natsConfig env.NATSConfig) (int64, error) {
	natsBackendConfig := struct {
		env.DefaultSubscriptionConfig
		env.NATSConfig
	}{defaultSubscriptionConfig, natsConfig}
	hash, err := hashstructure.Hash(natsBackendConfig, hashstructure.FormatV2, nil)
	if err != nil {
		return 0, err
	}
	return int64(hash), nil
}

func getEventMeshBackendConfigHash(eventMeshSecret, eventTypePrefix, domain string) (int64, error) {
	eventMeshBackendConfig := fmt.Sprintf("[%s][%s][%s]", eventMeshSecret, eventTypePrefix, domain)
	hash, err := hashstructure.Hash(eventMeshBackendConfig, hashstructure.FormatV2, nil)
	if err != nil {
		return 0, err
	}
	return int64(hash), nil
}
