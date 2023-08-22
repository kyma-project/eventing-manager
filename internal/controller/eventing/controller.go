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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/object"
	ecsubscriptionmanager "github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager"
	v1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FinalizerName             = "eventing.operator.kyma-project.io/finalizer"
	ControllerName            = "eventing-manager-controller"
	ManagedByLabelKey         = "app.kubernetes.io/managed-by"
	ManagedByLabelValue       = ControllerName
	NatsServerNotAvailableMsg = "NATS server is not available"
	natsClientPort            = 4222

	BEBBackendSecretLabelKey   = "kyma-project.io/eventing-backend"
	BEBBackendSecretLabelValue = "beb"

	BEBSecretNameSuffix = "-beb-oauth2"

	BackendCRLabelKey   = "kyma-project.io/eventing"
	BackendCRLabelValue = "backend"

	AppLabelValue             = deployment.PublisherName
	PublisherSecretEMSHostKey = "ems-publish-host"

	TokenEndpointFormat             = "%s?grant_type=%s&response_type=token"
	NamespacePrefix                 = "/"
	BEBPublishEndpointForSubscriber = "/sap/ems/v1"
	BEBPublishEndpointForPublisher  = "/sap/ems/v1/events"
)

// Reconciler reconciles an Eventing object
//
//go:generate mockery --name=Controller --dir=../../../vendor/sigs.k8s.io/controller-runtime/pkg/controller --outpkg=mocks --case=underscore
//go:generate mockery --name=Manager --dir=../../../vendor/sigs.k8s.io/controller-runtime/pkg/manager --outpkg=mocks --case=underscore
type Reconciler struct {
	client.Client
	logger                  *logger.Logger
	ctrlManager             ctrl.Manager
	eventingManager         eventing.Manager
	kubeClient              k8s.Client
	scheme                  *runtime.Scheme
	recorder                record.EventRecorder
	subManagerFactory       subscriptionmanager.ManagerFactory
	natsSubManager          ecsubscriptionmanager.Manager
	isNATSSubManagerStarted bool
	natsConfigHandler       NatsConfigHandler
}

