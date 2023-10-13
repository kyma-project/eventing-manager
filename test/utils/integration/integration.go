package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager"
	subscriptionmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks"
	ecsubmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks/ec"
	ecsubscriptionmanager "github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager"
	"github.com/stretchr/testify/mock"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/kyma-project/eventing-manager/test"

	"github.com/avast/retry-go/v3"
	"github.com/go-logr/zapr"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingctrl "github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	evnttestutils "github.com/kyma-project/eventing-manager/test/utils"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apiextensionsv1clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
)

const (
	useExistingCluster       = true
	attachControlPlaneOutput = false
	testEnvStartDelay        = time.Minute
	testEnvStartAttempts     = 10
	BigPollingInterval       = 3 * time.Second
	BigTimeOut               = 60 * time.Second
	SmallTimeOut             = 6 * time.Second
	SmallPollingInterval     = 1 * time.Second
)

// TestEnvironment provides mocked resources for integration tests.
type TestEnvironment struct {
	Context             context.Context
	EnvTestInstance     *envtest.Environment
	k8sClient           client.Client
	KubeClient          k8s.Client
	K8sDynamicClient    *dynamic.DynamicClient
	Reconciler          *eventingctrl.Reconciler
	Logger              *logger.Logger
	Recorder            *record.EventRecorder
	TestCancelFn        context.CancelFunc
	SubManagerFactory   subscriptionmanager.ManagerFactory
	JetStreamSubManager ecsubscriptionmanager.Manager
}

//nolint:funlen // Used in testing
func NewTestEnvironment(projectRootDir string, celValidationEnabled bool,
	allowedEventingCR *eventingv1alpha1.Eventing) (*TestEnvironment, error) {
	var err error
	// setup context
	ctx := context.Background()

	opts := options.New()
	if err := opts.Parse(); err != nil {
		return nil, err
	}

	// setup logger
	ctrLogger, err := logger.New(opts.LogFormat, opts.LogLevel)
	if err != nil {
		return nil, err
	}
	// Set controller core logger.
	ctrl.SetLogger(zapr.NewLogger(ctrLogger.WithContext().Desugar()))

	testEnv, envTestKubeCfg, err := StartEnvTest(projectRootDir, celValidationEnabled)
	if err != nil {
		return nil, err
	}

	// add Eventing CRD scheme
	err = eventingv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	err = natsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	// +kubebuilder:scaffold:scheme

	k8sClient, err := client.New(envTestKubeCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(envTestKubeCfg)
	if err != nil {
		return nil, err
	}

	apiextensionsclient, err := apiextensionsv1clientset.NewForConfig(envTestKubeCfg)
	if err != nil {
		panic(err.Error())
	}

	// setup ctrl manager
	metricsPort, err := testutils.GetFreePort()
	if err != nil {
		return nil, err
	}

	ctrlMgr, err := ctrl.NewManager(envTestKubeCfg, ctrl.Options{
		Scheme:                 scheme.Scheme,
		Port:                   metricsPort,
		MetricsBindAddress:     "0", // disable
		HealthProbeBindAddress: "0", // disable
	})
	if err != nil {
		return nil, err
	}
	recorder := ctrlMgr.GetEventRecorderFor("eventing-manager-test")

	os.Setenv("WEBHOOK_TOKEN_ENDPOINT", "https://oauth2.ev-manager.kymatunas.shoot.canary.k8s-hana.ondemand.com/oauth2/token")
	os.Setenv("DOMAIN", "my.test.domain")

	// create k8s clients.
	kubeClient := k8s.NewKubeClient(ctrlMgr.GetClient(), dynamicClient, apiextensionsclient, "eventing-manager")

	// get backend configs.
	backendConfig := env.GetBackendConfig()

	// create eventing manager instance.
	eventingManager := eventing.NewEventingManager(ctx, k8sClient, kubeClient, backendConfig, ctrLogger, recorder)

	// define JetStream subscription manager mock.
	jetStreamSubManagerMock := new(ecsubmanagermocks.Manager)
	jetStreamSubManagerMock.On("Init", mock.Anything).Return(nil)
	jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil)
	jetStreamSubManagerMock.On("Stop", mock.Anything).Return(nil)

	// define EventMesh subscription manager mock.
	eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
	eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil)
	eventMeshSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil)
	eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil)

	// define subscription manager factory mock.
	subManagerFactoryMock := new(subscriptionmanagermocks.ManagerFactory)
	subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(jetStreamSubManagerMock)
	subManagerFactoryMock.On("NewEventMeshManager").Return(eventMeshSubManagerMock, nil)

	// create a new watcher
	eventingReconciler := eventingctrl.NewReconciler(
		k8sClient,
		kubeClient,
		dynamicClient,
		ctrlMgr.GetScheme(),
		ctrLogger,
		ctrlMgr.GetEventRecorderFor("eventing-manager-test"),
		eventingManager,
		backendConfig,
		subManagerFactoryMock,
		opts,
		allowedEventingCR,
	)

	if err = (eventingReconciler).SetupWithManager(ctrlMgr); err != nil {
		return nil, err
	}

	// start manager
	var cancelCtx context.CancelFunc
	go func() {
		var mgrCtx context.Context
		mgrCtx, cancelCtx = context.WithCancel(ctrl.SetupSignalHandler())
		err = ctrlMgr.Start(mgrCtx)
		if err != nil {
			panic(err)
		}
	}()

	// create namespace
	ns := testutils.NewNamespace(getTestBackendConfig().Namespace)
	if err = client.IgnoreAlreadyExists(k8sClient.Create(ctx, ns)); err != nil {
		return nil, err
	}

	// create webhook cert secret.
	newCABundle := make([]byte, 40)
	if _, err := rand.Read(newCABundle); err != nil {
		return nil, err
	}
	err = k8sClient.Create(ctx, newSecretWithTLSSecret(newCABundle))
	if err != nil {
		return nil, err
	}

	return &TestEnvironment{
		Context:             ctx,
		k8sClient:           k8sClient,
		KubeClient:          kubeClient,
		K8sDynamicClient:    dynamicClient,
		Reconciler:          eventingReconciler,
		Logger:              ctrLogger,
		Recorder:            &recorder,
		EnvTestInstance:     testEnv,
		TestCancelFn:        cancelCtx,
		SubManagerFactory:   subManagerFactoryMock,
		JetStreamSubManager: jetStreamSubManagerMock,
	}, nil
}

