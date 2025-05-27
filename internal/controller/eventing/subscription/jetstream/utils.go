package jetstream

import (
	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/utils"
)

// isInDeletion checks if the subscription needs to be deleted.
func isInDeletion(subscription *eventingv1alpha2.Subscription) bool {
	return !subscription.DeletionTimestamp.IsZero()
}

// containsFinalizer checks if the subscription contains our Finalizer.
func containsFinalizer(sub *eventingv1alpha2.Subscription) bool {
	return utils.ContainsString(sub.Finalizers, eventingv1alpha2.Finalizer)
}