func NewReconciler(
	client client.Client,
	kubeClient k8s.Client,
	scheme *runtime.Scheme,
	logger *logger.Logger,
	recorder record.EventRecorder,
	manager eventing.Manager,
	subManagerFactory subscriptionmanager.ManagerFactory,
	opts *options.Options,
) *Reconciler {
	return &Reconciler{
		Client:                  client,
		logger:                  logger,
		ctrlManager:             nil, // ctrlManager will be initialized in `SetupWithManager`.
		eventingManager:         manager,
		kubeClient:              kubeClient,
		scheme:                  scheme,
		recorder:                recorder,
		subManagerFactory:       subManagerFactory,
		natsSubManager:          nil,
		isNATSSubManagerStarted: false,
		natsConfigHandler:       NewNatsConfigHandler(kubeClient, opts),
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
//+kubebuilder:rbac:groups="applicationconnector.kyma-project.io",resources=applications,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="eventing.kyma-project.io",resources=subscriptions,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups=eventing.kyma-project.io,resources=subscriptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources=subscriptions,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=subscriptions/status,verbs=get;update;patch
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
		Owns(&v1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
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

func (r *Reconciler) handleEventingDeletion(ctx context.Context, eventing *eventingv1alpha1.Eventing,
	log *zap.SugaredLogger) (ctrl.Result, error) {
	// skip reconciliation for deletion if the finalizer is not set.
	if !r.containsFinalizer(eventing) {
		log.Debug("skipped reconciliation for deletion as finalizer is not set.")
		return ctrl.Result{}, nil
	}

	log.Info("handling Eventing deletion...")
	if err := r.stopNATSSubManager(true, log); err != nil {
		return ctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	// TODO: Implement me, this is a dummy implementation for testing.

	return r.removeFinalizer(ctx, eventing)
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

	for _, backend := range eventing.Spec.Backends {
		switch backend.Type {
		case eventingv1alpha1.NatsBackendType:
			return r.reconcileNATSBackend(ctx, eventing, log)
		case eventingv1alpha1.EventMeshBackendType:
			return r.reconcileEventMeshBackend(ctx, eventing, log)
		default:
			return ctrl.Result{Requeue: false}, fmt.Errorf("not supported backend type %s", backend.Type)
		}
	}
	// this should never happen, but if happens do nothing
	return ctrl.Result{Requeue: false}, fmt.Errorf("no backend is provided in the spec")
}

func (r *Reconciler) reconcileNATSBackend(ctx context.Context, eventing *eventingv1alpha1.Eventing, log *zap.SugaredLogger) (ctrl.Result, error) {
	// check nats CR if it exists and is in natsAvailable state
	err := r.checkNATSAvailability(ctx, eventing)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithNATSErr(ctx, eventing, err, log)
	}

	// set NATSAvailable condition to true and update status
	eventing.Status.SetNATSAvailableConditionToTrue()

	deployment, err := r.handlePublisherProxy(ctx, eventing, eventing.GetNATSBackend().Type)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithPublisherProxyErr(ctx, eventing, err, log)
	}

	// start NATS subscription manager
	if err := r.reconcileNATSSubManager(eventing, log); err != nil {
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
	natsConfig, err := r.natsConfigHandler.GetNatsConfig(ctx, *eventing)
	if err != nil {
		return nil, err
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
	// retrieve secret to authenticate with EventMesh
	substrings := strings.Split(eventing.GetEventMeshBackend().Config.EventMeshSecret, "/")
	eventMeshSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: substrings[0], Name: substrings[1]}, eventMeshSecret); err != nil {
		return ctrl.Result{}, err
	}

	// CreateOrUpdate deployment for publisher proxy secret
	secretForPublisher, err := r.SyncPublisherProxySecret(ctx, eventMeshSecret)
	if err != nil {
		// backendStatus.SetPublisherReadyCondition(false, eventingv1alpha1.ConditionReasonPublisherProxySecretError, err.Error())
		// if updateErr := r.syncBackendStatus(ctx, backendStatus, nil); updateErr != nil {
		// return ctrl.Result{}, fmt.Errorf("failed to update status while syncing Event Publisher secret: %v", err)
		// }
		return ctrl.Result{}, err
	}

	// Set environment with secrets for BEB subscription controller
	err = setUpEnvironmentForBEBController(secretForPublisher)
	if err != nil {
		// backendStatus.SetSubscriptionControllerReadyCondition(false, eventingv1alpha1.ConditionReasonControllerStartFailed, err.Error())
		// if updateErr := r.syncBackendStatus(ctx, backendStatus, nil); updateErr != nil {
		// 	return ctrl.Result{},fmt.Errorf(err, "failed to update status while setting up environment variables for BEB controller")
		// }
		return ctrl.Result{}, fmt.Errorf("failed to setup environment variables for EventMesh controller: %v", err)
	}

	//TODO: Start the EventMesh subscription controller

	_, err = r.handlePublisherProxy(ctx, eventing, eventing.GetEventMeshBackend().Type)
	if err != nil {
		return ctrl.Result{}, r.syncStatusWithPublisherProxyErr(ctx, eventing, err, log)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) namedLogger() *zap.SugaredLogger {
	return r.logger.WithContext().Named(ControllerName)
}

func (r *Reconciler) SyncPublisherProxySecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
	secretNamespacedName := types.NamespacedName{
		Namespace: deployment.PublisherNamespace,
		Name:      deployment.PublisherName,
	}
	currentSecret := new(corev1.Secret)

	desiredSecret, err := getSecretForPublisher(secret)
	if err != nil {
		return nil, fmt.Errorf("invalid secret for Event Publisher: %v", err)
	}
	err = r.Get(ctx, secretNamespacedName, currentSecret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create secret
			r.namedLogger().Debug("Creating secret for BEB publisher")
			err := r.Create(ctx, desiredSecret)
			if err != nil {
				return nil, fmt.Errorf("create secret for Event Publisher failed: %v", err)
			}
			return desiredSecret, nil
		}
		return nil, fmt.Errorf("Failed to get Event Publisher secret failed: %v", err)
	}

	if object.Semantic.DeepEqual(currentSecret, desiredSecret) {
		r.namedLogger().Debug("No need to update secret for BEB Event Publisher")
		return currentSecret, nil
	}

	// Update secret
	desiredSecret.ResourceVersion = currentSecret.ResourceVersion
	if err := r.Update(ctx, desiredSecret); err != nil {
		return nil, fmt.Errorf("failed to update Event Publisher secret: %v", err)
	}

	return desiredSecret, nil
}

func newSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getSecretForPublisher(bebSecret *corev1.Secret) (*corev1.Secret, error) {
	secret := newSecret(deployment.PublisherName, deployment.PublisherNamespace)

	secret.Labels = map[string]string{
		deployment.AppLabelKey: AppLabelValue,
	}

	if _, ok := bebSecret.Data["messaging"]; !ok {
		return nil, errors.New("message is missing from BEB secret")
	}
	messagingBytes := bebSecret.Data["messaging"]

	if _, ok := bebSecret.Data["namespace"]; !ok {
		return nil, errors.New("namespace is missing from BEB secret")
	}
	namespaceBytes := bebSecret.Data["namespace"]

	var messages []Message
	err := json.Unmarshal(messagingBytes, &messages)
	if err != nil {
		return nil, err
	}

	for _, m := range messages {
		if m.Broker.BrokerType == "saprestmgw" {
			if len(m.OA2.ClientID) == 0 {
				return nil, errors.New("client ID is missing")
			}
			if len(m.OA2.ClientSecret) == 0 {
				return nil, errors.New("client secret is missing")
			}
			if len(m.OA2.TokenEndpoint) == 0 {
				return nil, errors.New("tokenendpoint is missing")
			}
			if len(m.OA2.GrantType) == 0 {
				return nil, errors.New("granttype is missing")
			}
			if len(m.URI) == 0 {
				return nil, errors.New("publish URL is missing")
			}

			secret.StringData = getSecretStringData(m.OA2.ClientID, m.OA2.ClientSecret, m.OA2.TokenEndpoint, m.OA2.GrantType, m.URI, string(namespaceBytes))
			break
		}
	}

	return secret, nil
}

func getSecretStringData(clientID, clientSecret, tokenEndpoint, grantType, publishURL, namespace string) map[string]string {
	return map[string]string{
		deployment.PublisherSecretClientIDKey:      clientID,
		deployment.PublisherSecretClientSecretKey:  clientSecret,
		deployment.PublisherSecretTokenEndpointKey: fmt.Sprintf(TokenEndpointFormat, tokenEndpoint, grantType),
		deployment.PublisherSecretEMSURLKey:        fmt.Sprintf("%s%s", publishURL, BEBPublishEndpointForPublisher),
		PublisherSecretEMSHostKey:                  publishURL,
		deployment.PublisherSecretBEBNamespaceKey:  namespace,
	}
}

func setUpEnvironmentForBEBController(secret *corev1.Secret) error {
	err := os.Setenv("BEB_API_URL", fmt.Sprintf("%s%s", string(secret.Data[PublisherSecretEMSHostKey]), BEBPublishEndpointForSubscriber))
	if err != nil {
		return fmt.Errorf("set BEB_API_URL env var failed: %v", err)
	}

	err = os.Setenv("CLIENT_ID", string(secret.Data[deployment.PublisherSecretClientIDKey]))
	if err != nil {
		return fmt.Errorf("set CLIENT_ID env var failed: %v", err)
	}

	err = os.Setenv("CLIENT_SECRET", string(secret.Data[deployment.PublisherSecretClientSecretKey]))
	if err != nil {
		return fmt.Errorf("set CLIENT_SECRET env var failed: %v", err)
	}

	err = os.Setenv("TOKEN_ENDPOINT", string(secret.Data[deployment.PublisherSecretTokenEndpointKey]))
	if err != nil {
		return fmt.Errorf("set TOKEN_ENDPOINT env var failed: %v", err)
	}

	err = os.Setenv("BEB_NAMESPACE", fmt.Sprintf("%s%s", NamespacePrefix, string(secret.Data[deployment.PublisherSecretBEBNamespaceKey])))
	if err != nil {
		return fmt.Errorf("set BEB_NAMESPACE env var failed: %v", err)
	}

	return nil
}
