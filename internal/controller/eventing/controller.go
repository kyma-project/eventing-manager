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
	"fmt"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/object"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	FinalizerName       = "eventing.operator.kyma-project.io/finalizer"
	ControllerName      = "eventing-manager"
	ManagedByLabelKey   = "app.kubernetes.io/managed-by"
	ManagedByLabelValue = ControllerName
)

var (
	// allowedAnnotations are the publisher proxy deployment spec template annotations
	// which should be preserved during reconciliation.
	allowedAnnotations = map[string]string{
		"kubectl.kubernetes.io/restartedAt": "",
	}
)

// Reconciler reconciles a Eventing object
type Reconciler struct {
	client.Client
	controller  controller.Controller
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
	logger      *zap.SugaredLogger
	ctrlManager ctrl.Manager
	backendType eventingv1alpha1.BackendType
}

func NewReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	logger *zap.SugaredLogger,
	recorder record.EventRecorder,
) *Reconciler {
	return &Reconciler{
		Client:     client,
		scheme:     scheme,
		recorder:   recorder,
		logger:     logger,
		controller: nil,
	}
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Info("Reconciliation triggered")
	// fetch latest subscription object
	currentEventing := &eventingv1alpha1.Eventing{}
	if err := r.Get(ctx, req.NamespacedName, currentEventing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// copy the object, so we don't modify the source object
	eventing := currentEventing.DeepCopy()

	// logger with eventing details
	log := r.loggerWithEventing(eventing)

	// check if eventing is in deletion state
	if !eventing.DeletionTimestamp.IsZero() {
		return r.handleEventingDeletion(ctx, eventing, log)
	}

	// handle reconciliation
	return r.handleEventingReconcile(ctx, eventing, log)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ctrlManager = mgr
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventingv1alpha1.Eventing{}).
		Complete(r)
}

// loggerWithEventing returns a logger with the given Eventing CR details.
func (r *Reconciler) loggerWithEventing(eventing *eventingv1alpha1.Eventing) *zap.SugaredLogger {
	return r.logger.With(
		"kind", eventing.GetObjectKind().GroupVersionKind().Kind,
		"resourceVersion", eventing.GetResourceVersion(),
		"generation", eventing.GetGeneration(),
		"namespace", eventing.GetNamespace(),
		"name", eventing.GetName(),
	)
}

func (r *Reconciler) handleEventingDeletion(_ context.Context, _ *eventingv1alpha1.Eventing,
	log *zap.SugaredLogger) (ctrl.Result, error) {
	log.Info("handling Eventing deletion...")
	// TODO: Implement me.
	return ctrl.Result{}, nil
}

func (r *Reconciler) handleEventingReconcile(ctx context.Context,
	eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) (ctrl.Result, error) {
	log.Info("handling Eventing reconciliation...")
	// TODO: Implement EventMesh reconciliation.

	// just to use the variables.
	log.Info(FinalizerName, ManagedByLabelKey, ManagedByLabelValue)

	return r.reconcileNATSBackend(ctx, eventing)
}

func (r *Reconciler) reconcileNATSBackend(ctx context.Context, eventing *eventingv1alpha1.Eventing) (ctrl.Result, error) {

	// TODO: check nats CR if it exists and is in ready state

	// create or install Eventing Publisher Proxy
	// TODO: change to support multiple backends in the future
	backendType := eventing.Spec.Backends[0].Type
	r.CreateOrUpdatePublisherProxy(ctx, backendType)

	return ctrl.Result{}, nil
}

