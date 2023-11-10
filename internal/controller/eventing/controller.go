/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package eventing

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/options"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	"github.com/kyma-project/eventing-manager/pkg/object"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager"
	"github.com/kyma-project/eventing-manager/pkg/watcher"
)

const (
	FinalizerName             = "eventing.operator.kyma-project.io/finalizer"
	ControllerName            = "eventing-manager-controller"
	NatsServerNotAvailableMsg = "NATS server is not available"
	natsClientPort            = 4222

	AppLabelValue             = eventing.PublisherName
	PublisherSecretEMSHostKey = "ems-publish-host"

	TokenEndpointFormat                   = "%s?grant_type=%s&response_type=token"
	NamespacePrefix                       = "/"
	EventMeshPublishEndpointForSubscriber = "/sap/ems/v1"
	EventMeshPublishEndpointForPublisher  = "/sap/ems/v1/events"

	SubscriptionExistsErrMessage = "cannot delete the eventing module as subscription exists"
)

// Reconciler reconciles an Eventing object
//
//go:generate go run github.com/vektra/mockery/v2 --name=Controller --dir=../../../vendor/sigs.k8s.io/controller-runtime/pkg/controller --outpkg=mocks --case=underscore
//go:generate go run github.com/vektra/mockery/v2 --name=Manager --dir=../../../vendor/sigs.k8s.io/controller-runtime/pkg/manager --outpkg=mocks --case=underscore
type Reconciler struct {
	client.Client
	logger                        *logger.Logger
	ctrlManager                   ctrl.Manager
	controller                    controller.Controller
	eventingManager               eventing.Manager
	kubeClient                    k8s.Client
	dynamicClient                 dynamic.Interface
	scheme                        *runtime.Scheme
	recorder                      record.EventRecorder
	subManagerFactory             subscriptionmanager.ManagerFactory
	natsSubManager                manager.Manager
	eventMeshSubManager           manager.Manager
	isNATSSubManagerStarted       bool
	isEventMeshSubManagerStarted  bool
	natsConfigHandler             NatsConfigHandler
	oauth2credentials             oauth2Credentials
	backendConfig                 env.BackendConfig
	allowedEventingCR             *eventingv1alpha1.Eventing
	clusterScopedResourcesWatched bool
	natsCRWatchStarted            bool
	natsWatchers                  map[string]watcher.Watcher
}

func NewReconciler(
	client client.Client,
	kubeClient k8s.Client,
	dynamicClient dynamic.Interface,
	scheme *runtime.Scheme,
	logger *logger.Logger,
	recorder record.EventRecorder,
	manager eventing.Manager,
	backendConfig env.BackendConfig,
	subManagerFactory subscriptionmanager.ManagerFactory,
	opts *options.Options,
	allowedEventingCR *eventingv1alpha1.Eventing,
) *Reconciler {
	return &Reconciler{
		Client:                  client,
		logger:                  logger,
		ctrlManager:             nil, // ctrlManager will be initialized in `SetupWithManager`.
		eventingManager:         manager,
		kubeClient:              kubeClient,
		dynamicClient:           dynamicClient,
		scheme:                  scheme,
		recorder:                recorder,
		backendConfig:           backendConfig,
		subManagerFactory:       subManagerFactory,
		natsSubManager:          nil,
		eventMeshSubManager:     nil,
		isNATSSubManagerStarted: false,
		natsConfigHandler:       NewNatsConfigHandler(kubeClient, opts),
		allowedEventingCR:       allowedEventingCR,
		natsWatchers:            make(map[string]watcher.Watcher),
	}
}

