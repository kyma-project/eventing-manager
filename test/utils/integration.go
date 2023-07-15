package utils

import (
	"context"
	"crypto/rand"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/avast/retry-go/v3"
	"github.com/go-logr/zapr"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventing "github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	ecdeployment "github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	useExistingCluster       = false
	attachControlPlaneOutput = false
	testEnvStartDelay        = time.Minute
	testEnvStartAttempts     = 10
	namespacePrefixLength    = 5
	TwoMinTimeOut            = 120 * time.Second
	BigPollingInterval       = 3 * time.Second
	BigTimeOut               = 30 * time.Second
	SmallTimeOut             = 5 * time.Second
	SmallPollingInterval     = 1 * time.Second
	EventTypePrefix          = "prefix"
	JSStreamName             = "kyma"
)

// TestEnvironment provides mocked resources for integration tests.
type TestEnvironment struct {
	Context         context.Context
	EnvTestInstance *envtest.Environment
	k8sClient       client.Client
	KubeClient      *k8s.Client
	Reconciler      *eventing.Reconciler
	Logger          *logger.Logger
	Recorder        *record.EventRecorder
	TestCancelFn    context.CancelFunc
}

//nolint:funlen // Used in testing
func NewTestEnvironment(projectRootDir string, celValidationEnabled bool,
	allowedNATSCR *natsv1alpha1.NATS) (*TestEnvironment, error) {
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

	// add NATS CRD scheme
	err = natsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	// +kubebuilder:scaffold:scheme

	k8sClient, err := client.New(envTestKubeCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
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

	kubeClient := k8s.NewKubeClient(ctrlMgr.GetClient())

	// create NATS manager instance

	// setup reconciler
	natsConfig := env.NATSConfig{
		EventTypePrefix: EventTypePrefix,
		JSStreamName:    JSStreamName,
	}
	os.Setenv("WEBHOOK_TOKEN_ENDPOINT", "https://oauth2.ev-manager.kymatunas.shoot.canary.k8s-hana.ondemand.com/oauth2/token")
	os.Setenv("DOMAIN", "my.test.domain")
	os.Setenv("EVENT_TYPE_PREFIX", EventTypePrefix)
	eventingReconciler := eventing.NewReconciler(
		ctx,
		natsConfig,
		k8sClient,
		ctrlMgr.GetScheme(),
		ctrLogger,
		ctrlMgr.GetEventRecorderFor("eventing-manager-test"),
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

	return &TestEnvironment{
		Context:         ctx,
		k8sClient:       k8sClient,
		KubeClient:      &kubeClient,
		Reconciler:      eventingReconciler,
		Logger:          ctrLogger,
		Recorder:        &recorder,
		EnvTestInstance: testEnv,
		TestCancelFn:    cancelCtx,
	}, nil
}

func StartEnvTest(projectRootDir string, celValidationEnabled bool) (*envtest.Environment, *rest.Config, error) {
	// Reference: https://book.kubebuilder.io/reference/envtest.html
	useExistingCluster := useExistingCluster

	dummyCABundle := make([]byte, 20)
	if _, err := rand.Read(dummyCABundle); err != nil {
		return nil, nil, err
	}

	newCABundle := make([]byte, 20)
	if _, err := rand.Read(newCABundle); err != nil {
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
	mwh.Name = "subscription-mutating-webhook-configuration"

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
	vwh.Name = "subscription-validating-webhook-configuration"

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

	// retry to stop the api-server
	sleepTime := 1 * time.Second
	var err error
	const retries = 20
	for i := 0; i < retries; i++ {
		if err = env.EnvTestInstance.Stop(); err == nil {
			break
		}
		time.Sleep(sleepTime)
	}
	return err
}

// GetEventingAssert fetches a Eventing from k8s and allows making assertions on it.
func (env TestEnvironment) GetEventingAssert(g *gomega.GomegaWithT,
	eventing *eventingv1alpha1.Eventing) gomega.AsyncAssertion {
	return g.Eventually(func() *eventingv1alpha1.Eventing {
		gotEventing, err := env.GetEventingFromK8s(eventing.Name, eventing.Namespace)
		if err != nil {
			log.Printf("fetch eventing %s/%s failed: %v", eventing.Name, eventing.Namespace, err)
			return nil
		}
		return gotEventing
	}, BigTimeOut, SmallPollingInterval)
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

func (env TestEnvironment) EnsureDeploymentExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetDeploymentFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of Deployment")
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

func (env TestEnvironment) EnsureNATSResourceStateReady(t *testing.T, nats *natsv1alpha1.NATS) {
	env.makeNatsCrReady(t, nats)
	require.Eventually(t, func() bool {
		err := env.k8sClient.Get(env.Context, types.NamespacedName{Name: nats.Name, Namespace: nats.Namespace}, nats)
		return err == nil && nats.Status.State == natsv1alpha1.StateReady
	}, BigTimeOut, BigPollingInterval, "failed to ensure NATS CR is stored")
}

func (env TestEnvironment) EnsureEventingSpecPublisherReflected(t *testing.T, eventing *v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		deployment, err := env.GetDeploymentFromK8s(ecdeployment.PublisherName, eventing.Namespace)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to get Eventing resource", "error", err,
				"name", eventing.Name, "namespace", eventing.Namespace)
		}
		return eventing.Spec.Publisher.Resources.Limits.Cpu().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu()) &&
			eventing.Spec.Publisher.Resources.Limits.Memory().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory()) &&
			eventing.Spec.Publisher.Resources.Requests.Cpu().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu()) &&
			eventing.Spec.Publisher.Resources.Requests.Memory().Equal(*deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Memory())
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure Eventing spec publisher is reflected")
}

