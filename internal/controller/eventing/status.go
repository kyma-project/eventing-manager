package eventing

import (
	"context"
	"errors"
	"time"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stype "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const RequeueTimeForStatusCheck = 10

// InitStateProcessing initializes the state of the EventingStatus if it is not set
func (es *Reconciler) InitStateProcessing(eventing *eventingv1alpha1.Eventing) {
	if eventing.Status.State == "" {
		eventing.Status.SetStateProcessing()
	}
}

// syncStatusWithNATSErr syncs Eventing status and sets an error state.
// Returns the relevant error.
func (r *Reconciler) syncStatusWithNATSErr(ctx context.Context,
	eventing *eventingv1alpha1.Eventing, err error, log *zap.SugaredLogger) error {
	// Set error state in status
	eventing.Status.SetStateError()
	eventing.Status.UpdateConditionNATSAvailable(metav1.ConditionFalse, eventingv1alpha1.ConditionReasonNATSNotAvailable,
		err.Error())

	return errors.Join(err, r.syncEventingStatus(ctx, eventing, log))
}

// syncStatusWithPublisherProxyErr updates Publisher Proxy condition and sets an error state.
// Returns the relevant error.
func (r *Reconciler) syncStatusWithPublisherProxyErr(ctx context.Context,
	eventing *eventingv1alpha1.Eventing, err error, log *zap.SugaredLogger) error {
	return r.syncStatusWithPublisherProxyErrWithReason(ctx, eventingv1alpha1.ConditionReasonDeployedFailed,
		eventing, err, log)
}

func (r *Reconciler) syncStatusWithPublisherProxyErrWithReason(ctx context.Context,
	reason eventingv1alpha1.ConditionReason,
	eventing *eventingv1alpha1.Eventing, err error, log *zap.SugaredLogger) error {
	// Set error state in status
	eventing.Status.SetStateError()
	eventing.Status.UpdateConditionPublisherProxyReady(metav1.ConditionFalse, reason,
		err.Error())

	return errors.Join(err, r.syncEventingStatus(ctx, eventing, log))
}

// syncStatusWithSubscriptionManagerErr updates subscription manager condition and sets an error state.
// Returns the relevant error.
func (r *Reconciler) syncStatusWithSubscriptionManagerErr(ctx context.Context,
	eventing *eventingv1alpha1.Eventing, err error, log *zap.SugaredLogger) error {
	return r.syncStatusWithSubscriptionManagerErrWithReason(ctx,
		eventingv1alpha1.ConditionReasonEventMeshSubManagerFailed, eventing, err, log)
}

func (r *Reconciler) syncStatusWithSubscriptionManagerErrWithReason(ctx context.Context,
	reason eventingv1alpha1.ConditionReason,
	eventing *eventingv1alpha1.Eventing,
	err error, log *zap.SugaredLogger) error {
	// Set error state in status
	eventing.Status.SetStateError()
	eventing.Status.UpdateConditionSubscriptionManagerReady(metav1.ConditionFalse, reason, err.Error())
	return errors.Join(err, r.syncEventingStatus(ctx, eventing, log))
}

// syncStatusWithSubscriptionManagerFailedCondition updates subscription manager condition and
// sets an error state. It doesn't return the incoming error.
func (r *Reconciler) syncStatusWithSubscriptionManagerFailedCondition(ctx context.Context,
	eventing *eventingv1alpha1.Eventing,
	err error, log *zap.SugaredLogger) error {
	// Set error state in status
	eventing.Status.SetStateError()
	eventing.Status.UpdateConditionSubscriptionManagerReady(metav1.ConditionFalse,
		eventingv1alpha1.ConditionReasonEventMeshSubManagerFailed, err.Error())
	return r.syncEventingStatus(ctx, eventing, log)
}

func (r *Reconciler) syncStatusWithWebhookErr(ctx context.Context,
	eventing *eventingv1alpha1.Eventing, err error, log *zap.SugaredLogger) error {
	// Set error state in status
	eventing.Status.SetStateError()
	eventing.Status.UpdateConditionWebhookReady(metav1.ConditionFalse, eventingv1alpha1.ConditionReasonWebhookFailed,
		err.Error())

	return errors.Join(err, r.syncEventingStatus(ctx, eventing, log))
}

// syncEventingStatus syncs Eventing status.
func (r *Reconciler) syncEventingStatus(ctx context.Context,
	eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) error {
	namespacedName := &k8stype.NamespacedName{
		Name:      eventing.Name,
		Namespace: eventing.Namespace,
	}

	// Fetch the latest Eventing object, to avoid k8s conflict errors.
	actualEventing := &eventingv1alpha1.Eventing{}
	if err := r.Client.Get(ctx, *namespacedName, actualEventing); err != nil {
		return err
	}

	// Copy new changes to the latest object
	desiredEventing := actualEventing.DeepCopy()
	desiredEventing.Status = eventing.Status

	// Sync Eventing resource status with k8s
	return r.updateStatus(ctx, actualEventing, desiredEventing, log)
}

// updateStatus updates the status to k8s if modified.
func (r *Reconciler) updateStatus(ctx context.Context, oldEventing, newEventing *eventingv1alpha1.Eventing,
	logger *zap.SugaredLogger) error {
	// Compare the status taking into consideration lastTransitionTime in conditions
	if oldEventing.Status.IsEqual(newEventing.Status) {
		return nil
	}

	// Update the status for Eventing resource
	if err := r.Status().Update(ctx, newEventing); err != nil {
		return err
	}

	logger.Debugw("Updated Eventing status",
		"oldStatus", oldEventing.Status, "newStatus", newEventing.Status)

	return nil
}

func (r *Reconciler) handleEventingState(ctx context.Context, deployment *v1.Deployment, eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) (ctrl.Result, error) {
	// checking if publisher proxy is ready.
	// get k8s deployment for publisher proxy
	deployment, err := r.kubeClient.GetDeployment(ctx, deployment.Name, deployment.Namespace)
	if err != nil {
		eventing.Status.UpdateConditionPublisherProxyReady(metav1.ConditionFalse,
			eventingv1alpha1.ConditionReasonDeploymentStatusSyncFailed, err.Error())
		return ctrl.Result{}, r.syncStatusWithPublisherProxyErr(ctx, eventing, err, log)
	}

	if !IsDeploymentReady(deployment) {
		eventing.Status.SetStateProcessing()
		eventing.Status.UpdateConditionPublisherProxyReady(metav1.ConditionFalse,
			eventingv1alpha1.ConditionReasonProcessing, eventingv1alpha1.ConditionPublisherProxyProcessingMessage)
		log.Info("Reconciliation successful: waiting for publisher proxy to get ready...")
		return ctrl.Result{RequeueAfter: RequeueTimeForStatusCheck * time.Second}, r.syncEventingStatus(ctx, eventing, log)
	}
	//
	eventing.Status.SetPublisherProxyReadyToTrue()

	// @TODO: emit events for any change in conditions
	log.Info("Reconciliation successful")
	return ctrl.Result{}, r.syncEventingStatus(ctx, eventing, log)
}

// to be able to mock this function in tests
var IsDeploymentReady = func(deployment *v1.Deployment) bool {
	return deployment.Status.AvailableReplicas == *deployment.Spec.Replicas
}