// RBAC permissions.
//nolint:lll
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete;patch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="autoscaling",resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=security.istio.io,resources=customresourcedefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=validatingwebhookconfigurations,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="applicationconnector.kyma-project.io",resources=applications,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups=gateway.kyma-project.io,resources=apirules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="eventing.kyma-project.io",resources=subscriptions,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups=eventing.kyma-project.io,resources=subscriptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources=subscriptions,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=subscriptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=security.istio.io,resources=peerauthentications,verbs=get;list;watch;create;update;patch;delete
// Generate required RBAC to emit kubernetes events in the controller.
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.namedLogger().Info("Reconciliation triggered")
	// fetch latest subscription object
	currentEventing := &eventingv1alpha1.Eventing{}
	if err := r.Get(ctx, req.NamespacedName, currentEventing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// copy the object, so we don't modify the source object
	eventing := currentEventing.DeepCopy()

	// watch cluster-scoped resources, such as clusterrole and clusterrolebinding.
	if !r.clusterScopedResourcesWatched {
		if err := r.watchResource(&rbacv1.ClusterRole{}, eventing); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.watchResource(&rbacv1.ClusterRoleBinding{}, eventing); err != nil {
			return ctrl.Result{}, err
		}
		r.clusterScopedResourcesWatched = true
	}

	// logger with eventing details
	log := r.loggerWithEventing(eventing)

	// check if eventing is in deletion state
	if !eventing.DeletionTimestamp.IsZero() {
		return r.handleEventingDeletion(ctx, eventing, log)
	}

	// check if the Eventing CR is allowed to be created.
	if r.allowedEventingCR != nil {
		if result, err := r.handleEventingCRAllowedCheck(ctx, eventing, log); !result || err != nil {
			return ctrl.Result{}, err
		}
	}

	// handle reconciliation
	return r.handleEventingReconcile(ctx, eventing, log)
}

// handleEventingCRAllowedCheck checks if Eventing CR is allowed to be created or not.
// returns true if the Eventing CR is allowed.
func (r *Reconciler) handleEventingCRAllowedCheck(ctx context.Context, eventing *eventingv1alpha1.Eventing,
	log *zap.SugaredLogger) (bool, error) {
	// if the name and namespace matches with allowed NATS CR then allow the CR to be reconciled.
	if eventing.Name == r.allowedEventingCR.Name && eventing.Namespace == r.allowedEventingCR.Namespace {
		return true, nil
	}

	// set error state in status.
	eventing.Status.SetStateError()
	// update conditions in status.
	errorMessage := fmt.Sprintf("Only a single Eventing CR with name: %s and namespace: %s "+
		"is allowed to be created in a Kyma cluster.", r.allowedEventingCR.Name, r.allowedEventingCR.Namespace)
	eventing.Status.UpdateConditionPublisherProxyReady(metav1.ConditionFalse,
		eventingv1alpha1.ConditionReasonForbidden, errorMessage)

	return false, r.syncEventingStatus(ctx, eventing, log)
}

func (r *Reconciler) SkipEnqueueOnUpdateAfterSemanticCompare(e event.UpdateEvent) bool {
	res := !object.Semantic.DeepEqual(e.ObjectOld, e.ObjectNew)
	r.namedLogger().With("result", res).
		With("GVK", e.ObjectNew.GetObjectKind().GroupVersionKind()).
		With("Name", e.ObjectNew.GetName()).
		With("Namespace", e.ObjectNew.GetNamespace()).
		Debug("UpdateEvent received")
	return res
}
func (r *Reconciler) SkipEnqueueOnCreate(e event.CreateEvent) bool {
	r.namedLogger().
		With("GVK", e.Object.GetObjectKind().GroupVersionKind()).
		With("Name", e.Object.GetName()).
		With("Namespace", e.Object.GetNamespace()).
		Debug("CreateEvent received. Skipping")
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ctrlManager = mgr

	var err error
	r.controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&eventingv1alpha1.Eventing{}).
		Owns(&v1.Deployment{}, builder.WithPredicates(
			predicate.Funcs{
				CreateFunc: r.SkipEnqueueOnCreate,
				UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare,
			},
		)).
		Owns(&corev1.Service{}, builder.WithPredicates(
			predicate.Funcs{
				CreateFunc: r.SkipEnqueueOnCreate,
				UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare,
			})).
		Owns(&corev1.ServiceAccount{}, builder.WithPredicates(
			predicate.Funcs{
				CreateFunc: r.SkipEnqueueOnCreate,
				UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare,
			})).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}, builder.WithPredicates(
			predicate.Funcs{
				CreateFunc: r.SkipEnqueueOnCreate,
				UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare,
			})).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 0,
		}).
		Build(r)

	return err
}

