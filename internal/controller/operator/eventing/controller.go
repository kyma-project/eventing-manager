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
	"strings"

	"go.uber.org/zap"
	kappsv1 "k8s.io/api/apps/v1"
	kautoscalingv2 "k8s.io/api/autoscaling/v2"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	kctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
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

	PublisherSecretEMSHostKey = "ems-publish-host"

	TokenEndpointFormat                   = "%s?grant_type=%s&response_type=token"
	NamespacePrefix                       = "/"
	EventMeshPublishEndpointForSubscriber = "/sap/ems/v1"
	EventMeshPublishEndpointForPublisher  = "/sap/ems/v1/events"

	SubscriptionExistsErrMessage = "cannot delete the eventing module as subscription exists"
)

var (
	ErrSubscriptionExists      = errors.New(SubscriptionExistsErrMessage)
	ErrUnsupportedBackedType   = errors.New("backend type not supported")
	ErrNatsModuleMissing       = errors.New("NATS module has to be installed")
	ErrNATSServerUnavailable   = errors.New(NatsServerNotAvailableMsg)
	ErrAPIGatewayModuleMissing = errors.New("API-Gateway module is needed for EventMesh backend. APIRules CRD is not installed")
)

// Reconciler reconciles an Eventing object
//
//go:generate go run github.com/vektra/mockery/v2 --name=Controller --dir=../../../../vendor/sigs.k8s.io/controller-runtime/pkg/controller --outpkg=mocks --case=underscore
//go:generate go run github.com/vektra/mockery/v2 --name=Manager --dir=../../../../vendor/sigs.k8s.io/controller-runtime/pkg/manager --outpkg=mocks --case=underscore
type Reconciler struct {
	client.Client
	logger                        *logger.Logger
	ctrlManager                   kctrl.Manager
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
	allowedEventingCR             *operatorv1alpha1.Eventing
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
	allowedEventingCR *operatorv1alpha1.Eventing,
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
//+kubebuilder:rbac:groups=security.istio.io,resources=peerauthentications,verbs=get;list;watch;create;update;patch;delete
// Generate required RBAC to emit kubernetes events in the controller.
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req kctrl.Request) (kctrl.Result, error) {
	r.namedLogger().Info("Reconciliation triggered")
	// fetch latest subscription object
	currentEventing := &operatorv1alpha1.Eventing{}
	if err := r.Get(ctx, req.NamespacedName, currentEventing); err != nil {
		return kctrl.Result{}, client.IgnoreNotFound(err)
	}

	// copy the object, so we don't modify the source object
	eventingCR := currentEventing.DeepCopy()

	// watch cluster-scoped resources, such as ClusterRole and ClusterRoleBinding.
	if !r.clusterScopedResourcesWatched {
		if err := r.watchResource(&krbacv1.ClusterRole{}, eventingCR); err != nil {
			return kctrl.Result{}, err
		}
		if err := r.watchResource(&krbacv1.ClusterRoleBinding{}, eventingCR); err != nil {
			return kctrl.Result{}, err
		}
		r.clusterScopedResourcesWatched = true
	}

	// logger with eventing details
	log := r.loggerWithEventing(eventingCR)

	// check if eventing is in deletion state
	if !eventingCR.DeletionTimestamp.IsZero() {
		return r.handleEventingDeletion(ctx, eventingCR, log)
	}

	// check if the Eventing CR is allowed to be created.
	if r.allowedEventingCR != nil {
		if result, err := r.handleEventingCRAllowedCheck(ctx, eventingCR, log); !result || err != nil {
			return kctrl.Result{}, err
		}
	}

	// handle reconciliation
	return r.handleEventingReconcile(ctx, eventingCR, log)
}

// handleEventingCRAllowedCheck checks if Eventing CR is allowed to be created or not.
// returns true if the Eventing CR is allowed.
func (r *Reconciler) handleEventingCRAllowedCheck(ctx context.Context, eventing *operatorv1alpha1.Eventing,
	log *zap.SugaredLogger,
) (bool, error) {
	// if the name and namespace matches with allowed NATS CR then allow the CR to be reconciled.
	if eventing.Name == r.allowedEventingCR.Name && eventing.Namespace == r.allowedEventingCR.Namespace {
		return true, nil
	}

	// set error state in status.
	eventing.Status.SetStateError()
	// update conditions in status.
	errorMessage := fmt.Sprintf("Only a single Eventing CR with name: %s and namespace: %s "+
		"is allowed to be created in a Kyma cluster.", r.allowedEventingCR.Name, r.allowedEventingCR.Namespace)
	eventing.Status.UpdateConditionPublisherProxyReady(kmetav1.ConditionFalse,
		operatorv1alpha1.ConditionReasonForbidden, errorMessage)

	return false, r.syncEventingStatus(ctx, eventing, log)
}

