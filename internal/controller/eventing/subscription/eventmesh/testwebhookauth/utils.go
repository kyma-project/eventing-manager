//nolint:gosec //this is just a test, and security issues found here will not result in code used in a prod environment
package testwebhookauth

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/avast/retry-go/v3"
	"github.com/go-logr/zapr"
	apigatewayv1beta1 "github.com/kyma-project/api-gateway/apis/gateway/v1beta1"
	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	kctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	kctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	subscriptioncontrollereventmesh "github.com/kyma-project/eventing-manager/internal/controller/eventing/subscription/eventmesh"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	backendeventmesh "github.com/kyma-project/eventing-manager/pkg/backend/eventmesh"
	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	"github.com/kyma-project/eventing-manager/pkg/backend/sink"
	backendutils "github.com/kyma-project/eventing-manager/pkg/backend/utils"
	emstypes "github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/featureflags"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	"github.com/kyma-project/eventing-manager/pkg/utils"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

type eventMeshTestEnsemble struct {
	k8sClient     client.Client
	testEnv       *envtest.Environment
	eventMeshMock *eventingtesting.EventMeshMock
	nameMapper    backendutils.NameMapper
	envConfig     env.Config
}

const (
	useExistingCluster       = false
	attachControlPlaneOutput = false
	testEnvStartDelay        = time.Minute
	testEnvStartAttempts     = 10
	twoMinTimeOut            = 120 * time.Second
	bigPollingInterval       = 3 * time.Second
	bigTimeOut               = 40 * time.Second
	namespacePrefixLength    = 5
	syncPeriodSeconds        = 2
	maxReconnects            = 10
	eventMeshMockKeyPrefix   = "/messaging/events/subscriptions"
	tokenURL                 = "https://domain.com/oauth2/token"
	certsURL                 = "https://domain.com/oauth2/certs"
)

//nolint:gochecknoglobals // only used in tests
var (
	emTestEnsemble   *eventMeshTestEnsemble
	k8sCancelFn      context.CancelFunc
	eventMeshBackend *backendeventmesh.EventMesh
	testReconciler   *subscriptioncontrollereventmesh.Reconciler
	credentials      = &backendeventmesh.OAuth2ClientCredentials{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     tokenURL,
		CertsURL:     certsURL,
	}
)

