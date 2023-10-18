package eventmesh

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/kyma-project/eventing-manager/pkg/apigateway"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	apigatewayv1beta1 "github.com/kyma-incubator/api-gateway/api/v1beta1"
	recerrors "github.com/kyma-project/eventing-manager/internal/controller/errors"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/object"
	"golang.org/x/xerrors"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kyma-project/eventing-manager/pkg/backend/eventmesh"
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"

	"go.uber.org/zap"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/pkg/backend/sink"
	backendutils "github.com/kyma-project/eventing-manager/pkg/backend/utils"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/logger"
)

type syncConditionWebhookCallStatusFunc func(subscription *eventingv1alpha2.Subscription)

// Reconciler reconciles a Subscription object.
type Reconciler struct {
	ctx context.Context
	client.Client
	logger            *logger.Logger
	recorder          record.EventRecorder
	Backend           eventmesh.Backend
	Domain            string
	cleaner           cleaner.Cleaner
	oauth2credentials *eventmesh.OAuth2ClientCredentials
	// nameMapper is used to map the Kyma subscription name to a subscription name on EventMesh.
	nameMapper                     backendutils.NameMapper
	sinkValidator                  sink.Validator
	collector                      *metrics.Collector
	syncConditionWebhookCallStatus syncConditionWebhookCallStatusFunc
	apiGateway                     apigateway.APIGateway
}

const (
	suffixLength                = 10
	externalHostPrefix          = "web"
	externalSinkScheme          = "https"
	apiRuleNamePrefix           = "webhook-"
	reconcilerName              = "eventMesh-subscription-reconciler"
	timeoutRetryActiveEmsStatus = time.Second * 30
	requeueAfterDuration        = time.Second * 2
	backendType                 = "EventMesh"
)

func NewReconciler(ctx context.Context,
	client client.Client,
	logger *logger.Logger,
	recorder record.EventRecorder,
	cfg env.Config, cleaner cleaner.Cleaner,
	eventMeshBackend eventmesh.Backend,
	credential *eventmesh.OAuth2ClientCredentials,
	apiGateway apigateway.APIGateway,
	mapper backendutils.NameMapper,
	validator sink.Validator,
	collector *metrics.Collector) *Reconciler {
	if err := eventMeshBackend.Initialize(cfg); err != nil {
		logger.WithContext().Errorw("Failed to start reconciler", "name",
			reconcilerName, "error", err)
		panic(err)
	}
	return &Reconciler{
		ctx:                            ctx,
		Client:                         client,
		logger:                         logger,
		recorder:                       recorder,
		Backend:                        eventMeshBackend,
		Domain:                         cfg.Domain,
		cleaner:                        cleaner,
		oauth2credentials:              credential,
		nameMapper:                     mapper,
		sinkValidator:                  validator,
		collector:                      collector,
		syncConditionWebhookCallStatus: syncConditionWebhookCallStatus,
		apiGateway:                     apiGateway,
	}
}

