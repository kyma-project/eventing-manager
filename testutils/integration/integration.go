package integration

import (
	"context"
	"github.com/avast/retry-go/v3"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/controller/eventing"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/kyma-project/eventing-manager/testutils"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"log"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"testing"
	"time"
)

const (
	useExistingCluster       = false
	attachControlPlaneOutput = false
	testEnvStartDelay        = time.Minute
	testEnvStartAttempts     = 10
	BigTimeOut               = 40 * time.Second
	SmallPollingInterval     = 1 * time.Second
)

// TestEnvironment provides mocked resources for integration tests.
type TestEnvironment struct {
	Context          context.Context
	EnvTestInstance  *envtest.Environment
	k8sClient        client.Client
	K8sDynamicClient *dynamic.DynamicClient
	Reconciler       *eventing.Reconciler
	Logger           *zap.SugaredLogger
	Recorder         *record.EventRecorder
	TestCancelFn     context.CancelFunc
}

//nolint:funlen // Used in testing
func NewTestEnvironment(projectRootDir string, celValidationEnabled bool) (*TestEnvironment, error) {
	var err error
	// setup context
	ctx := context.Background()

	// setup logger
	sugaredLogger, err := testutils.NewSugaredLogger()
	if err != nil {
		return nil, err
	}

	testEnv, envTestKubeCfg, err := StartEnvTest(projectRootDir, celValidationEnabled)
	if err != nil {
		return nil, err
	}

	// add to Scheme
	err = eventingv1alpha1.AddToScheme(scheme.Scheme)
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
	recorder := ctrlMgr.GetEventRecorderFor("eventing-manager")

	// setup reconciler
	eventingReconciler := eventingcontroller.NewReconciler(
		ctrlMgr.GetClient(),
		ctrlMgr.GetScheme(),
		sugaredLogger,
		recorder,
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
		Context:          ctx,
		k8sClient:        k8sClient,
		K8sDynamicClient: dynamicClient,
		Reconciler:       eventingReconciler,
		Logger:           sugaredLogger,
		Recorder:         &recorder,
		EnvTestInstance:  testEnv,
		TestCancelFn:     cancelCtx,
	}, nil
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

func StartEnvTest(projectRootDir string, celValidationEnabled bool) (*envtest.Environment, *rest.Config, error) {
	// Reference: https://book.kubebuilder.io/reference/envtest.html
	useExistingCluster := useExistingCluster
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(projectRootDir, "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: attachControlPlaneOutput,
		UseExistingCluster:       &useExistingCluster,
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

// GetEventingAssert fetches an Eventing from k8s and allows making assertions on it.
func (env TestEnvironment) GetEventingAssert(g *gomega.GomegaWithT,
	eventing *eventingv1alpha1.Eventing) gomega.AsyncAssertion {
	return g.Eventually(func() *eventingv1alpha1.Eventing {
		gotEventing, err := env.GetEventingFromK8s(eventing.Name, eventing.Namespace)
		if err != nil {
			log.Printf("fetch subscription %s/%s failed: %v", eventing.Name, eventing.Namespace, err)
			return nil
		}
		return &gotEventing
	}, BigTimeOut, SmallPollingInterval)
}

func (env TestEnvironment) GetEventingFromK8s(name, namespace string) (eventingv1alpha1.Eventing, error) {
	var eventing eventingv1alpha1.Eventing
	err := env.k8sClient.Get(env.Context, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &eventing)
	return eventing, err
}

func (env TestEnvironment) EnsureNamespaceCreation(t *testing.T, namespace string) {
	if namespace == "default" {
		return
	}
	// create namespace
	ns := testutils.NewNamespace(namespace)
	require.NoError(t, client.IgnoreAlreadyExists(env.k8sClient.Create(env.Context, ns)))
}

func (env TestEnvironment) CreateUnstructuredK8sResource(obj *unstructured.Unstructured) error {
	return env.k8sClient.Create(env.Context, obj)
}

func (env TestEnvironment) EnsureK8sUnStructResourceCreated(t *testing.T, obj *unstructured.Unstructured) {
	require.NoError(t, env.k8sClient.Create(env.Context, obj))
}
