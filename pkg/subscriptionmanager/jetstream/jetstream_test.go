package jetstream

import (
	"fmt"
	"testing"
	"time"

	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	"github.com/kyma-project/eventing-manager/pkg/backend/jetstream"
	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

const (
	subscriptionName      = "test"
	subscriptionNamespace = "test"
)

func TestCleanup(t *testing.T) {
	// given
	testEnv := setUpTestEnvironment(t)
	defer eventingtesting.ShutDownNATSServer(testEnv.natsServer)
	defer testEnv.subscriber.Shutdown()
	// a consumer should exist on JetStream
	testEnv.consumersEquals(t, 1)

	// when
	err := cleanupv2(testEnv.jsBackend, testEnv.dynamicClient, testEnv.defaultLogger.WithContext())

	// then
	require.NoError(t, err)
	// the consumer on JetStream should have being deleted
	testEnv.consumersEquals(t, 0)
}

// utilities and helper functions

type TestEnvironment struct {
	dynamicClient dynamic.Interface
	jsBackend     *jetstream.JetStream
	jsCtx         nats.JetStreamContext
	natsServer    *server.Server
	subscriber    *eventingtesting.Subscriber
	envConf       env.NATSConfig
	defaultLogger *logger.Logger
}

func getJetStreamClient(t *testing.T, natsURL string) nats.JetStreamContext {
	t.Helper()
	conn, err := nats.Connect(natsURL)
	require.NoError(t, err)
	jsClient, err := conn.JetStream()
	require.NoError(t, err)
	return jsClient
}

func getNATSConf(natsURL string, natsPort int) env.NATSConfig {
	streamName := fmt.Sprintf("%s%d", eventingtesting.JSStreamName, natsPort)
	return env.NATSConfig{
		URL:                     natsURL,
		MaxReconnects:           10,
		ReconnectWait:           time.Second,
		EventTypePrefix:         eventingtesting.EventTypePrefix,
		JSStreamName:            streamName,
		JSSubjectPrefix:         streamName,
		JSStreamStorageType:     jetstream.StorageTypeMemory,
		JSStreamRetentionPolicy: jetstream.RetentionPolicyInterest,
		JSStreamDiscardPolicy:   jetstream.DiscardPolicyNew,
	}
}

func createAndSyncSubscription(t *testing.T, sinkURL string, jsBackend *jetstream.JetStream) *eventingv1alpha2.Subscription {
	t.Helper()
	// create test subscription
	testSub := eventingtesting.NewSubscription(
		subscriptionName, subscriptionNamespace,
		eventingtesting.WithSource(eventingtesting.EventSourceClean),
		eventingtesting.WithSinkURL(sinkURL),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithStatusTypes([]eventingv1alpha2.EventType{
			{
				OriginalType: eventingtesting.OrderCreatedV1Event,
				CleanType:    eventingtesting.OrderCreatedV1Event,
			},
		}),
	)

	// create NATS subscription
	err := jsBackend.SyncSubscription(testSub)
	require.NoError(t, err)
	return testSub
}

func (te *TestEnvironment) consumersEquals(t *testing.T, length int) {
	t.Helper()
	// verify that the number of consumers is one
	info, err := te.jsCtx.StreamInfo(te.envConf.JSStreamName)
	require.NoError(t, err)
	require.Equal(t, length, info.State.Consumers)
}

func setUpTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()
	// create a test subscriber and natsServer
	subscriber := eventingtesting.NewSubscriber()
	// create NATS Server with JetStream enabled
	natsPort, err := eventingtesting.GetFreePort()
	require.NoError(t, err)
	natsServer := eventingtesting.StartDefaultJetStreamServer(natsPort)

	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	envConf := getNATSConf(natsServer.ClientURL(), natsPort)
	jsClient := getJetStreamClient(t, natsServer.ClientURL())

	// init the metrics collector
	metricsCollector := metrics.NewCollector()

	jsCleaner := cleaner.NewJetStreamCleaner(defaultLogger)
	defaultSubsConfig := env.DefaultSubscriptionConfig{MaxInFlightMessages: 9}

	// Create an instance of the JetStream Backend
	jsBackend := jetstream.NewJetStream(envConf, metricsCollector, jsCleaner, defaultSubsConfig, defaultLogger)

	// Initialize JetStream Backend
	err = jsBackend.Initialize(nil)
	require.NoError(t, err)

	testSub := createAndSyncSubscription(t, subscriber.SinkURL, jsBackend)
	// create fake Dynamic clients
	fakeClient, err := eventingtesting.NewFakeSubscriptionClient(testSub)
	require.NoError(t, err)

	return &TestEnvironment{
		dynamicClient: fakeClient,
		jsBackend:     jsBackend,
		jsCtx:         jsClient,
		natsServer:    natsServer,
		subscriber:    subscriber,
		envConf:       envConf,
		defaultLogger: defaultLogger,
	}
}