func (r *Reconciler) watchResource(kind client.Object, eventing *eventingv1alpha1.Eventing) error {
	err := r.controller.Watch(
		source.Kind(r.ctrlManager.GetCache(), kind),
		handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
			// Enqueue a reconcile request for the eventing resource
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Namespace: eventing.Namespace,
					Name:      eventing.Name,
				}},
			}
		}),
		predicate.ResourceVersionChangedPredicate{},
		predicate.Funcs{
			// don't reconcile for create events
			CreateFunc: r.SkipEnqueueOnCreate,
			UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare,
		},
	)
	return err
}

func (r *Reconciler) startNATSCRWatch(eventing *eventingv1alpha1.Eventing) error {
	natsWatcher, found := r.natsWatchers[eventing.Namespace]
	if found && natsWatcher.IsStarted() {
		return nil
	}

	if !found {
		natsWatcher = watcher.NewResourceWatcher(r.dynamicClient, k8s.NatsGVK, eventing.Namespace)
		r.natsWatchers[eventing.Namespace] = natsWatcher
	}

	if err := r.controller.Watch(&source.Channel{Source: natsWatcher.GetEventsChannel()},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Namespace: eventing.Namespace,
					Name:      eventing.Name,
				}},
			}
		}),
		predicate.ResourceVersionChangedPredicate{},
	); err != nil {
		return err
	}
	natsWatcher.Start()

	return nil
}

func (r *Reconciler) stopNATSCRWatch(eventing *eventingv1alpha1.Eventing) {
	natsWatcher, found := r.natsWatchers[eventing.Namespace]
	if found {
		natsWatcher.Stop()
		delete(r.natsWatchers, eventing.Namespace)
	}
}

// loggerWithEventing returns a logger with the given Eventing CR details.
func (r *Reconciler) loggerWithEventing(eventing *eventingv1alpha1.Eventing) *zap.SugaredLogger {
	return r.namedLogger().With(
		"kind", eventing.GetObjectKind().GroupVersionKind().Kind,
		"resourceVersion", eventing.GetResourceVersion(),
		"generation", eventing.GetGeneration(),
		"namespace", eventing.GetNamespace(),
		"name", eventing.GetName(),
	)
}

func (r *Reconciler) handleEventingDeletion(ctx context.Context, eventing *eventingv1alpha1.Eventing,
	log *zap.SugaredLogger) (ctrl.Result, error) {
	// skip reconciliation for deletion if the finalizer is not set.
	if !r.containsFinalizer(eventing) {
		log.Debug("skipped reconciliation for deletion as finalizer is not set.")
		return ctrl.Result{}, nil
	}

	// check if subscription resources exist
	exists, err := r.eventingManager.SubscriptionExists(ctx)
	if err != nil {
		eventing.Status.SetStateError()
		return ctrl.Result{}, r.syncStatusWithDeletionErr(ctx, eventing, err, log)
	}
	if exists {
		eventing.Status.SetStateWarning()
		return ctrl.Result{Requeue: true}, r.syncStatusWithDeletionErr(ctx, eventing,
			errors.New(SubscriptionExistsErrMessage), log)
	}

	log.Info("handling Eventing deletion...")
	if eventing.Spec.Backend.Type == eventingv1alpha1.NatsBackendType {
		if err := r.stopNATSSubManager(true, log); err != nil {
			return ctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
		}
	} else {
		if err := r.stopEventMeshSubManager(true, log); err != nil {
			return ctrl.Result{}, r.syncStatusWithSubscriptionManagerErrWithReason(ctx,
				eventingv1alpha1.ConditionReasonEventMeshSubManagerStopFailed,
				eventing, err, log)
		}
	}
	eventing.Status.SetSubscriptionManagerReadyConditionToFalse(
		eventingv1alpha1.ConditionReasonStopped,
		eventingv1alpha1.ConditionSubscriptionManagerStoppedMessage)

	// delete cluster-scoped resources, such as clusterrole and clusterrolebinding.
	if err := r.deleteClusterScopedResources(ctx, eventing); err != nil {
		return ctrl.Result{}, r.syncStatusWithPublisherProxyErrWithReason(ctx,
			eventingv1alpha1.ConditionReasonDeletionError, eventing, err, log)
	}
	eventing.Status.SetPublisherProxyConditionToFalse(
		eventingv1alpha1.ConditionReasonDeleted,
		eventingv1alpha1.ConditionPublisherProxyDeletedMessage)

	return r.removeFinalizer(ctx, eventing)
}

