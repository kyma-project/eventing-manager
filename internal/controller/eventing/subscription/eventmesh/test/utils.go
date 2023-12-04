//nolint:gosec //this is just a test, and security issues found here will not result in code used in a prod environment
package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	subscriptioncontrollereventmesh "github.com/kyma-project/eventing-manager/internal/controller/eventing/subscription/eventmesh"

	"github.com/avast/retry-go/v3"
	"github.com/go-logr/zapr"
	apigateway "github.com/kyma-incubator/api-gateway/api/v1beta1"
	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"

	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	backendeventmesh "github.com/kyma-project/eventing-manager/pkg/backend/eventmesh"
	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	"github.com/kyma-project/eventing-manager/pkg/backend/sink"
	backendutils "github.com/kyma-project/eventing-manager/pkg/backend/utils"
	"github.com/kyma-project/eventing-manager/pkg/constants"
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
	smallTimeOut             = 5 * time.Second
	smallPollingInterval     = 1 * time.Second
	namespacePrefixLength    = 5
	syncPeriodSeconds        = 2
	maxReconnects            = 10
	eventMeshMockKeyPrefix   = "/messaging/events/subscriptions"
	certsURL                 = "https://domain.com/oauth2/certs"
)

//nolint:gochecknoglobals // only used in tests
var (
	emTestEnsemble    *eventMeshTestEnsemble
	k8sCancelFn       context.CancelFunc
	acceptableMethods = []string{http.MethodPost, http.MethodOptions}
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
	ctrllog.SetLogger(zapr.NewLogger(defaultLogger.WithContext().Desugar()))

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

	if err = apigateway.AddToScheme(scheme.Scheme); err != nil {
		return err
	}
	// +kubebuilder:scaffold:scheme

	// start eventMesh manager instance
	syncPeriod := syncPeriodSeconds * time.Second
	webhookInstallOptions := &emTestEnsemble.testEnv.WebhookInstallOptions
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Cache:                  cache.Options{SyncPeriod: &syncPeriod},
		HealthProbeBindAddress: "0", // disable
		Scheme:                 scheme.Scheme,
		Metrics:                server.Options{BindAddress: "0"}, // disable
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookInstallOptions.LocalServingPort,
			Host:    webhookInstallOptions.LocalServingHost,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
	})
	if err != nil {
		return err
	}

	// setup nameMapper for EventMesh
	emTestEnsemble.nameMapper = backendutils.NewBEBSubscriptionNameMapper(testutils.Domain,
		backendeventmesh.MaxSubscriptionNameLength)

	// setup eventMesh reconciler
	recorder := k8sManager.GetEventRecorderFor("eventing-controller")
	sinkValidator := sink.NewValidator(context.Background(), k8sManager.GetClient(), recorder)
	credentials := &backendeventmesh.OAuth2ClientCredentials{
		ClientID:     "foo-client-id",
		ClientSecret: "foo-client-secret",
		TokenURL:     "foo-token-url",
		CertsURL:     certsURL,
	}
	emTestEnsemble.envConfig = getEnvConfig()
	col := metrics.NewCollector()
	testReconciler := subscriptioncontrollereventmesh.NewReconciler(
		context.Background(),
		k8sManager.GetClient(),
		defaultLogger,
		recorder,
		getEnvConfig(),
		cleaner.NewEventMeshCleaner(defaultLogger),
		backendeventmesh.NewEventMesh(credentials, emTestEnsemble.nameMapper, defaultLogger),
		credentials,
		emTestEnsemble.nameMapper,
		sinkValidator,
		col,
		testutils.Domain,
	)

	if err = testReconciler.SetupUnmanaged(k8sManager); err != nil {
		return err
	}

	// start k8s client
	go func() {
		var ctx context.Context
		ctx, k8sCancelFn = context.WithCancel(ctrl.SetupSignalHandler())
		err = k8sManager.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	emTestEnsemble.k8sClient = k8sManager.GetClient()

	return startAndWaitForWebhookServer(k8sManager, webhookInstallOptions)
}

func startAndWaitForWebhookServer(k8sManager manager.Manager, webhookInstallOpts *envtest.WebhookInstallOptions) error {
	if err := (&eventingv1alpha2.Subscription{}).SetupWebhookWithManager(k8sManager); err != nil {
		return err
	}
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOpts.LocalServingHost, webhookInstallOpts.LocalServingPort)
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
	useExistingCluster := useExistingCluster
	emTestEnsemble.testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("../../../../../../", "config", "crd", "bases"),
			filepath.Join("../../../../../../", "config", "crd", "for-tests"),
		},
		AttachControlPlaneOutput: attachControlPlaneOutput,
		UseExistingCluster:       &useExistingCluster,
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

func GenerateInvalidSubscriptionError(subName, errType string, path *field.Path) error {
	webhookErr := "admission webhook \"vsubscription.kb.io\" denied the request: "
	givenError := kerrors.NewInvalid(
		eventingv1alpha2.GroupKind, subName,
		field.ErrorList{eventingv1alpha2.MakeInvalidFieldError(path, subName, errType)})
	givenError.ErrStatus.Message = webhookErr + givenError.ErrStatus.Message
	return givenError
}

func getTestNamespace() string {
	return fmt.Sprintf("ns-%s", utils.GetRandString(namespacePrefixLength))
}