func (r *Reconciler) logEventForResource(name, namespace, resourceType, eventType string, enqueue bool) {
	r.namedLogger().
		With("Name", name).
		With("Namespace", namespace).
		With("ResourceType", resourceType).
		With("EventType", eventType).
		With("Enqueue", enqueue).
		Debug("Event received")
}

func (r *Reconciler) SkipEnqueueOnCreate() func(event.CreateEvent) bool {
	return func(e event.CreateEvent) bool {
		// always skip enqueue
		return false
	}
}

func (r *Reconciler) SkipEnqueueOnUpdateAfterSemanticCompare(resourceType, nameFilter string) func(event.UpdateEvent) bool {
	return func(e event.UpdateEvent) bool {
		// skip enqueue if the resource is not related to Eventing
		if !strings.Contains(e.ObjectOld.GetName(), nameFilter) {
			return false
		}

		// enqueue only if the resource changed
		enqueue := !object.Semantic.DeepEqual(e.ObjectOld, e.ObjectNew)
		r.logEventForResource(e.ObjectOld.GetName(), e.ObjectOld.GetNamespace(), resourceType, "Update", enqueue)
		return enqueue
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr kctrl.Manager) error {
	r.ctrlManager = mgr

	var err error
	r.controller, err = kctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Eventing{}).
		Owns(&kappsv1.Deployment{},
			builder.WithPredicates(
				predicate.Funcs{
					CreateFunc: r.SkipEnqueueOnCreate(),
					UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare("Deployment", "eventing"),
				},
			),
		).
		Owns(&kcorev1.Service{},
			builder.WithPredicates(
				predicate.Funcs{
					CreateFunc: r.SkipEnqueueOnCreate(),
					UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare("Service", "eventing"),
				},
			),
		).
		Owns(&kcorev1.ServiceAccount{},
			builder.WithPredicates(
				predicate.Funcs{
					CreateFunc: r.SkipEnqueueOnCreate(),
					UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare("SA", "eventing"),
				},
			),
		).
		Owns(&kautoscalingv2.HorizontalPodAutoscaler{},
			builder.WithPredicates(
				predicate.Funcs{
					CreateFunc: r.SkipEnqueueOnCreate(),
					UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare("HPA", "eventing"),
				},
			),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 0,
		}).
		Build(r)

	return err
}

func (r *Reconciler) watchResource(kind client.Object, eventing *operatorv1alpha1.Eventing) error {
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
			CreateFunc: r.SkipEnqueueOnCreate(),
			UpdateFunc: r.SkipEnqueueOnUpdateAfterSemanticCompare("CR/CRB", "eventing"),
		},
	)
	return err
}

