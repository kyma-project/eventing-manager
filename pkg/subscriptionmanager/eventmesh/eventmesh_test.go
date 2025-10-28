package eventmesh

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	backendeventmesh "github.com/kyma-project/eventing-manager/pkg/backend/eventmesh"
	"github.com/kyma-project/eventing-manager/pkg/backend/utils"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/client"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	submgrmanager "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

type bebSubMgrMock struct {
	Client           dynamic.Interface
	eventMeshBackend backendeventmesh.Backend
}

func (c *bebSubMgrMock) Init(_ manager.Manager) error {
	return nil
}

func (c *bebSubMgrMock) Start(_ env.DefaultSubscriptionConfig, _ submgrmanager.Params) error {
	return nil
}

func (c *bebSubMgrMock) Stop(_ bool) error {
	return nil
}

func Test_cleanupEventMesh(t *testing.T) {
	// given
	bebSubMgr := bebSubMgrMock{}
	ctx := context.Background()

	// create a Kyma subscription
	subscription := eventingtesting.NewSubscription("test", "test",
		eventingtesting.WithWebhookAuthForEventMesh(),
		eventingtesting.WithFakeSubscriptionStatus(),
		eventingtesting.WithOrderCreatedFilter(),
	)
	subscription.Spec.Sink = "https://bla.test.svc.cluster.local"

	// create an APIRule
	h := apigatewayv2.Host("host-test")
	hosts := []*apigatewayv2.Host{&h}
	apiRule := eventingtesting.NewAPIRule(subscription,
		eventingtesting.WithPath(),
		eventingtesting.WithService("svc-test", hosts),
	)
	subscription.Status.Backend.APIRuleName = apiRule.Name

	// start BEB Mock
	bebMock := startBEBMock()
	envConf := env.Config{
		BEBAPIURL:                bebMock.MessagingURL,
		ClientID:                 "client-id",
		ClientSecret:             "client-secret",
		TokenEndpoint:            bebMock.TokenURL,
		WebhookActivationTimeout: 0,
		EventTypePrefix:          eventingtesting.EventTypePrefix,
		BEBNamespace:             "/default/ns",
		Qos:                      string(types.QosAtLeastOnce),
	}
	credentials := &backendeventmesh.OAuth2ClientCredentials{
		ClientID:     "webhook_client_id",
		ClientSecret: "webhook_client_secret",
	}

	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	// create a EventMesh handler to connect to BEB Mock
	nameMapper := utils.NewBEBSubscriptionNameMapper("mydomain.com",
		backendeventmesh.MaxSubscriptionNameLength)
	eventMeshHandler := backendeventmesh.NewEventMesh(credentials, nameMapper, defaultLogger)
	err = eventMeshHandler.Initialize(envConf)
	require.NoError(t, err)
	bebSubMgr.eventMeshBackend = eventMeshHandler

	// create fake Dynamic clients
	fakeClient, err := eventingtesting.NewFakeSubscriptionClient(subscription)
	require.NoError(t, err)
	bebSubMgr.Client = fakeClient

	// Create APIRule
	unstructuredAPIRule, err := eventingtesting.ToUnstructuredAPIRule(apiRule)
	require.NoError(t, err)
	unstructuredAPIRuleBeforeCleanup, err := bebSubMgr.Client.Resource(utils.APIRuleGroupVersionResource()).Namespace(
		"test").Create(ctx, unstructuredAPIRule, kmetav1.CreateOptions{})
	require.NoError(t, err)
	require.NotNil(t, unstructuredAPIRuleBeforeCleanup)

	// create an EventMesh subscription from Kyma subscription
	eventMeshCleaner := cleaner.NewEventMeshCleaner(defaultLogger)
	_, err = bebSubMgr.eventMeshBackend.SyncSubscription(subscription, eventMeshCleaner, apiRule)
	require.NoError(t, err)

	// check that the subscription exist in bebMock
	getSubscriptionURL := fmt.Sprintf(client.GetURLFormat, nameMapper.MapSubscriptionName(subscription.Name,
		subscription.Namespace))
	getSubscriptionURL = bebMock.MessagingURL + getSubscriptionURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getSubscriptionURL, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// check that the Kyma subscription exists
	unstructuredSub, err := bebSubMgr.Client.Resource(eventingtesting.SubscriptionGroupVersionResource()).Namespace(
		"test").Get(ctx, subscription.Name, kmetav1.GetOptions{})
	require.NoError(t, err)
	_, err = eventingtesting.ToSubscription(unstructuredSub)
	require.NoError(t, err)

	// check that the APIRule exists
	unstructuredAPIRuleBeforeCleanup, err = bebSubMgr.Client.Resource(utils.APIRuleGroupVersionResource()).Namespace(
		"test").Get(ctx, apiRule.Name, kmetav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, unstructuredAPIRuleBeforeCleanup)

	// when
	err = cleanupEventMesh(bebSubMgr.eventMeshBackend, bebSubMgr.Client, defaultLogger.WithContext())
	require.NoError(t, err)

	// then
	// the BEB subscription should be deleted from BEB Mock
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, getSubscriptionURL, nil)
	require.NoError(t, err)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// the Kyma subscription status should be empty
	unstructuredSub, err = bebSubMgr.Client.Resource(eventingtesting.SubscriptionGroupVersionResource()).Namespace(
		"test").Get(ctx, subscription.Name, kmetav1.GetOptions{})
	require.NoError(t, err)
	gotSub, err := eventingtesting.ToSubscription(unstructuredSub)
	require.NoError(t, err)
	expectedSubStatus := eventingv1alpha2.SubscriptionStatus{Types: []eventingv1alpha2.EventType{}}
	require.Equal(t, expectedSubStatus, gotSub.Status)

	// the associated APIRule should be deleted
	unstructuredAPIRuleAfterCleanup, err := bebSubMgr.Client.Resource(utils.APIRuleGroupVersionResource()).Namespace(
		"test").Get(ctx, apiRule.Name, kmetav1.GetOptions{})
	require.Error(t, err)
	require.Nil(t, unstructuredAPIRuleAfterCleanup)
	bebMock.Stop()
}