func ensureNamespaceCreated(ctx context.Context, t *testing.T, namespace string) {
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
	require.NoError(t, emTestEnsemble.k8sClient.Create(ctx, obj))
}

func ensureK8sResourceNotCreated(ctx context.Context, t *testing.T, obj client.Object, err error) {
	require.Equal(t, emTestEnsemble.k8sClient.Create(ctx, obj), err)
}

func ensureK8sResourceDeleted(ctx context.Context, t *testing.T, obj client.Object) {
	require.NoError(t, emTestEnsemble.k8sClient.Delete(ctx, obj))
}

func ensureK8sSubscriptionUpdated(ctx context.Context, t *testing.T, subscription *eventingv1alpha2.Subscription) {
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
func ensureAPIRuleStatusUpdatedWithStatusReady(ctx context.Context, t *testing.T, apiRule *apigateway.APIRule) {
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

// ensureAPIRuleNotFound ensures that a APIRule does not exists (or deleted).
func ensureAPIRuleNotFound(ctx context.Context, t *testing.T, apiRule *apigateway.APIRule) {
	require.Eventually(t, func() bool {
		apiRuleKey := client.ObjectKey{
			Namespace: apiRule.Namespace,
			Name:      apiRule.Name,
		}

		apiRule2 := new(apigateway.APIRule)
		err := emTestEnsemble.k8sClient.Get(ctx, apiRuleKey, apiRule2)
		return kerrors.IsNotFound(err)
	}, bigTimeOut, bigPollingInterval)
}

func getAPIRulesList(ctx context.Context, svc *kcorev1.Service) (*apigateway.APIRuleList, error) {
	labels := map[string]string{
		constants.ControllerServiceLabelKey:  svc.Name,
		constants.ControllerIdentityLabelKey: constants.ControllerIdentityLabelValue,
	}
	apiRules := &apigateway.APIRuleList{}
	err := emTestEnsemble.k8sClient.List(ctx, apiRules, &client.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels),
		Namespace:     svc.Namespace,
	})
	return apiRules, err
}

func getAPIRule(ctx context.Context, apiRule *apigateway.APIRule) (*apigateway.APIRule, error) {
	lookUpKey := types.NamespacedName{
		Namespace: apiRule.Namespace,
		Name:      apiRule.Name,
	}
	err := emTestEnsemble.k8sClient.Get(ctx, lookUpKey, apiRule)
	return apiRule, err
}

func filterAPIRulesForASvc(apiRules *apigateway.APIRuleList, svc *kcorev1.Service) apigateway.APIRule {
	if len(apiRules.Items) == 1 && *apiRules.Items[0].Spec.Service.Name == svc.Name {
		return apiRules.Items[0]
	}
	return apigateway.APIRule{}
}

// countEventMeshRequests returns how many requests for a given subscription are sent for each HTTP method
//
//nolint:gocognit
func countEventMeshRequests(subscriptionName, eventType string) (int, int, int) {
	countGet, countPost, countDelete := 0, 0, 0
	emTestEnsemble.eventMeshMock.Requests.ReadEach(
		func(request *http.Request, payload interface{}) {
			switch method := request.Method; method {
			case http.MethodGet:
				if strings.Contains(request.URL.Path, subscriptionName) {
					countGet++
				}
			case http.MethodPost:
				if sub, ok := payload.(emstypes.Subscription); ok {
					if len(sub.Events) > 0 {
						for _, event := range sub.Events {
							if event.Type == eventType && sub.Name == subscriptionName {
								countPost++
							}
						}
					}
				}
			case http.MethodDelete:
				if strings.Contains(request.URL.Path, subscriptionName) {
					countDelete++
				}
			}
		})
	return countGet, countPost, countDelete
}

func getEventMeshSubFromMock(subscriptionName, subscriptionNamespace string) *emstypes.Subscription {
	key := getEventMeshSubKeyForMock(subscriptionName, subscriptionNamespace)
	return emTestEnsemble.eventMeshMock.Subscriptions.GetSubscription(key)
}

func getEventMeshSubKeyForMock(subscriptionName, subscriptionNamespace string) string {
	nm1 := emTestEnsemble.nameMapper.MapSubscriptionName(subscriptionName, subscriptionNamespace)
	return fmt.Sprintf("%s/%s", eventMeshMockKeyPrefix, nm1)
}

func getEventMeshKeyForMock(name string) string {
	return fmt.Sprintf("%s/%s", eventMeshMockKeyPrefix, name)
}

// ensureK8sEventReceived checks if a certain event have triggered for the given namespace.
func ensureK8sEventReceived(t *testing.T, event kcorev1.Event, namespace string) {
	ctx := context.TODO()
	require.Eventually(t, func() bool {
		// get all events from k8s for namespace
		eventList := &kcorev1.EventList{}
		err := emTestEnsemble.k8sClient.List(ctx, eventList, client.InNamespace(namespace))
		require.NoError(t, err)

		// find the desired event
		var receivedEvent *kcorev1.Event
		for i, e := range eventList.Items {
			if e.Reason == event.Reason {
				receivedEvent = &eventList.Items[i]
				break
			}
		}

		// check the received event
		require.NotNil(t, receivedEvent)
		require.Equal(t, receivedEvent.Reason, event.Reason)
		require.Equal(t, receivedEvent.Message, event.Message)
		require.Equal(t, receivedEvent.Type, event.Type)
		return true
	}, bigTimeOut, bigPollingInterval)
}
