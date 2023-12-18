package v1alpha1

import (
	kctrl "sigs.k8s.io/controller-runtime"
)

func (r *Subscription) SetupWebhookWithManager(mgr kctrl.Manager) error {
	return kctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}