func Test_markAllV1Alpha2SubscriptionsAsNotReady(t *testing.T) {
	// given
	ctx := context.Background()

	// create a Kyma subscription
	subscription := eventingtesting.NewSubscription("test", "test",
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedFilter(),
		eventingtesting.WithStatus(true),
	)

	// set hashes
	const (
		ev2Hash            = int64(6118518533334734626)
		eventMeshHash      = int64(6748405436686967274)
		webhookAuthHash    = int64(6118518533334734627)
		eventMeshLocalHash = int64(6883494500014499539)
	)
	subscription.Status.Backend.Ev2hash = ev2Hash
	subscription.Status.Backend.EventMeshHash = eventMeshHash
	subscription.Status.Backend.WebhookAuthHash = webhookAuthHash
	subscription.Status.Backend.EventMeshLocalHash = eventMeshLocalHash

	// create logger
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	// create fake k8s dynamic client
	fakeClient, err := eventingtesting.NewFakeSubscriptionClient(subscription)
	require.NoError(t, err)

	// verify that the subscription status is ready
	unstructuredSub, err := fakeClient.Resource(eventingtesting.SubscriptionGroupVersionResource()).Namespace(
		"test").Get(ctx, subscription.Name, kmetav1.GetOptions{})
	require.NoError(t, err)
	gotSub, err := eventingtesting.ToSubscription(unstructuredSub)
	require.NoError(t, err)
	require.True(t, gotSub.Status.Ready)

	// when
	err = markAllV1Alpha2SubscriptionsAsNotReady(fakeClient, defaultLogger.WithContext())
	require.NoError(t, err)

	// then
	unstructuredSub, err = fakeClient.Resource(eventingtesting.SubscriptionGroupVersionResource()).Namespace(
		"test").Get(ctx, subscription.Name, kmetav1.GetOptions{})
	require.NoError(t, err)
	gotSub, err = eventingtesting.ToSubscription(unstructuredSub)
	require.NoError(t, err)
	require.False(t, gotSub.Status.Ready)

	// ensure hashes are preserved
	require.Equal(t, ev2Hash, gotSub.Status.Backend.Ev2hash)
	require.Equal(t, eventMeshHash, gotSub.Status.Backend.EventMeshHash)
	require.Equal(t, webhookAuthHash, gotSub.Status.Backend.WebhookAuthHash)
	require.Equal(t, eventMeshLocalHash, gotSub.Status.Backend.EventMeshLocalHash)
}

func startBEBMock() *eventingtesting.EventMeshMock {
	b := eventingtesting.NewEventMeshMock()
	b.Start()
	return b
}
