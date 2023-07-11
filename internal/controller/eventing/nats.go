package eventing

import (
	"context"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// retrieves all NATS CRs in the given namespace and checks if any of them is in ready state.
// Normally, there should be only one NATS CR in the namespace. More than is not supported.
func (r *Reconciler) isNATSAvailable(ctx context.Context, namespace string) (bool, error) {
	natsList := &natsv1alpha1.NATSList{}
	err := r.List(ctx, natsList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return false, err
	}
	for _, nats := range natsList.Items {
		if nats.Status.State == "Ready" {
			return true, nil
		}
	}
	return false, nil
}