func (env TestEnvironment) EnsureEventingReplicasReflected(t *testing.T, eventing *v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		hpa, err := env.GetHPAFromK8s(ecdeployment.PublisherName, eventing.Namespace)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to get Eventing resource", "error", err,
				"name", eventing.Name, "namespace", eventing.Namespace)
		}
		return *hpa.Spec.MinReplicas == int32(eventing.Spec.Publisher.Replicas.Min) && hpa.Spec.MaxReplicas == int32(eventing.Spec.Publisher.Replicas.Max)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure Eventing spec replicas is reflected")
}

func (env TestEnvironment) EnsureDeploymentOwnerReferenceSet(t *testing.T, eventing *v1alpha1.Eventing) {
	require.Eventually(t, func() bool {
		deployment, err := env.GetDeploymentFromK8s(ecdeployment.PublisherName, eventing.Namespace)
		if err != nil {
			env.Logger.WithContext().Errorw("failed to get Eventing resource", "error", err,
				"name", eventing.Name, "namespace", eventing.Namespace)
		}
		return len(deployment.OwnerReferences) > 0 && deployment.OwnerReferences[0].Name == eventing.Name &&
			deployment.OwnerReferences[0].Kind == "Eventing" &&
			deployment.OwnerReferences[0].UID == eventing.UID
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure Eventing owner reference is set")
}

func (env TestEnvironment) UpdateEventingStatus(eventing *v1alpha1.Eventing) error {
	return env.k8sClient.Status().Update(env.Context, eventing)
}

func (env TestEnvironment) UpdateNATSStatus(nats *natsv1alpha1.NATS) error {
	return env.k8sClient.Status().Update(env.Context, nats)
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

func (env TestEnvironment) GetNATSFromK8s(name, namespace string) (*natsv1alpha1.NATS, error) {
	var nats *natsv1alpha1.NATS
	err := env.k8sClient.Get(env.Context, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, nats)
	return nats, err
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
		WebhookSecretName:     "webhookSecret",
		MutatingWebhookName:   "mutatingWH",
		ValidatingWebhookName: "validatingWH",
	}
}

func (env TestEnvironment) GetEventingFromK8s(name, namespace string) (*eventingv1alpha1.Eventing, error) {
	eventing := &eventingv1alpha1.Eventing{}
	err := env.k8sClient.Get(env.Context, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, eventing)
	if err != nil {
		return nil, err
	}
	return eventing, err
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