func StartEnvTest(projectRootDir string, celValidationEnabled bool) (*envtest.Environment, *rest.Config, error) {
	// Reference: https://book.kubebuilder.io/reference/envtest.html
	useExistingCluster := useExistingCluster

	dummyCABundle := make([]byte, 20)
	if _, err := rand.Read(dummyCABundle); err != nil {
		return nil, nil, err
	}

	url := "https://eventing-controller.kyma-system.svc.cluster.local"
	sideEffectClassNone := admissionv1.SideEffectClassNone
	mwh := getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
		{
			Name: "reconciler.eventing.test",
			ClientConfig: admissionv1.WebhookClientConfig{
				URL:      &url,
				CABundle: dummyCABundle,
			},
			SideEffects:             &sideEffectClassNone,
			AdmissionReviewVersions: []string{"v1beta1"},
		},
	})
	mwh.Name = getTestBackendConfig().MutatingWebhookName

	// setup dummy validating webhook
	vwh := getValidatingWebhookConfig([]admissionv1.ValidatingWebhook{
		{
			Name: "reconciler2.eventing.test",
			ClientConfig: admissionv1.WebhookClientConfig{
				URL:      &url,
				CABundle: dummyCABundle,
			},
			SideEffects:             &sideEffectClassNone,
			AdmissionReviewVersions: []string{"v1beta1"},
		},
	})
	vwh.Name = getTestBackendConfig().ValidatingWebhookName

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(projectRootDir, "config", "crd", "bases"),
			filepath.Join(projectRootDir, "config", "crd", "external"),
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: attachControlPlaneOutput,
		UseExistingCluster:       &useExistingCluster,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			MutatingWebhooks:   []*admissionv1.MutatingWebhookConfiguration{mwh},
			ValidatingWebhooks: []*admissionv1.ValidatingWebhookConfiguration{vwh},
		},
	}

	args := testEnv.ControlPlane.GetAPIServer().Configure()
	if celValidationEnabled {
		args.Set("feature-gates", "CustomResourceValidationExpressions=true")
	} else {
		args.Set("feature-gates", "CustomResourceValidationExpressions=false")
	}

	var cfg *rest.Config
	err := retry.Do(func() error {
		defer func() {
			if r := recover(); r != nil {
				log.Println("panic recovered:", r)
			}
		}()
		cfgLocal, startErr := testEnv.Start()
		cfg = cfgLocal
		return startErr
	},
		retry.Delay(testEnvStartDelay),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(testEnvStartAttempts),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("[%v] try failed to start testenv: %s", n, err)
			if stopErr := testEnv.Stop(); stopErr != nil {
				log.Printf("failed to stop testenv: %s", stopErr)
			}
		}),
	)
	return testEnv, cfg, err
}