func (r *Reconciler) CreateOrUpdatePublisherProxy(ctx context.Context, backendType eventingv1alpha1.BackendType) (*appsv1.Deployment, error) {
	var desiredPublisher *appsv1.Deployment
	switch backendType {
	case eventingv1alpha1.NatsBackendType:
		// new env.NATSConfig
		opts := options.New()
		if err := opts.Parse(); err != nil {
			r.logger.Fatalf("Failed to parse options, error: %v", err)
		}
		natsConfig, err := env.GetNATSConfig(opts.MaxReconnects, opts.ReconnectWait)
		if err != nil {
			r.logger.Fatalw("Failed to load configuration", "error", err)
		}
		cfg := env.GetBackendConfig()
		desiredPublisher = deployment.NewNATSPublisherDeployment(natsConfig, cfg.PublisherConfig)
	case eventingv1alpha1.EventMeshBackendType:
		// TODO: create EventMesh publisher deployment
	default:
		return nil, fmt.Errorf("unknown EventingBackend type %q", backendType)
	}

	if err := r.setAsOwnerReference(ctx, desiredPublisher); err != nil {
		return nil, fmt.Errorf("set owner reference for Event Publisher failed: %w", err)
	}

	currentPublisher, err := r.getEPPDeployment(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Event Publisher deployment")
	}

	if currentPublisher == nil { // no deployment found
		// delete the publisher proxy with invalid backend type if it still exists
		if err := r.deletePublisherProxy(ctx); err != nil {
			return nil, err
		}
		// Create
		r.logger.Debug("Creating Event Publisher deployment")
		return desiredPublisher, r.Create(ctx, desiredPublisher)
	}

	desiredPublisher.ResourceVersion = currentPublisher.ResourceVersion

	// preserve only allowed annotations
	desiredPublisher.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	for k, v := range currentPublisher.Spec.Template.ObjectMeta.Annotations {
		if _, ok := allowedAnnotations[k]; ok {
			desiredPublisher.Spec.Template.ObjectMeta.Annotations[k] = v
		}
	}

	if object.Semantic.DeepEqual(currentPublisher, desiredPublisher) {
		return currentPublisher, nil
	}

	// Update publisher proxy deployment
	if err := r.Update(ctx, desiredPublisher); err != nil {
		return nil, errors.Wrapf(err, "update Event Publisher deployment failed")
	}

	return desiredPublisher, nil
}

// getEPPDeployment fetches the event publisher by the current active backend type.
func (r *Reconciler) getEPPDeployment(ctx context.Context) (*appsv1.Deployment, error) {
	var list appsv1.DeploymentList
	if err := r.List(ctx, &list, client.MatchingLabels{
		deployment.AppLabelKey:       deployment.PublisherName,
		deployment.InstanceLabelKey:  deployment.InstanceLabelValue,
		deployment.DashboardLabelKey: deployment.DashboardLabelValue,
		deployment.BackendLabelKey:   fmt.Sprint(r.backendType),
	}); err != nil {
		return nil, err
	}

	if len(list.Items) == 0 { // no deployment found
		return nil, nil
	}
	return &list.Items[0], nil
}

// deletePublisherProxy removes the existing publisher proxy.
func (r *Reconciler) deletePublisherProxy(ctx context.Context) error {
	publisherNamespacedName := types.NamespacedName{
		Namespace: deployment.PublisherNamespace,
		Name:      deployment.PublisherName,
	}
	publisher := new(appsv1.Deployment)
	err := r.Get(ctx, publisherNamespacedName, publisher)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	r.logger.Debug("Event Publisher with invalid backend type found, deleting it")
	err = r.Delete(ctx, publisher)
	return err
}

func (r *Reconciler) setAsOwnerReference(ctx context.Context, obj metav1.Object) error {
	controllerNamespacedName := types.NamespacedName{
		Namespace: deployment.ControllerNamespace,
		Name:      deployment.ControllerName,
	}
	var deploymentController appsv1.Deployment
	if err := r.Get(ctx, controllerNamespacedName, &deploymentController); err != nil {
		r.logger.Errorw("get controller NamespacedName failed", "error", err)
		return err
	}
	references := []metav1.OwnerReference{
		*metav1.NewControllerRef(&deploymentController, schema.GroupVersionKind{
			Group:   appsv1.SchemeGroupVersion.Group,
			Version: appsv1.SchemeGroupVersion.Version,
			Kind:    "Deployment",
		}),
	}
	obj.SetOwnerReferences(references)
	return nil
}
