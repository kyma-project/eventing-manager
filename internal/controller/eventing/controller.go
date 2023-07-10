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

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	eceventingv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	ecbackend "github.com/kyma-project/kyma/components/eventing-controller/controllers/backend"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	FinalizerName       = "eventing.operator.kyma-project.io/finalizer"
	ControllerName      = "eventing-manager-controller"
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
	ctx          context.Context
	controller   controller.Controller
	natsConfig   env.NATSConfig
	scheme       *runtime.Scheme
	recorder     record.EventRecorder
	logger       *logger.Logger
	ctrlManager  ctrl.Manager
	ecReconciler *ecbackend.Reconciler
}

func NewReconciler(
	ctx context.Context,
	natsConfig env.NATSConfig,
	client client.Client,
	scheme *runtime.Scheme,
	logger *logger.Logger,
	recorder record.EventRecorder,
) *Reconciler {
	return &Reconciler{
		ctx:        ctx,
		natsConfig: natsConfig,
		Client:     client,
		scheme:     scheme,
		recorder:   recorder,
		logger:     logger,
		controller: nil,
		ecReconciler: ecbackend.NewReconciler(ctx, nil, natsConfig, env.GetConfig(), env.GetBackendConfig(), nil,
			client, logger, recorder),
	}
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=eventings/finalizers,verbs=update

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
	return r.namedLogger().With(
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
	// check nats CR if it exists and is in ready state
	ready, err := r.isNATSReady(ctx, eventing.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ready {
		return ctrl.Result{}, fmt.Errorf("NATS server is not ready in namespace %s", eventing.Namespace)
	}

	// TODO: change to support multiple backends in the future
	ecBackendType, err := convertECBackendType(eventing.Spec.Backends[0].Type)
	if err != nil {
		return ctrl.Result{}, err
	}

	updateNatsConfig(&r.natsConfig, eventing)

	backendConfig := env.GetBackendConfig()
	updatePublisherConfig(&backendConfig, eventing)

	// create an instance of ecbackend Reconciler
	r.ecReconciler = ecbackend.NewReconciler(
		ctx,
		nil,
		r.natsConfig,
		env.GetConfig(),
		backendConfig,
		nil,
		r.Client,
		r.logger,
		r.recorder,
	)

	// CreateOrUpdate deployment for publisher proxy
	deployment, err := r.ecReconciler.CreateOrUpdatePublisherProxy(ctx, ecBackendType)
	if err != nil {
		return ctrl.Result{}, err
	}

	// CreateOrUpdate HPA for publisher proxy deployment
	err = r.createOrUpdateHPA(ctx, deployment, eventing, 50)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) namedLogger() *zap.SugaredLogger {
	return r.logger.WithContext().Named(ControllerName)
}

// createOrUpdateHorizontalPodAutoscaler creates or updates the HPA for the given deployment.
func (r *Reconciler) createOrUpdateHPA(ctx context.Context, deployment *v1.Deployment, eventing *eventingv1alpha1.Eventing, cpuUtilization int32) error {
	// try to get the existing horizontal pod autoscaler object
	hpa := &autoscalingv1.HorizontalPodAutoscaler{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, hpa)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get horizontal pod autoscaler: %v", err)
	}
	min := int32(eventing.Spec.Publisher.Min)
	max := int32(eventing.Spec.Publisher.Max)
	// if the horizontal pod autoscaler object does not exist, create it
	if errors.IsNotFound(err) {
		// create a new horizontal pod autoscaler object
		hpa = createHorizontalPodAutoscaler(deployment, min, max, cpuUtilization)
		err = r.Client.Create(ctx, hpa)
		if err != nil {
			return fmt.Errorf("failed to create horizontal pod autoscaler: %v", err)
		}
		return nil
	}

	// if the horizontal pod autoscaler object exists, update it
	hpa.Spec.MinReplicas = &min
	hpa.Spec.MaxReplicas = max
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err = r.Client.Update(ctx, hpa)
		if err != nil {
			return fmt.Errorf("failed to update horizontal pod autoscaler: %v", err)
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("failed to update horizontal pod autoscaler: %v", retryErr)
	}

	return nil
}

func updateNatsConfig(natsConfig *env.NATSConfig, eventing *v1alpha1.Eventing) {
	natsConfig.JSStreamStorageType = eventing.Spec.Backends[0].Config.NATSStorageType
	natsConfig.JSStreamReplicas = eventing.Spec.Backends[0].Config.NATSStreamReplicas
	natsConfig.JSStreamMaxBytes = string(eventing.Spec.Backends[0].Config.MaxStreamSize.Value() * 1024 * 1024 * 1024)
	natsConfig.JSStreamMaxMsgsPerTopic = eventing.Spec.Backends[0].Config.MaxMsgsPerTopic
}

func updatePublisherConfig(backendConfig *env.BackendConfig, eventing *v1alpha1.Eventing) {
	backendConfig.PublisherConfig.RequestsCPU = eventing.Spec.Publisher.Resources.Requests.Cpu().String()
	backendConfig.PublisherConfig.RequestsMemory = eventing.Spec.Publisher.Resources.Requests.Memory().String()
	backendConfig.PublisherConfig.LimitsCPU = eventing.Spec.Publisher.Resources.Limits.Cpu().String()
	backendConfig.PublisherConfig.LimitsMemory = eventing.Spec.Publisher.Resources.Limits.Memory().String()
}

func convertECBackendType(backendType eventingv1alpha1.BackendType) (eceventingv1alpha1.BackendType, error) {
	switch backendType {
	case eventingv1alpha1.EventMeshBackendType:
		return eceventingv1alpha1.BEBBackendType, nil
	case eventingv1alpha1.NatsBackendType:
		return eceventingv1alpha1.NatsBackendType, nil
	default:
		return "", fmt.Errorf("unknown backend type: %s", backendType)
	}
}

func createHorizontalPodAutoscaler(deployment *v1.Deployment, min int32, max int32, cpuUtilization int32) *autoscalingv1.HorizontalPodAutoscaler {
	return &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       deployment.Name,
				APIVersion: "apps/v1",
			},
			MinReplicas:                    &min,
			MaxReplicas:                    max,
			TargetCPUUtilizationPercentage: &cpuUtilization,
		},
	}
}

func (r *Reconciler) isNATSReady(ctx context.Context, namespace string) (bool, error) {
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