func (env TestEnvironment) TearDown() error {
	if env.TestCancelFn != nil {
		env.TestCancelFn()
	}

	// clean-up created resources
	err := env.DeleteSecretFromK8s(getTestBackendConfig().WebhookSecretName, getTestBackendConfig().Namespace)
	if err != nil {
		fmt.Printf("couldn't clean the webhook secret: %s", err)
	}

	// retry to stop the api-server
	sleepTime := 1 * time.Second
	const retries = 20
	for i := 0; i < retries; i++ {
		if err = env.EnvTestInstance.Stop(); err == nil {
			break
		}
		time.Sleep(sleepTime)
	}
	return err
}

// GetEventingAssert fetches Eventing from k8s and allows making assertions on it.
func (env TestEnvironment) GetEventingAssert(g *gomega.GomegaWithT,
	eventing *eventingv1alpha1.Eventing) gomega.AsyncAssertion {
	return g.Eventually(func() *eventingv1alpha1.Eventing {
		gotEventing, err := env.GetEventingFromK8s(eventing.Name, eventing.Namespace)
		if err != nil {
			log.Printf("fetch eventing %s/%s failed: %v", eventing.Name, eventing.Namespace, err)
			return nil
		}
		return gotEventing
	}, BigTimeOut, BigPollingInterval)
}

func (env TestEnvironment) EnsureNamespaceCreation(t *testing.T, namespace string) {
	if namespace == "default" {
		return
	}
	// create namespace
	ns := testutils.NewNamespace(namespace)
	require.NoError(t, client.IgnoreAlreadyExists(env.k8sClient.Create(env.Context, ns)))
}

func (env TestEnvironment) CreateK8sResource(obj client.Object) error {
	return env.k8sClient.Create(env.Context, obj)
}

func (env TestEnvironment) EnsureK8sResourceCreated(t *testing.T, obj client.Object) {
	require.NoError(t, env.k8sClient.Create(env.Context, obj))
}

func (env TestEnvironment) EnsureEPPK8sResourcesExists(t *testing.T, eventingCR v1alpha1.Eventing) {
	env.EnsureK8sServiceExists(t,
		eventing.GetPublisherPublishServiceName(eventingCR), eventingCR.Namespace)
	env.EnsureK8sServiceExists(t,
		eventing.GetPublisherMetricsServiceName(eventingCR), eventingCR.Namespace)
	env.EnsureK8sServiceExists(t,
		eventing.GetPublisherHealthServiceName(eventingCR), eventingCR.Namespace)
	env.EnsureK8sServiceAccountExists(t,
		eventing.GetPublisherServiceAccountName(eventingCR), eventingCR.Namespace)
	env.EnsureK8sClusterRoleExists(t,
		eventing.GetPublisherClusterRoleName(eventingCR), eventingCR.Namespace)
	env.EnsureK8sClusterRoleBindingExists(t,
		eventing.GetPublisherClusterRoleBindingName(eventingCR), eventingCR.Namespace)
}

func (env TestEnvironment) EnsureEPPK8sResourcesHaveOwnerReference(t *testing.T, eventingCR v1alpha1.Eventing) {
	env.EnsureEPPPublishServiceOwnerReferenceSet(t, eventingCR)
	env.EnsureEPPMetricsServiceOwnerReferenceSet(t, eventingCR)
	env.EnsureEPPHealthServiceOwnerReferenceSet(t, eventingCR)
	env.EnsureEPPServiceAccountOwnerReferenceSet(t, eventingCR)
}

func (env TestEnvironment) EnsureDeploymentExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetDeploymentFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of Deployment")
}