func setupSuite() error {
	featureflags.SetEventingWebhookAuthEnabled(true)
	emTestEnsemble = &eventMeshTestEnsemble{}

	// define logger
	var err error
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	if err != nil {
		return err
	}
	kctrllog.SetLogger(zapr.NewLogger(defaultLogger.WithContext().Desugar()))

	// setup test Env
	cfg, err := startTestEnv()
	if err != nil || cfg == nil {
		return err
	}

	// start event mesh mock
	emTestEnsemble.eventMeshMock = startNewEventMeshMock()

	// add schemes
	if err = eventingv1alpha2.AddToScheme(scheme.Scheme); err != nil {
		return err
	}

	if err = apigatewayv1beta1.AddToScheme(scheme.Scheme); err != nil {
		return err
	}
	// +kubebuilder:scaffold:scheme

	// start eventMesh manager instance
	k8sManager, webhookInstallOptions, err := setupManager(cfg)
	if err != nil {
		return err
	}

	// setup nameMapper for EventMesh
	emTestEnsemble.nameMapper = backendutils.NewBEBSubscriptionNameMapper(testutils.Domain,
		backendeventmesh.MaxSubscriptionNameLength)

	// setup eventMesh reconciler
	recorder := k8sManager.GetEventRecorderFor("eventing-controller")
	sinkValidator := sink.NewValidator(k8sManager.GetClient(), recorder)
	emTestEnsemble.envConfig = getEnvConfig()
	eventMeshBackend = backendeventmesh.NewEventMesh(credentials, emTestEnsemble.nameMapper, defaultLogger)
	col := metrics.NewCollector()
	testReconciler = subscriptioncontrollereventmesh.NewReconciler(
		k8sManager.GetClient(),
		defaultLogger,
		recorder,
		getEnvConfig(),
		cleaner.NewEventMeshCleaner(defaultLogger),
		eventMeshBackend,
		credentials,
		emTestEnsemble.nameMapper,
		sinkValidator,
		col,
		testutils.Domain,
	)

	if err = testReconciler.SetupUnmanaged(context.Background(), k8sManager); err != nil {
		return err
	}

	// start k8s client
	go func() {
		var ctx context.Context
		ctx, k8sCancelFn = context.WithCancel(kctrl.SetupSignalHandler())
		err = k8sManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	emTestEnsemble.k8sClient = k8sManager.GetClient()

	return startAndWaitForWebhookServer(k8sManager, webhookInstallOptions)
}

func setupManager(cfg *rest.Config) (manager.Manager, *envtest.WebhookInstallOptions, error) {
	syncPeriod := syncPeriodSeconds * time.Second
	webhookInstallOptions := &emTestEnsemble.testEnv.WebhookInstallOptions
	k8sManager, err := kctrl.NewManager(cfg, kctrl.Options{
		Scheme:                 scheme.Scheme,
		Cache:                  cache.Options{SyncPeriod: &syncPeriod},
		Metrics:                server.Options{BindAddress: "0"}, // disable
		HealthProbeBindAddress: "0",                              // disable
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
	})
	return k8sManager, webhookInstallOptions, err
}

func startAndWaitForWebhookServer(manager manager.Manager, installOpts *envtest.WebhookInstallOptions) error {
	if err := (&eventingv1alpha2.Subscription{}).SetupWebhookWithManager(manager); err != nil {
		return err
	}
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", installOpts.LocalServingHost, installOpts.LocalServingPort)
	// wait for the webhook server to get ready
	err := retry.Do(func() error {
		conn, connErr := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if connErr != nil {
			return connErr
		}
		return conn.Close()
	}, retry.Attempts(maxReconnects))
	return err
}

func startTestEnv() (*rest.Config, error) {
	emTestEnsemble.testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("../../../../../../", "config", "crd", "bases"),
			filepath.Join("../../../../../../", "config", "crd", "for-tests"),
		},
		AttachControlPlaneOutput: attachControlPlaneOutput,
		UseExistingCluster:       utils.BoolPtr(useExistingCluster),
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("../../../../../../", "config", "webhook")},
		},
	}

	var cfg *rest.Config
	err := retry.Do(func() error {
		defer func() {
			if r := recover(); r != nil {
				log.Println("panic recovered:", r)
			}
		}()

		cfgLocal, startErr := emTestEnsemble.testEnv.Start()
		cfg = cfgLocal
		return startErr
	},
		retry.Delay(testEnvStartDelay),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(testEnvStartAttempts),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("[%v] try failed to start testenv: %s", n, err)
			if stopErr := emTestEnsemble.testEnv.Stop(); stopErr != nil {
				log.Printf("failed to stop testenv: %s", stopErr)
			}
		}),
	)
	return cfg, err
}

func getEnvConfig() env.Config {
	return env.Config{
		BEBAPIURL:                emTestEnsemble.eventMeshMock.MessagingURL,
		ClientID:                 "foo-id",
		ClientSecret:             "foo-secret",
		TokenEndpoint:            emTestEnsemble.eventMeshMock.TokenURL,
		WebhookActivationTimeout: 0,
		EventTypePrefix:          eventingtesting.EventMeshPrefix,
		BEBNamespace:             eventingtesting.EventMeshNamespaceNS,
		Qos:                      string(emstypes.QosAtLeastOnce),
	}
}

func tearDownSuite() error {
	if k8sCancelFn != nil {
		k8sCancelFn()
	}
	err := emTestEnsemble.testEnv.Stop()
	emTestEnsemble.eventMeshMock.Stop()
	return err
}

