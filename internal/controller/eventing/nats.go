package eventing

import (
	"context"
	"fmt"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
)

const natsClientPort = 4222

// retrieves all NATS CRs in the given namespace and checks if any of them is in ready state.
// Normally, there should be only one NATS CR in the namespace. More than is not supported.
func (r *Reconciler) isNATSAvailable(ctx context.Context, namespace string) (bool, error) {
	natsList, err := r.kubeClient.GetNATSResources(ctx, namespace)
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

func (r *Reconciler) GetNATSUrl(ctx context.Context, namespace string) (string, error) {
	natsList, err := r.kubeClient.GetNATSResources(ctx, namespace)
	if err != nil {
		return "", err
	}
	for _, nats := range natsList.Items {
		return fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", nats.Name, nats.Namespace, natsClientPort), nil
	}
	return "", fmt.Errorf("no NATS CR found to build NATS server URL")
}