// +kubebuilder:rbac:groups=eventing.kyma-project.io,resources=subscriptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventing.kyma-project.io,resources=subscriptions/status,verbs=get;update;patch
// Generate required RBAC to emit kubernetes events in the controller.
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=gateway.kyma-project.io,resources=apirules,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// fetch current subscription object and ensure the object was not deleted in the meantime
	currentSubscription := &eventingv1alpha2.Subscription{}
	if err := r.Client.Get(ctx, req.NamespacedName, currentSubscription); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// copy the subscription object, so we don't modify the source object
	sub := currentSubscription.DeepCopy()

	// bind fields to logger
	log := backendutils.LoggerWithSubscription(r.namedLogger(), sub)
	log.Debugw("Received new reconcile request")

	// instantiate a return object
	result := ctrl.Result{}

	// handle deletion of the subscription
	if isInDeletion(sub) {
		return r.handleDeleteSubscription(ctx, sub, log)
	}

	defer r.collector.RecordSubscriptionStatus(sub.Status.Ready,
		sub.Name,
		sub.Namespace,
		backendType,
		"",
		"",
	)

	// sync the initial Subscription status
	r.syncInitialStatus(sub)

	// sync Finalizers, ensure the finalizer is set
	if err := r.syncFinalizer(sub, log); err != nil {
		if updateErr := r.updateSubscription(ctx, sub, log); updateErr != nil {
			return ctrl.Result{}, xerrors.Errorf(updateErr.Error()+": %v", err)
		}
		return ctrl.Result{}, xerrors.Errorf("failed to sync finalizer: %v", err)
	}

	// validate sink.
	if err := r.sinkValidator.Validate(sub); err != nil {
		return ctrl.Result{}, err
	}

	// sync APIRule for the desired subscription
	webhookURL, err := r.apiGateway.ExposeSink(ctx, *sub, r.Domain, *r.oauth2credentials, log) // TODO: fix the args.
	// sync the condition for webhook status.
	sub.Status.SetConditionAPIRuleStatus(err)
	if !recerrors.IsSkippable(err) {
		if updateErr := r.updateSubscription(ctx, sub, log); updateErr != nil {
			return ctrl.Result{}, xerrors.Errorf(updateErr.Error()+": %v", err)
		}
		return ctrl.Result{}, err
	}

	// sync the EventMesh Subscription with the Subscription CR
	ready, err := r.syncEventMeshSubscription(sub, webhookURL, log)
	if err != nil {
		if updateErr := r.updateSubscription(ctx, sub, log); updateErr != nil {
			return ctrl.Result{}, xerrors.Errorf(updateErr.Error()+": %v", err)
		}
		return ctrl.Result{}, err
	}
	// if eventMesh subscription is not ready, then requeue
	if !ready {
		log.Debugw("Requeuing reconciliation because EventMesh subscription is not ready")
		result.RequeueAfter = requeueAfterDuration
	}

	// update the subscription if modified
	if err := r.updateSubscription(ctx, sub, log); err != nil {
		return ctrl.Result{}, err
	}

	return result, nil
}

// updateSubscription updates the subscription changes to k8s.
func (r *Reconciler) updateSubscription(ctx context.Context, sub *eventingv1alpha2.Subscription, logger *zap.SugaredLogger) error {
	namespacedName := &k8stypes.NamespacedName{
		Name:      sub.Name,
		Namespace: sub.Namespace,
	}

	// fetch the latest subscription object, to avoid k8s conflict errors
	latestSubscription := &eventingv1alpha2.Subscription{}
	if err := r.Client.Get(ctx, *namespacedName, latestSubscription); err != nil {
		return err
	}

	// copy new changes to the latest object
	newSubscription := latestSubscription.DeepCopy()
	newSubscription.Status = sub.Status
	newSubscription.ObjectMeta.Finalizers = sub.ObjectMeta.Finalizers

	// emit the condition events if needed
	r.emitConditionEvents(latestSubscription, newSubscription, logger)

	// sync sub status with k8s
	if err := r.updateStatus(ctx, latestSubscription, newSubscription, logger); err != nil {
		return err
	}

	// update the subscription object in k8s
	if !reflect.DeepEqual(latestSubscription.ObjectMeta.Finalizers, newSubscription.ObjectMeta.Finalizers) {
		if err := r.Update(ctx, newSubscription); err != nil {
			return xerrors.Errorf("failed to remove finalizer name '%s': %v", eventingv1alpha2.Finalizer, err)
		}
		logger.Debugw("Updated subscription meta for finalizers", "oldFinalizers", latestSubscription.ObjectMeta.Finalizers, "newFinalizers", newSubscription.ObjectMeta.Finalizers)
	}

	return nil
}

// emitConditionEvents check each condition, if the condition is modified then emit an event.
func (r *Reconciler) emitConditionEvents(oldSubscription, newSubscription *eventingv1alpha2.Subscription, logger *zap.SugaredLogger) {
	for _, condition := range newSubscription.Status.Conditions {
		oldCondition := oldSubscription.Status.FindCondition(condition.Type)
		if oldCondition != nil && eventingv1alpha2.ConditionEquals(*oldCondition, condition) {
			continue
		}
		// condition is modified, so emit an event
		r.emitConditionEvent(newSubscription, condition)
		logger.Debug("Emitted condition event", condition)
	}
}

// updateStatus updates the status to k8s if modified.
func (r *Reconciler) updateStatus(ctx context.Context, oldSubscription, newSubscription *eventingv1alpha2.Subscription, logger *zap.SugaredLogger) error {
	// compare the status taking into consideration lastTransitionTime in conditions
	if object.IsSubscriptionStatusEqual(oldSubscription.Status, newSubscription.Status) {
		return nil
	}

	// update the status for subscription in k8s
	if err := r.Status().Update(ctx, newSubscription); err != nil {
		return xerrors.Errorf("failed to update subscription status: %v", err)
	}
	logger.Debugw("Updated subscription status", "oldStatus", oldSubscription.Status, "newStatus", newSubscription.Status)

	return nil
}