func (env TestEnvironment) EnsureK8sServiceExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of Service")
}

func (env TestEnvironment) EnsureK8sServiceAccountExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceAccountFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of ServiceAccount")
}

func (env TestEnvironment) EnsureK8sClusterRoleExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetClusterRoleFromK8s(name, namespace)
		return err == nil && result != nil
	}, BigTimeOut, BigPollingInterval, "failed to ensure existence of ClusterRole")
}

func (env TestEnvironment) EnsureK8sClusterRoleBindingExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetClusterRoleBindingFromK8s(name, namespace)
		return err == nil && result != nil
	}, BigTimeOut, BigPollingInterval, "failed to ensure existence of ClusterRoleBinding")
}

func (env TestEnvironment) EnsureHPAExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetHPAFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of HPA")
}

func (env TestEnvironment) EnsureK8sResourceUpdated(t *testing.T, obj client.Object) {
	require.NoError(t, env.k8sClient.Update(env.Context, obj))
}

func (env TestEnvironment) EnsureK8sResourceDeleted(t *testing.T, obj client.Object) {
	require.NoError(t, env.k8sClient.Delete(env.Context, obj))
}

func (env TestEnvironment) EnsureNamespaceDeleted(t *testing.T, namespace string) {
	require.NoError(t, env.k8sClient.Delete(env.Context, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}))
}

func (env TestEnvironment) EnsureDeploymentDeletion(t *testing.T, name, namespace string) {
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	env.EnsureK8sResourceDeleted(t, deployment)
	require.Eventually(t, func() bool {
		_, err := env.GetDeploymentFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure deletion of Deployment")
}

func (env TestEnvironment) EnsureDeploymentNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetDeploymentFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure deletion of Deployment")
}

func (env TestEnvironment) EnsureK8sServiceNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetServiceFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure non-existence of Service")
}

func (env TestEnvironment) EnsureK8sServiceAccountNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetServiceAccountFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure non-existence of ServiceAccount")
}

func (env TestEnvironment) EnsureK8sClusterRoleNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetClusterRoleFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure non-existence of ClusterRole")
}

func (env TestEnvironment) EnsureK8sClusterRoleBindingNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetClusterRoleBindingFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure non-existence of ClusterRoleBinding")
}

func (env TestEnvironment) EnsureHPADeletion(t *testing.T, name, namespace string) {
	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	env.EnsureK8sResourceDeleted(t, hpa)
	require.Eventually(t, func() bool {
		_, err := env.GetHPAFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure deletion of HPA")
}

func (env TestEnvironment) EnsureHPANotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetHPAFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure deletion of HPA")
}

func (env TestEnvironment) EnsureEventingResourceDeletion(t *testing.T, name, namespace string) {
	eventing := &eventingv1alpha1.Eventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	env.EnsureK8sResourceDeleted(t, eventing)
	require.Eventually(t, func() bool {
		_, err := env.GetEventingFromK8s(name, namespace)
		return err != nil && errors.IsNotFound(err)
	}, BigTimeOut, BigPollingInterval, "failed to ensure deletion of Eventing")
}

func (env TestEnvironment) EnsureEventingResourceDeletionStateError(t *testing.T, name, namespace string) {
	eventing := &eventingv1alpha1.Eventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	env.EnsureK8sResourceDeleted(t, eventing)
	require.Eventually(t, func() bool {
		err := env.k8sClient.Get(env.Context, types.NamespacedName{Name: name, Namespace: namespace}, eventing)
		return err == nil && eventing.Status.State == eventingv1alpha1.StateError
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure deletion of Eventing")
}

func (env TestEnvironment) EnsureNATSResourceStateReady(t *testing.T, nats *natsv1alpha1.NATS) {
	env.makeNatsCrReady(t, nats)
	require.Eventually(t, func() bool {
		err := env.k8sClient.Get(env.Context, types.NamespacedName{Name: nats.Name, Namespace: nats.Namespace}, nats)
		return err == nil && nats.Status.State == natsv1alpha1.StateReady
	}, BigTimeOut, BigPollingInterval, "failed to ensure NATS CR is stored")
}

func (env TestEnvironment) EnsureNATSResourceStateError(t *testing.T, nats *natsv1alpha1.NATS) {
	env.makeNatsCrError(t, nats)
	require.Eventually(t, func() bool {
		err := env.k8sClient.Get(env.Context, types.NamespacedName{Name: nats.Name, Namespace: nats.Namespace}, nats)
		return err == nil && nats.Status.State == natsv1alpha1.StateError
	}, BigTimeOut, BigPollingInterval, "failed to ensure NATS CR is stored")
}

func (env TestEnvironment) EnsureEventingSpecPublisherReflected(t *testing.T, eventingCR *v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		deployment, err := env.GetDeploymentFromK8s(eventing.GetPublisherDeploymentName(*eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to get Eventing resource", "error", err,
				"name", eventingCR.Name, "namespace", eventingCR.Namespace)
		}

		eventTypePrefix := test.FindEnvVar(deployment.Spec.Template.Spec.Containers[0].Env, "EVENT_TYPE_PREFIX")
		eventTypePrefixCheck := eventTypePrefix != nil && eventTypePrefix.Value == eventingCR.Spec.Backend.Config.EventTypePrefix
		return eventingCR.Spec.Publisher.Resources.Limits.Cpu().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu()) &&
			eventingCR.Spec.Publisher.Resources.Limits.Memory().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory()) &&
			eventingCR.Spec.Publisher.Resources.Requests.Cpu().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu()) &&
			eventingCR.Spec.Publisher.Resources.Requests.Memory().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory()) &&
			eventTypePrefixCheck
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure Eventing spec publisher is reflected")
}