// deleteClusterScopedResources deletes cluster-scoped resources, such as clusterrole and clusterrolebinding.
// K8s doesn't support cleaning cluster-scoped resources owned by namespace-scoped resources:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/#owner-references-in-object-specifications
func (r *Reconciler) deleteClusterScopedResources(ctx context.Context, eventingCR *eventingv1alpha1.Eventing) error {
	if err := r.kubeClient.DeleteClusterRole(ctx, eventing.GetPublisherClusterRoleName(*eventingCR), eventingCR.Namespace); err != nil {
		return err
	}
	return r.kubeClient.DeleteClusterRoleBinding(ctx, eventing.GetPublisherClusterRoleBindingName(*eventingCR), eventingCR.Namespace)
}

func (r *Reconciler) handleEventingReconcile(ctx context.Context,
	eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) (ctrl.Result, error) {
	log.Info("handling Eventing reconciliation...")

	// make sure the finalizer exists.
	if !r.containsFinalizer(eventing) {
		return r.addFinalizer(ctx, eventing)
	}

	// set state processing if not set yet
	r.InitStateProcessing(eventing)

	// sync webhooks CABundle.
	if err := r.reconcileWebhooksWithCABundle(ctx); err != nil {
		return ctrl.Result{}, r.syncStatusWithWebhookErr(ctx, eventing, err, log)
	}
	// set webhook condition to true.
	eventing.Status.SetWebhookReadyConditionToTrue()

	// handle switching of backend.
	if eventing.Status.ActiveBackend != "" {
		if err := r.handleBackendSwitching(eventing, log); err != nil {
			return ctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
		}
	}

	// update ActiveBackend in status.
	eventing.SyncStatusActiveBackend()

	// check if Application CRD is installed.
	isApplicationCRDEnabled, err := r.kubeClient.ApplicationCRDExists(ctx)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
	}
	r.backendConfig.PublisherConfig.ApplicationCRDEnabled = isApplicationCRDEnabled
	r.eventingManager.SetBackendConfig(r.backendConfig)

	// reconcile for specified backend.
	switch eventing.Spec.Backend.Type {
	case eventingv1alpha1.NatsBackendType:
		return r.reconcileNATSBackend(ctx, eventing, log)
	case eventingv1alpha1.EventMeshBackendType:
		return r.reconcileEventMeshBackend(ctx, eventing, log)
	default:
		return ctrl.Result{Requeue: false}, fmt.Errorf("not supported backend type %s", eventing.Spec.Backend.Type)
	}
}

func (r *Reconciler) handleBackendSwitching(
	eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) error {
	// check if the backend was changed.
	if !eventing.IsSpecBackendTypeChanged() {
		return nil
	}

	// stop the previously active backend.
	if eventing.Status.ActiveBackend == eventingv1alpha1.NatsBackendType {
		log.Info("Stopping the NATS subscription manager because backend is switched")
		if err := r.stopNATSSubManager(true, log); err != nil {
			return err
		}
		r.stopNATSCRWatch(eventing)
	} else if eventing.Status.ActiveBackend == eventingv1alpha1.EventMeshBackendType {
		if err := r.stopEventMeshSubManager(true, log); err != nil {
			return err
		}
	}

	// update the Eventing CR status.
	eventing.Status.SetStateProcessing()
	eventing.Status.ClearConditions()
	return nil
}