// syncFinalizer sets the finalizer in the Subscription.
func (r *Reconciler) syncFinalizer(subscription *eventingv1alpha2.Subscription, logger *zap.SugaredLogger) error {
	// Check if finalizer is already set
	if isFinalizerSet(subscription) {
		return nil
	}

	return addFinalizer(subscription, logger)
}

func (r *Reconciler) handleDeleteSubscription(ctx context.Context, subscription *eventingv1alpha2.Subscription,
	logger *zap.SugaredLogger) (ctrl.Result, error) {
	// delete EventMesh subscriptions
	logger.Debug("Deleting subscription on EventMesh")
	if err := r.Backend.DeleteSubscription(subscription); err != nil {
		return ctrl.Result{}, err
	}

	// update condition in subscription status
	condition := eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscribed,
		eventingv1alpha2.ConditionReasonSubscriptionDeleted, corev1.ConditionFalse, "")
	replaceStatusCondition(subscription, condition)

	// remove finalizers from subscription
	removeFinalizer(subscription)

	// update subscription CR with changes
	if err := r.updateSubscription(ctx, subscription, logger); err != nil {
		return ctrl.Result{}, err
	}

	r.collector.RemoveSubscriptionStatus(subscription.Name, subscription.Namespace, backendType, "", "")
	return ctrl.Result{Requeue: false}, nil
}

// syncEventMeshSubscription delegates the subscription synchronization to the backend client. It returns true if the subscription is ready.
func (r *Reconciler) syncEventMeshSubscription(subscription *eventingv1alpha2.Subscription, webhookURL string, logger *zap.SugaredLogger) (bool, error) {
	logger.Debug("Syncing subscription with EventMesh")

	if _, err := r.Backend.SyncSubscription(subscription, r.cleaner, webhookURL); err != nil {
		r.syncConditionSubscribed(subscription, err)
		return false, err
	}

	// check if the eventMesh subscription is active
	isActive, err := r.checkStatusActive(subscription)
	if err != nil {
		return false, xerrors.Errorf("reached retry timeout: %v", err)
	}

	// sync the condition: ConditionSubscribed
	r.syncConditionSubscribed(subscription, nil)

	// sync the condition: ConditionSubscriptionActive
	r.syncConditionSubscriptionActive(subscription, isActive, logger)

	// sync the condition: WebhookCallStatus
	r.syncConditionWebhookCallStatus(subscription)

	return isActive, nil
}

// syncConditionSubscribed syncs the condition ConditionSubscribed.
func (r *Reconciler) syncConditionSubscribed(subscription *eventingv1alpha2.Subscription, err error) {
	// Include the EventMesh subscription ID in the Condition message
	message := eventingv1alpha2.CreateMessageForConditionReasonSubscriptionCreated(r.nameMapper.MapSubscriptionName(subscription.Name, subscription.Namespace))
	condition := eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscribed, eventingv1alpha2.ConditionReasonSubscriptionCreated, corev1.ConditionTrue, message)
	if err != nil {
		message = err.Error()
		condition = eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscribed, eventingv1alpha2.ConditionReasonSubscriptionCreationFailed, corev1.ConditionFalse, message)
	}

	replaceStatusCondition(subscription, condition)
}

// syncConditionSubscriptionActive syncs the condition ConditionSubscribed.
func (r *Reconciler) syncConditionSubscriptionActive(subscription *eventingv1alpha2.Subscription, isActive bool, logger *zap.SugaredLogger) {
	condition := eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscriptionActive,
		eventingv1alpha2.ConditionReasonSubscriptionActive,
		corev1.ConditionTrue,
		"")
	if !isActive {
		logger.Infow("Waiting for subscription to be active", "name", subscription.Name,
			"status", subscription.Status.Backend.EventMeshSubscriptionStatus)
		message := "Waiting for subscription to be active"
		condition = eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscriptionActive,
			eventingv1alpha2.ConditionReasonSubscriptionNotActive,
			corev1.ConditionFalse,
			message)
	}
	replaceStatusCondition(subscription, condition)
}