func (r *Reconciler) startNATSCRWatch(eventing *operatorv1alpha1.Eventing) error {
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

func (r *Reconciler) stopNATSCRWatch(eventing *operatorv1alpha1.Eventing) {
	natsWatcher, found := r.natsWatchers[eventing.Namespace]
	if found {
		natsWatcher.Stop()
		delete(r.natsWatchers, eventing.Namespace)
	}
}

// loggerWithEventing returns a logger with the given Eventing CR details.
func (r *Reconciler) loggerWithEventing(eventing *operatorv1alpha1.Eventing) *zap.SugaredLogger {
	return r.namedLogger().With(
		"kind", eventing.GetObjectKind().GroupVersionKind().Kind,
		"resourceVersion", eventing.GetResourceVersion(),
		"generation", eventing.GetGeneration(),
		"namespace", eventing.GetNamespace(),
		"name", eventing.GetName(),
	)
}

func (r *Reconciler) handleEventingDeletion(ctx context.Context, eventing *operatorv1alpha1.Eventing,
	log *zap.SugaredLogger,
) (kctrl.Result, error) {
	// skip reconciliation for deletion if the finalizer is not set.
	if !r.containsFinalizer(eventing) {
		log.Debug("skipped reconciliation for deletion as finalizer is not set.")
		return kctrl.Result{}, nil
	}

	if eventing.Spec.Backend == nil {
		// backend config can only be empty during creation and nothing is created if it is missing.
		// Hence, eventing can be safely removed.
		return r.removeFinalizer(ctx, eventing)
	}

	// check if subscription resources exist
	exists, err := r.eventingManager.SubscriptionExists(ctx)
	if err != nil {
		eventing.Status.SetStateError()
		return kctrl.Result{}, r.syncStatusWithDeletionErr(ctx, eventing, err, log)
	}
	if exists {
		eventing.Status.SetStateWarning()
		return kctrl.Result{Requeue: true}, r.syncStatusWithDeletionErr(ctx, eventing, ErrSubscriptionExists, log)
	}

	log.Info("handling Eventing deletion...")
	if eventing.Spec.Backend.Type == operatorv1alpha1.NatsBackendType {
		if err := r.stopNATSSubManager(true, log); err != nil {
			return kctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
		}
	} else {
		if err := r.stopEventMeshSubManager(true, log); err != nil {
			return kctrl.Result{}, r.syncStatusWithSubscriptionManagerErrWithReason(ctx,
				operatorv1alpha1.ConditionReasonEventMeshSubManagerStopFailed,
				eventing, err, log)
		}
	}
	eventing.Status.SetSubscriptionManagerReadyConditionToFalse(
		operatorv1alpha1.ConditionReasonStopped,
		operatorv1alpha1.ConditionSubscriptionManagerStoppedMessage)

	// delete cluster-scoped resources, such as clusterrole and clusterrolebinding.
	if err := r.deleteClusterScopedResources(ctx, eventing); err != nil {
		return kctrl.Result{}, r.syncStatusWithPublisherProxyErrWithReason(ctx,
			operatorv1alpha1.ConditionReasonDeletionError, eventing, err, log)
	}
	eventing.Status.SetPublisherProxyConditionToFalse(
		operatorv1alpha1.ConditionReasonDeleted,
		operatorv1alpha1.ConditionPublisherProxyDeletedMessage)

	return r.removeFinalizer(ctx, eventing)
}

// deleteClusterScopedResources deletes cluster-scoped resources, such as clusterrole and clusterrolebinding.
// K8s doesn't support cleaning cluster-scoped resources owned by namespace-scoped resources:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/#owner-references-in-object-specifications
func (r *Reconciler) deleteClusterScopedResources(ctx context.Context, eventingCR *operatorv1alpha1.Eventing) error {
	if err := r.kubeClient.DeleteClusterRole(ctx, eventing.GetPublisherClusterRoleName(*eventingCR), eventingCR.Namespace); err != nil {
		return err
	}
	return r.kubeClient.DeleteClusterRoleBinding(ctx, eventing.GetPublisherClusterRoleBindingName(*eventingCR), eventingCR.Namespace)
}

func (r *Reconciler) handleEventingReconcile(ctx context.Context,
	eventing *operatorv1alpha1.Eventing, log *zap.SugaredLogger,
) (kctrl.Result, error) {
	log.Info("handling Eventing reconciliation...")

	// make sure the finalizer exists.
	if !r.containsFinalizer(eventing) {
		return r.addFinalizer(ctx, eventing)
	}

	// set state processing if not set yet
	r.InitStateProcessing(eventing)
	if eventing.Spec.Backend == nil {
		return kctrl.Result{Requeue: true}, r.syncStatusForEmptyBackend(ctx,
			operatorv1alpha1.ConditionReasonBackendNotSpecified,
			operatorv1alpha1.ConditionBackendNotSpecifiedMessage,
			eventing, log)
	}

	// sync webhooks CABundle.
	if err := r.reconcileWebhooksWithCABundle(ctx); err != nil {
		return kctrl.Result{}, r.syncStatusWithWebhookErr(ctx, eventing, err, log)
	}
	// set webhook condition to true.
	eventing.Status.SetWebhookReadyConditionToTrue()

	// handle switching of backend.
	if eventing.Status.ActiveBackend != "" {
		if err := r.handleBackendSwitching(eventing, log); err != nil {
			return kctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
		}
	}

	// update ActiveBackend in status.
	eventing.SyncStatusActiveBackend()

	// check if Application CRD is installed.
	isApplicationCRDEnabled, err := r.kubeClient.ApplicationCRDExists(ctx)
	if err != nil {
		return kctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
	}
	r.backendConfig.PublisherConfig.ApplicationCRDEnabled = isApplicationCRDEnabled
	r.eventingManager.SetBackendConfig(r.backendConfig)

	// reconcile for specified backend.
	switch eventing.Spec.Backend.Type {
	case operatorv1alpha1.NatsBackendType:
		return r.reconcileNATSBackend(ctx, eventing, log)
	case operatorv1alpha1.EventMeshBackendType:
		return r.reconcileEventMeshBackend(ctx, eventing, log)
	default:
		return kctrl.Result{Requeue: false}, fmt.Errorf("%w: %s", ErrUnsupportedBackedType, eventing.Spec.Backend.Type)
	}
}

func (r *Reconciler) handleBackendSwitching(
	eventing *operatorv1alpha1.Eventing, log *zap.SugaredLogger,
) error {
	// check if the backend was changed.
	if !eventing.IsSpecBackendTypeChanged() {
		return nil
	}

	// stop the previously active backend.
	if eventing.Status.ActiveBackend == operatorv1alpha1.NatsBackendType {
		log.Info("Stopping the NATS subscription manager because backend is switched")
		if err := r.stopNATSSubManager(true, log); err != nil {
			return err
		}
		r.stopNATSCRWatch(eventing)
	} else if eventing.Status.ActiveBackend == operatorv1alpha1.EventMeshBackendType {
		if err := r.stopEventMeshSubManager(true, log); err != nil {
			return err
		}
	}

	// update the Eventing CR status.
	eventing.Status.SetStateProcessing()
	eventing.Status.ClearConditions()
	return nil
}

func (r *Reconciler) reconcileNATSBackend(ctx context.Context, eventing *operatorv1alpha1.Eventing, log *zap.SugaredLogger) (kctrl.Result, error) {
	// retrieves the NATS CRD
	_, err := r.kubeClient.GetCRD(ctx, k8s.NatsGVK.GroupResource().String())
	if err != nil {
		if kerrors.IsNotFound(err) {
			// delete the publisher proxy resources, because the publisher deployment will go
			// into CrashLoopBackOff.
			log.Infof("NATS module not enabled, deleting publisher proxy resources")
			delErr := r.eventingManager.DeletePublisherProxyResources(ctx, eventing)
			if delErr != nil {
				return kctrl.Result{}, delErr
			}
			// update the Eventing CR status.
			notFoundErr := fmt.Errorf("%w: %v", ErrNatsModuleMissing, err)
			return kctrl.Result{}, r.syncStatusWithNATSState(ctx, operatorv1alpha1.StateWarning, eventing,
				notFoundErr, log)
		}
		return kctrl.Result{}, err
	}

	if err = r.startNATSCRWatch(eventing); err != nil {
		return kctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	// check nats CR if it exists and is in natsAvailable state
	err = r.checkNATSAvailability(ctx, eventing)
	if err != nil {
		return kctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	// set NATSAvailable condition to true and update status
	eventing.Status.SetNATSAvailableConditionToTrue()

	deployment, err := r.handlePublisherProxy(ctx, eventing, eventing.Spec.Backend.Type)
	if err != nil {
		return kctrl.Result{}, r.syncStatusWithPublisherProxyErr(ctx, eventing, err, log)
	}

	// start NATS subscription manager
	if err := r.reconcileNATSSubManager(ctx, eventing, log); err != nil {
		return kctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	return r.handleEventingState(ctx, deployment, eventing, log)
}

func (r *Reconciler) checkNATSAvailability(ctx context.Context, eventing *operatorv1alpha1.Eventing) error {
	natsAvailable, err := r.eventingManager.IsNATSAvailable(ctx, eventing.Namespace)
	if err != nil {
		return err
	}
	if !natsAvailable {
		return ErrNATSServerUnavailable
	}
	return nil
}

func (r *Reconciler) handlePublisherProxy(
	ctx context.Context,
	eventing *operatorv1alpha1.Eventing,
	backendType operatorv1alpha1.BackendType,
) (*kappsv1.Deployment, error) {
	// get nats config with NATS server url
	var natsConfig *env.NATSConfig
	if backendType == operatorv1alpha1.NatsBackendType {
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

func (r *Reconciler) reconcileEventMeshBackend(ctx context.Context, eventing *operatorv1alpha1.Eventing, log *zap.SugaredLogger) (kctrl.Result, error) {
	// check if APIRule CRD is installed.
	isAPIRuleCRDEnabled, err := r.kubeClient.APIRuleCRDExists(ctx)
	if err != nil {
		return kctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
	} else if !isAPIRuleCRDEnabled {
		return kctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, ErrAPIGatewayModuleMissing, log)
	}

	// retrieve secret used to authenticate with EventMesh
	eventMeshSecret, err := r.kubeClient.GetSecret(ctx, eventing.Spec.Backend.Config.EventMeshSecret)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return kctrl.Result{}, r.syncSubManagerStatusWithNATSState(ctx, operatorv1alpha1.StateWarning, eventing,
				fmt.Errorf(EventMeshSecretMissingMessage), log)
		}
		return kctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing,
			fmt.Errorf("failed to get EventMesh secret: %v", err), log)
	}

	// Start the EventMesh subscription controller
	err = r.reconcileEventMeshSubManager(ctx, eventing, eventMeshSecret, log)
	if err != nil {
		return kctrl.Result{}, r.syncStatusWithSubscriptionManagerErr(ctx, eventing, err, log)
	}
	eventing.Status.SetSubscriptionManagerReadyConditionToTrue()

	deployment, err := r.handlePublisherProxy(ctx, eventing, eventing.Spec.Backend.Type)
	if err != nil {
		return kctrl.Result{}, r.syncStatusWithPublisherProxyErr(ctx, eventing, err, log)
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