func (r *Reconciler) reconcileNATSBackend(ctx context.Context, eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) (ctrl.Result, error) {
	// retrieves the NATS CRD
	_, err := r.kubeClient.GetCRD(ctx, k8s.NatsGVK.GroupResource().String())
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf("NATS module has to be installed: %v", err)
			return ctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
		}
		return ctrl.Result{}, err
	}

	if err = r.startNATSCRWatch(eventing); err != nil {
		return ctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	// check nats CR if it exists and is in natsAvailable state
	err = r.checkNATSAvailability(ctx, eventing)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	// set NATSAvailable condition to true and update status
	eventing.Status.SetNATSAvailableConditionToTrue()

	deployment, err := r.handlePublisherProxy(ctx, eventing, eventing.Spec.Backend.Type)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithPublisherProxyErr(ctx, eventing, err, log)
	}

	// start NATS subscription manager
	if err := r.reconcileNATSSubManager(ctx, eventing, log); err != nil {
		return ctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	return r.handleEventingState(ctx, deployment, eventing, log)
}

func (r *Reconciler) checkNATSAvailability(ctx context.Context, eventing *eventingv1alpha1.Eventing) error {
	natsAvailable, err := r.eventingManager.IsNATSAvailable(ctx, eventing.Namespace)
	if err != nil {
		return err
	}
	if !natsAvailable {
		return fmt.Errorf(NatsServerNotAvailableMsg)
	}
	return nil
}

func (r *Reconciler) handlePublisherProxy(
	ctx context.Context,
	eventing *eventingv1alpha1.Eventing,
	backendType eventingv1alpha1.BackendType) (*v1.Deployment, error) {
	// get nats config with NATS server url
	var natsConfig *env.NATSConfig
	if backendType == eventingv1alpha1.NatsBackendType {
		var err error
		natsConfig, err = r.natsConfigHandler.GetNatsConfig(ctx, *eventing)
		if err != nil {
			return nil, err
		}
	}
	// CreateOrUpdate deployment for eventing publisher proxy deployment
	deployment, err := r.eventingManager.DeployPublisherProxy(ctx, eventing, natsConfig, backendType)
	if err != nil {
		return nil, err
	}

	// deploy publisher proxy resources.
	if err = r.eventingManager.DeployPublisherProxyResources(ctx, eventing, deployment); err != nil {
		return deployment, err
	}

	return deployment, nil
}

func (r *Reconciler) reconcileEventMeshBackend(ctx context.Context, eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) (ctrl.Result, error) {
	// check if APIRule CRD is installed.
	isAPIRuleCRDEnabled, err := r.kubeClient.APIRuleCRDExists(ctx)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
	} else if !isAPIRuleCRDEnabled {
		apiRuleMissingErr := errors.New("API-Gateway module is needed for EventMesh backend. APIRules CRD is not installed")
		return ctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, apiRuleMissingErr, log)
	}

	// Start the EventMesh subscription controller
	err = r.reconcileEventMeshSubManager(ctx, eventing)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
	}
	eventing.Status.SetSubscriptionManagerReadyConditionToTrue()

	deployment, err := r.handlePublisherProxy(ctx, eventing, eventing.Spec.Backend.Type)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithPublisherProxyErr(ctx, eventing, err, log)
	}
	return r.handleEventingState(ctx, deployment, eventing, log)
}

func (r *Reconciler) GetEventMeshSubManager() manager.Manager {
	return r.eventMeshSubManager
}

func (r *Reconciler) SetEventMeshSubManager(eventMeshSubManager manager.Manager) {
	r.eventMeshSubManager = eventMeshSubManager
}

func (r *Reconciler) SetKubeClient(kubeClient k8s.Client) {
	r.kubeClient = kubeClient
}

func (r *Reconciler) namedLogger() *zap.SugaredLogger {
	return r.logger.WithContext().Named(ControllerName)
}