// syncConditionWebhookCallStatus syncs the condition WebhookCallStatus
// checks if the last webhook call returned an error.
func syncConditionWebhookCallStatus(subscription *eventingv1alpha2.Subscription) {
	condition := eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionWebhookCallStatus,
		eventingv1alpha2.ConditionReasonWebhookCallStatus, corev1.ConditionFalse, "")
	if isWebhookCallError, err := checkLastFailedDelivery(subscription); err != nil {
		condition.Message = err.Error()
	} else if isWebhookCallError {
		condition.Message = subscription.Status.Backend.EventMeshSubscriptionStatus.LastFailedDeliveryReason
	} else {
		condition.Status = corev1.ConditionTrue
	}
	replaceStatusCondition(subscription, condition)
}

// syncInitialStatus determines the desired initial status and updates it accordingly (if conditions changed).
func (r *Reconciler) syncInitialStatus(subscription *eventingv1alpha2.Subscription) {
	if subscription.Status.Types == nil {
		subscription.Status.InitializeEventTypes()
	}

	expectedStatus := eventingv1alpha2.SubscriptionStatus{}
	expectedStatus.InitializeConditions()

	// case: conditions are already initialized and there is no change in the Ready status
	if eventingv1alpha2.ContainSameConditionTypes(subscription.Status.Conditions, expectedStatus.Conditions) &&
		!subscription.Status.ShouldUpdateReadyStatus() {
		return
	}

	if len(subscription.Status.Conditions) == 0 {
		expectedStatus.Backend.CopyHashes(subscription.Status.Backend)
		subscription.Status = expectedStatus
	} else {
		requiredConditions := getRequiredConditions(subscription.Status.Conditions, expectedStatus.Conditions)
		subscription.Status.Conditions = requiredConditions
		subscription.Status.Ready = !subscription.Status.Ready
	}

	// reset the status for apiRule
	subscription.Status.Backend.APIRuleName = ""
	subscription.Status.Backend.ExternalSink = ""
}

// getRequiredConditions removes the non-required conditions from the subscription  and adds any missing required-conditions.
func getRequiredConditions(subscriptionConditions, expectedConditions []eventingv1alpha2.Condition) []eventingv1alpha2.Condition {
	var requiredConditions []eventingv1alpha2.Condition
	expectedConditionsMap := make(map[eventingv1alpha2.ConditionType]eventingv1alpha2.Condition)
	for _, condition := range expectedConditions {
		expectedConditionsMap[condition.Type] = condition
	}

	// add the current subscription's conditions if it exists in the expectedConditions
	for _, condition := range subscriptionConditions {
		if _, ok := expectedConditionsMap[condition.Type]; ok {
			requiredConditions = append(requiredConditions, condition)
			delete(expectedConditionsMap, condition.Type)
		}
	}
	// add the remaining conditions that weren't present in the subscription
	for _, condition := range expectedConditionsMap {
		requiredConditions = append(requiredConditions, condition)
	}

	return requiredConditions
}

// replaceStatusCondition replaces the given condition on the subscription. Also it sets the readiness in the status.
// So make sure you always use this method then changing a condition.
func replaceStatusCondition(subscription *eventingv1alpha2.Subscription,
	condition eventingv1alpha2.Condition) bool {
	// the subscription is ready if all conditions are fulfilled
	isReady := true

	// compile list of desired conditions
	desiredConditions := make([]eventingv1alpha2.Condition, 0)
	for _, c := range subscription.Status.Conditions {
		var chosenCondition eventingv1alpha2.Condition
		if c.Type == condition.Type {
			// take given condition
			chosenCondition = condition
		} else {
			// take already present condition
			chosenCondition = c
		}
		desiredConditions = append(desiredConditions, chosenCondition)
		if string(chosenCondition.Status) != string(v1.ConditionTrue) {
			isReady = false
		}
	}

	// prevent unnecessary updates
	if eventingv1alpha2.ConditionsEquals(subscription.Status.Conditions, desiredConditions) && subscription.Status.Ready == isReady {
		return false
	}

	// update the status
	subscription.Status.Conditions = desiredConditions
	subscription.Status.Ready = isReady
	return true
}

// emitConditionEvent emits a kubernetes event and sets the event type based on the Condition status.
func (r *Reconciler) emitConditionEvent(subscription *eventingv1alpha2.Subscription, condition eventingv1alpha2.Condition) {
	eventType := corev1.EventTypeNormal
	if condition.Status == corev1.ConditionFalse {
		eventType = corev1.EventTypeWarning
	}
	r.recorder.Event(subscription, eventType, string(condition.Reason), condition.Message)
}