func startNewEventMeshMock() *eventingtesting.EventMeshMock {
	emMock := eventingtesting.NewEventMeshMock()
	emMock.Start()
	return emMock
}

func getTestNamespace() string {
	return fmt.Sprintf("ns-%s", utils.GetRandString(namespacePrefixLength))
}

func ensureNamespaceCreated(ctx context.Context, t *testing.T, namespace string) {
	t.Helper()
	if namespace == "default" {
		return
	}
	// create namespace
	ns := fixtureNamespace(namespace)
	err := emTestEnsemble.k8sClient.Create(ctx, ns)
	if !kerrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}
}

func fixtureNamespace(name string) *kcorev1.Namespace {
	namespace := kcorev1.Namespace{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

func ensureK8sResourceCreated(ctx context.Context, t *testing.T, obj client.Object) {
	t.Helper()
	require.NoError(t, emTestEnsemble.k8sClient.Create(ctx, obj))
}

func ensureK8sSubscriptionUpdated(ctx context.Context, t *testing.T, subscription *eventingv1alpha2.Subscription) {
	t.Helper()
	require.Eventually(t, func() bool {
		latestSubscription := &eventingv1alpha2.Subscription{}
		lookupKey := types.NamespacedName{
			Namespace: subscription.Namespace,
			Name:      subscription.Name,
		}
		require.NoError(t, emTestEnsemble.k8sClient.Get(ctx, lookupKey, latestSubscription))
		require.NotEmpty(t, latestSubscription.Name)
		latestSubscription.Spec = subscription.Spec
		latestSubscription.Labels = subscription.Labels
		require.NoError(t, emTestEnsemble.k8sClient.Update(ctx, latestSubscription))
		return true
	}, bigTimeOut, bigPollingInterval)
}

// ensureAPIRuleStatusUpdatedWithStatusReady updates the status fof the APIRule (mocking APIGateway controller).
func ensureAPIRuleStatusUpdatedWithStatusReady(ctx context.Context, t *testing.T, apiRule *apigatewayv1beta1.APIRule) {
	t.Helper()
	require.Eventually(t, func() bool {
		fetchedAPIRule, err := getAPIRule(ctx, apiRule)
		if err != nil {
			return false
		}

		newAPIRule := fetchedAPIRule.DeepCopy()
		// mark the ApiRule status as ready
		eventingtesting.MarkReady(newAPIRule)

		// update ApiRule status on k8s
		err = emTestEnsemble.k8sClient.Status().Update(ctx, newAPIRule)
		return err == nil
	}, bigTimeOut, bigPollingInterval)
}

func getAPIRule(ctx context.Context, apiRule *apigatewayv1beta1.APIRule) (*apigatewayv1beta1.APIRule, error) {
	lookUpKey := types.NamespacedName{
		Namespace: apiRule.Namespace,
		Name:      apiRule.Name,
	}
	err := emTestEnsemble.k8sClient.Get(ctx, lookUpKey, apiRule)
	return apiRule, err
}

func getEventMeshSubFromMock(subscriptionName, subscriptionNamespace string) *emstypes.Subscription {
	key := getEventMeshSubKeyForMock(subscriptionName, subscriptionNamespace)
	return emTestEnsemble.eventMeshMock.Subscriptions.GetSubscription(key)
}

func getEventMeshSubKeyForMock(subscriptionName, subscriptionNamespace string) string {
	nm1 := emTestEnsemble.nameMapper.MapSubscriptionName(subscriptionName, subscriptionNamespace)
	return fmt.Sprintf("%s/%s", eventMeshMockKeyPrefix, nm1)
}

func setCredentials(credentials *backendeventmesh.OAuth2ClientCredentials) {
	eventMeshBackend.SetCredentials(credentials)
	testReconciler.SetCredentials(credentials)
}