func (env TestEnvironment) EnsureEventingReplicasReflected(t *testing.T, eventingCR *v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		hpa, err := env.GetHPAFromK8s(eventing.GetPublisherDeploymentName(*eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to get Eventing resource", "error", err,
				"name", eventingCR.Name, "namespace", eventingCR.Namespace)
		}
		return *hpa.Spec.MinReplicas == int32(eventingCR.Spec.Publisher.Replicas.Min) && hpa.Spec.MaxReplicas == int32(eventingCR.Spec.Publisher.Replicas.Max)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure Eventing spec replicas is reflected")
}

func (env TestEnvironment) EnsureDeploymentOwnerReferenceSet(t *testing.T, eventingCR *v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		deployment, err := env.GetDeploymentFromK8s(eventing.GetPublisherDeploymentName(*eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to get Eventing resource", "error", err,
				"name", eventingCR.Name, "namespace", eventingCR.Namespace)
		}
		return evnttestutils.HasOwnerReference(deployment, *eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure Eventing owner reference is set")
}

func (env TestEnvironment) EnsureEPPPublishServiceOwnerReferenceSet(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(eventing.GetPublisherPublishServiceName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.HasOwnerReference(result, eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure PublishService owner reference is set")
}

func (env TestEnvironment) EnsureEPPMetricsServiceOwnerReferenceSet(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(eventing.GetPublisherMetricsServiceName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.HasOwnerReference(result, eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure MetricsService owner reference is set")
}

func (env TestEnvironment) EnsureEPPHealthServiceOwnerReferenceSet(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(eventing.GetPublisherHealthServiceName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.HasOwnerReference(result, eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure HealthService owner reference is set")
}

func (env TestEnvironment) EnsureEPPServiceAccountOwnerReferenceSet(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceAccountFromK8s(eventing.GetPublisherServiceAccountName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.HasOwnerReference(result, eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure ServiceAccount owner reference is set")
}

func (env TestEnvironment) EnsureEPPClusterRoleOwnerReferenceSet(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetClusterRoleFromK8s(eventing.GetPublisherClusterRoleName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.HasOwnerReference(result, eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure ClusterRole owner reference is set")
}

func (env TestEnvironment) EnsureEPPClusterRoleBindingOwnerReferenceSet(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetClusterRoleBindingFromK8s(eventing.GetPublisherClusterRoleBindingName(eventingCR),
			eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.HasOwnerReference(result, eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure ClusterRoleBinding owner reference is set")
}

func (env TestEnvironment) EnsureEPPPublishServiceCorrect(t *testing.T, eppDeployment *v1.Deployment,
	eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(eventing.GetPublisherPublishServiceName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.IsEPPPublishServiceCorrect(*result, *eppDeployment)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure PublishService correctness.")
}

func (env TestEnvironment) EnsureEPPMetricsServiceCorrect(t *testing.T, eppDeployment *v1.Deployment,
	eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(eventing.GetPublisherMetricsServiceName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.IsEPPMetricsServiceCorrect(*result, *eppDeployment)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure MetricsService correctness.")
}

func (env TestEnvironment) EnsureEPPHealthServiceCorrect(t *testing.T, eppDeployment *v1.Deployment,
	eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(eventing.GetPublisherHealthServiceName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.IsEPPHealthServiceCorrect(*result, *eppDeployment)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure HealthService correctness.")
}

func (env TestEnvironment) EnsureEPPClusterRoleCorrect(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetClusterRoleFromK8s(eventing.GetPublisherClusterRoleName(eventingCR), eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.IsEPPClusterRoleCorrect(*result)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure ClusterRole correctness")
}

func (env TestEnvironment) EnsureEPPClusterRoleBindingCorrect(t *testing.T, eventingCR v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		result, err := env.GetClusterRoleBindingFromK8s(eventing.GetPublisherClusterRoleBindingName(eventingCR),
			eventingCR.Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}
		return evnttestutils.IsEPPClusterRoleBindingCorrect(*result, eventingCR)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure ClusterRoleBinding correctness")
}

func (env TestEnvironment) EnsureCABundleInjectedIntoWebhooks(t *testing.T) {
	require.Eventually(t, func() bool {
		// get cert secret from k8s.
		certSecret, err := env.GetSecretFromK8s(getTestBackendConfig().WebhookSecretName,
			getTestBackendConfig().Namespace)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}

		// get Mutating and validating webhook configurations from k8s.
		mwh, err := env.KubeClient.GetMutatingWebHookConfiguration(env.Context,
			getTestBackendConfig().MutatingWebhookName)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}

		vwh, err := env.KubeClient.GetValidatingWebHookConfiguration(env.Context,
			getTestBackendConfig().ValidatingWebhookName)
		if err != nil {
			env.Logger.WithContext().Error(err)
			return false
		}

		if len(mwh.Webhooks) == 0 || len(vwh.Webhooks) == 0 {
			env.Logger.WithContext().Error("Invalid mutating and validating webhook configurations")
			return false
		}

		if !bytes.Equal(mwh.Webhooks[0].ClientConfig.CABundle, certSecret.Data[eventingctrl.TLSCertField]) {
			env.Logger.WithContext().Error("CABundle of mutating configuration is not correct")
			return false
		}

		if !bytes.Equal(vwh.Webhooks[0].ClientConfig.CABundle, certSecret.Data[eventingctrl.TLSCertField]) {
			env.Logger.WithContext().Error("CABundle of validating configuration is not correct")
			return false
		}
		return true
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure correctness of CABundle in Webhooks")
}

func (env TestEnvironment) EnsureEventMeshSecretCreated(t *testing.T, eventing *v1alpha1.Eventing) {
	subarr := strings.Split(eventing.Spec.Backend.Config.EventMeshSecret, "/")
	secret := evnttestutils.NewEventMeshSecret(subarr[1], subarr[0])
	env.EnsureK8sResourceCreated(t, secret)
}

func (env TestEnvironment) EnsureEventMeshSecretDeleted(t *testing.T, eventing *v1alpha1.Eventing) {
	subarr := strings.Split(eventing.Spec.Backend.Config.EventMeshSecret, "/")
	secret := evnttestutils.NewEventMeshSecret(subarr[1], subarr[0])
	env.EnsureK8sResourceDeleted(t, secret)
}

func (env TestEnvironment) EnsureOAuthSecretCreated(t *testing.T, eventing *v1alpha1.Eventing) {
	secret := evnttestutils.NewOAuthSecret("eventing-webhook-auth", eventing.Namespace)
	env.EnsureK8sResourceCreated(t, secret)
}

func (env TestEnvironment) DeleteServiceFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteServiceAccountFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteClusterRoleFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteClusterRoleBindingFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteHPAFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) UpdateEventingStatus(eventing *v1alpha1.Eventing) error {
	return env.k8sClient.Status().Update(env.Context, eventing)
}

func (env TestEnvironment) UpdateNATSStatus(nats *natsv1alpha1.NATS) error {
	baseNats := &natsv1alpha1.NATS{}
	if err := env.k8sClient.Get(env.Context,
		types.NamespacedName{
			Namespace: nats.Namespace,
			Name:      nats.Name,
		}, baseNats); err != nil {
		return err
	}
	baseNats.Status = nats.Status
	return env.k8sClient.Status().Update(env.Context, baseNats)
}

func (env TestEnvironment) makeNatsCrReady(t *testing.T, nats *natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		nats.Status.State = natsv1alpha1.StateReady

		err := env.UpdateNATSStatus(nats)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to update NATS CR status", err)
			return false
		}
		return true
	}, BigTimeOut, BigPollingInterval, "failed to update status of NATS CR")
}

func (env TestEnvironment) makeNatsCrError(t *testing.T, nats *natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		nats.Status.State = natsv1alpha1.StateError

		err := env.UpdateNATSStatus(nats)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to update NATS CR status", err)
			return false
		}
		return true
	}, BigTimeOut, BigPollingInterval, "failed to update status of NATS CR")
}

func (env TestEnvironment) GetNATSFromK8s(name, namespace string) (*natsv1alpha1.NATS, error) {
	var nats *natsv1alpha1.NATS
	err := env.k8sClient.Get(env.Context, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, nats)
	return nats, err
}

func newSecretWithTLSSecret(dummyCABundle []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getTestBackendConfig().WebhookSecretName,
			Namespace: getTestBackendConfig().Namespace,
		},
		Data: map[string][]byte{
			eventingctrl.TLSCertField: dummyCABundle,
		},
	}
}

func getMutatingWebhookConfig(webhook []admissionv1.MutatingWebhook) *admissionv1.MutatingWebhookConfiguration {
	return &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: getTestBackendConfig().MutatingWebhookName,
		},
		Webhooks: webhook,
	}
}

func getValidatingWebhookConfig(webhook []admissionv1.ValidatingWebhook) *admissionv1.ValidatingWebhookConfiguration {
	return &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: getTestBackendConfig().ValidatingWebhookName,
		},
		Webhooks: webhook,
	}
}

func getTestBackendConfig() env.BackendConfig {
	return env.BackendConfig{
		WebhookSecretName:     "eventing-manager-webhook-server-cert",
		MutatingWebhookName:   "subscription-mutating-webhook-configuration",
		ValidatingWebhookName: "subscription-validating-webhook-configuration",
		Namespace:             "kyma-system",
	}
}

func (env TestEnvironment) GetEventingFromK8s(name, namespace string) (*eventingv1alpha1.Eventing, error) {
	eventing := &eventingv1alpha1.Eventing{}
	err := env.k8sClient.Get(env.Context, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, eventing)
	return eventing, err
}

func (env TestEnvironment) DeleteEventingFromK8s(name, namespace string) error {
	cr := &eventingv1alpha1.Eventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return env.k8sClient.Delete(env.Context, cr)
}

func (env TestEnvironment) DeleteSecretFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) GetDeploymentFromK8s(name, namespace string) (*v1.Deployment, error) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &v1.Deployment{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetServiceFromK8s(name, namespace string) (*corev1.Service, error) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.Service{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetSecretFromK8s(name, namespace string) (*corev1.Secret, error) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	return env.KubeClient.GetSecret(env.Context, nn.String())
}

func (env TestEnvironment) GetServiceAccountFromK8s(name, namespace string) (*corev1.ServiceAccount, error) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.ServiceAccount{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetClusterRoleFromK8s(name, namespace string) (*rbacv1.ClusterRole, error) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &rbacv1.ClusterRole{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetClusterRoleBindingFromK8s(name, namespace string) (*rbacv1.ClusterRoleBinding, error) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &rbacv1.ClusterRoleBinding{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetHPAFromK8s(name, namespace string) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &autoscalingv1.HorizontalPodAutoscaler{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) CreateUnstructuredK8sResource(obj *unstructured.Unstructured) error {
	return env.k8sClient.Create(env.Context, obj)
}

func (env TestEnvironment) EnsureK8sUnStructResourceCreated(t *testing.T, obj *unstructured.Unstructured) {
	require.NoError(t, env.k8sClient.Create(env.Context, obj))
}