// SetupUnmanaged creates a controller under the client control.
func (r *Reconciler) SetupUnmanaged(mgr ctrl.Manager) error {
	ctru, err := controller.NewUnmanaged(reconcilerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return fmt.Errorf("failed to create unmanaged controller: %w", err)
	}

	if err := ctru.Watch(source.Kind(mgr.GetCache(), &eventingv1alpha2.Subscription{}),
		&handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch subscriptions: %w", err)
	}

	apiRuleEventHandler := handler.EnqueueRequestForOwner(r.Scheme(), mgr.GetRESTMapper(),
		&eventingv1alpha2.Subscription{})
	if err := ctru.Watch(source.Kind(mgr.GetCache(), &apigatewayv1beta1.APIRule{}), apiRuleEventHandler); err != nil {
		return fmt.Errorf("failed to watch APIRule: %w", err)
	}

	go func(r *Reconciler, c controller.Controller) {
		if err := c.Start(r.ctx); err != nil {
			r.namedLogger().Fatalw("Failed to start controller",
				"name", reconcilerName, "error", err)
		}
	}(r, ctru)

	return nil
}

// checkStatusActive checks if the subscription is active and if not, sets a timer for retry.
func (r *Reconciler) checkStatusActive(subscription *eventingv1alpha2.Subscription) (active bool, err error) {
	if subscription.Status.Backend.EventMeshSubscriptionStatus == nil {
		return false, nil
	}

	// check if the EMS subscription status is active
	if subscription.Status.Backend.EventMeshSubscriptionStatus.Status == string(types.SubscriptionStatusActive) {
		if len(subscription.Status.Backend.FailedActivation) > 0 {
			subscription.Status.Backend.FailedActivation = ""
		}
		return true, nil
	}

	t1 := time.Now()
	if len(subscription.Status.Backend.FailedActivation) == 0 {
		// it's the first time
		subscription.Status.Backend.FailedActivation = t1.Format(time.RFC3339)
		return false, nil
	}

	// check the timeout
	if t0, er := time.Parse(time.RFC3339, subscription.Status.Backend.FailedActivation); er != nil {
		err = er
	} else if t1.Sub(t0) > timeoutRetryActiveEmsStatus {
		err = xerrors.Errorf("timeout waiting for the subscription to be active: %s", subscription.Name)
	}

	return false, err
}

// checkLastFailedDelivery checks if LastFailedDelivery exists and if it happened after LastSuccessfulDelivery.
func checkLastFailedDelivery(subscription *eventingv1alpha2.Subscription) (bool, error) {
	// Check if LastFailedDelivery exists.
	lastFailed := subscription.Status.Backend.EventMeshSubscriptionStatus.LastFailedDelivery
	if len(lastFailed) == 0 {
		return false, nil
	}

	// Try to parse LastFailedDelivery.
	var err error
	var lastFailedDeliveryTime time.Time
	if lastFailedDeliveryTime, err = time.Parse(time.RFC3339, lastFailed); err != nil {
		return true, xerrors.Errorf("failed to parse LastFailedDelivery: %v", err)
	}

	// Check if LastSuccessfulDelivery exists. If not, LastFailedDelivery happened last.
	lastSuccessful := subscription.Status.Backend.EventMeshSubscriptionStatus.LastSuccessfulDelivery
	if len(lastSuccessful) == 0 {
		return true, nil
	}

	// Try to parse LastSuccessfulDelivery.
	var lastSuccessfulDeliveryTime time.Time
	if lastSuccessfulDeliveryTime, err = time.Parse(time.RFC3339, lastSuccessful); err != nil {
		return true, xerrors.Errorf("failed to parse LastSuccessfulDelivery: %v", err)
	}

	return lastFailedDeliveryTime.After(lastSuccessfulDeliveryTime), nil
}

func (r *Reconciler) namedLogger() *zap.SugaredLogger {
	return r.logger.WithContext().Named(reconcilerName)
}

// SetCredentials sets the WebhookAuth credentials.
// WARNING: This functions should be used for testing purposes only.
func (r *Reconciler) SetCredentials(credentials *eventmesh.OAuth2ClientCredentials) {
	r.namedLogger().Warn("This logic should be used for testing purposes only")
	r.oauth2credentials = credentials
}
