package eventmesh

import (
	"fmt"
	"net/http"
	"testing"

	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/stretchr/testify/require"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	backendutils "github.com/kyma-project/eventing-manager/pkg/backend/utils"
	emsclientmocks "github.com/kyma-project/eventing-manager/pkg/ems/api/events/client/mocks"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

func Test_getProcessedEventTypes(t *testing.T) {
	// given
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	nameMapper := backendutils.NewBEBSubscriptionNameMapper("mydomain.com",
		MaxSubscriptionNameLength)

	// cases
	testCases := []struct {
		name                    string
		givenSubscription       *eventingv1alpha2.Subscription
		givenEventTypePrefix    string
		wantProcessedEventTypes []backendutils.EventTypeInfo
		wantError               bool
	}{
		{
			name: "success if the given subscription has already cleaned source and event types",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Types: []string{
						"order.created.v1",
					},
					Source: "test",
				},
			},
			givenEventTypePrefix: eventingtesting.EventMeshPrefix,
			wantProcessedEventTypes: []backendutils.EventTypeInfo{
				{
					OriginalType:  "order.created.v1",
					CleanType:     "order.created.v1",
					ProcessedType: fmt.Sprintf("%s.test.order.created.v1", eventingtesting.EventMeshPrefix),
				},
			},
			wantError: false,
		},
		{
			name: "success if the given subscription has uncleaned source and event types",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Types: []string{
						"Segment1.Segment2.Segment3.Segment4-Part1-Part2-Ä.Segment5-Part1-Part2-Ä.v1",
					},
					Source: "test-Ä.Segment2",
				},
			},
			givenEventTypePrefix: eventingtesting.EventMeshPrefix,
			wantProcessedEventTypes: []backendutils.EventTypeInfo{
				{
					OriginalType: "Segment1.Segment2.Segment3.Segment4-Part1-Part2-Ä.Segment5-Part1-Part2-Ä.v1",
					CleanType:    "Segment1Segment2Segment3Segment4Part1Part2.Segment5Part1Part2.v1",
					ProcessedType: fmt.Sprintf("%s.testSegment2.Segment1Segment2Segment3Segment4Part1Part2"+
						".Segment5Part1Part2.v1", eventingtesting.EventMeshPrefix),
				},
			},
			wantError: false,
		},
		{
			name: "success if the given subscription has duplicate event types",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Types: []string{
						"order.created.v1",
						"order.created.v1",
					},
					Source: "test",
				},
			},
			givenEventTypePrefix: eventingtesting.EventMeshPrefix,
			wantProcessedEventTypes: []backendutils.EventTypeInfo{
				{
					OriginalType:  "order.created.v1",
					CleanType:     "order.created.v1",
					ProcessedType: fmt.Sprintf("%s.test.order.created.v1", eventingtesting.EventMeshPrefix),
				},
			},
			wantError: false,
		},
		{
			name: "should not clean or process event type if the given subscription has matchingType=EXACT",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Types: []string{
						"test1.test2.test3.order.created.v1",
					},
					Source:       "test",
					TypeMatching: eventingv1alpha2.TypeMatchingExact,
				},
			},
			givenEventTypePrefix: eventingtesting.EventMeshPrefix,
			wantProcessedEventTypes: []backendutils.EventTypeInfo{
				{
					OriginalType:  "test1.test2.test3.order.created.v1",
					CleanType:     "test1.test2.test3.order.created.v1",
					ProcessedType: "test1.test2.test3.order.created.v1",
				},
			},
			wantError: false,
		},
		{
			name: "should fail if the given subscription types and EventMeshPrefix " +
				"exceeds the EventMesh segments limit",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Types: []string{
						"order.created.v1",
					},
					Source: "test",
				},
			},
			givenEventTypePrefix:    eventingtesting.InvalidEventMeshPrefix,
			wantProcessedEventTypes: nil,
			wantError:               true,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			eventMesh := NewEventMesh(&OAuth2ClientCredentials{}, nameMapper, defaultLogger)
			emCleaner := cleaner.NewEventMeshCleaner(defaultLogger)
			err = eventMesh.Initialize(env.Config{EventTypePrefix: testcase.givenEventTypePrefix})
			require.NoError(t, err)

			// when
			eventTypeInfos, err := eventMesh.getProcessedEventTypes(testcase.givenSubscription, emCleaner)

			// then
			require.Equal(t, testcase.wantError, err != nil)
			if !testcase.wantError {
				require.Equal(t, testcase.wantProcessedEventTypes, eventTypeInfos)
			}
		})
	}
}

//nolint:dupl // no duplicate as this tests the kyma side and not the eventmesh side
func Test_handleKymaSubModified(t *testing.T) {
	// given
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	nameMapper := backendutils.NewBEBSubscriptionNameMapper("mydomain.com",
		MaxSubscriptionNameLength)

	// cases
	testCases := []struct {
		name                      string
		givenKymaSub              *eventingv1alpha2.Subscription
		givenEventMeshSub         *types.Subscription
		givenClientDeleteResponse *types.DeleteResponse
		wantIsModified            bool
		wantError                 bool
	}{
		{
			name: "should not delete EventMesh sub if Kyma sub was not modified",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Backend: eventingv1alpha2.Backend{
						EventMeshLocalHash: int64(-9219276050977208880),
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentMode",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientDeleteResponse: &types.DeleteResponse{
				Response: types.Response{
					StatusCode: http.StatusNoContent,
					Message:    "",
				},
			},
			wantIsModified: false,
			wantError:      false,
		},
		{
			name: "should delete EventMesh sub if Kyma sub was modified",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Backend: eventingv1alpha2.Backend{
						Ev2hash: int64(-9219276050977208880),
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentModeModified",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientDeleteResponse: &types.DeleteResponse{
				Response: types.Response{
					StatusCode: http.StatusNoContent,
					Message:    "",
				},
			},
			wantIsModified: true,
			wantError:      false,
		},
		{
			name: "fail if delete EventMesh sub return error",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Backend: eventingv1alpha2.Backend{
						Ev2hash: int64(-9219276050977208880),
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentModeModified",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientDeleteResponse: &types.DeleteResponse{
				Response: types.Response{
					StatusCode: http.StatusInternalServerError,
					Message:    "",
				},
			},
			wantIsModified: true,
			wantError:      true,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			eventMesh := NewEventMesh(&OAuth2ClientCredentials{}, nameMapper, defaultLogger)
			// Set a mock client interface for EventMesh
			mockClient := new(emsclientmocks.PublisherManager)
			mockClient.On("Delete", testcase.givenEventMeshSub.Name).Return(testcase.givenClientDeleteResponse, nil)
			eventMesh.client = mockClient

			// when
			isModified, err := eventMesh.handleKymaSubModified(testcase.givenEventMeshSub, testcase.givenKymaSub)

			// then
			require.Equal(t, testcase.wantError, err != nil)
			if !testcase.wantError {
				require.Equal(t, testcase.wantIsModified, isModified)
			}
		})
	}
}

//nolint:dupl // no duplicate as this tests the eventmesh side and not the kyma side
func Test_handleEventMeshSubModified(t *testing.T) {
	// given
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	nameMapper := backendutils.NewBEBSubscriptionNameMapper("mydomain.com",
		MaxSubscriptionNameLength)

	// cases
	testCases := []struct {
		name                      string
		givenKymaSub              *eventingv1alpha2.Subscription
		givenEventMeshSub         *types.Subscription
		givenClientDeleteResponse *types.DeleteResponse
		wantIsModified            bool
		wantError                 bool
	}{
		{
			name: "should not delete EventMesh sub if EventMesh sub was not modified",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Backend: eventingv1alpha2.Backend{
						EventMeshHash: int64(-9219276050977208880),
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentMode",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientDeleteResponse: &types.DeleteResponse{
				Response: types.Response{
					StatusCode: http.StatusNoContent,
					Message:    "",
				},
			},
			wantIsModified: false,
			wantError:      false,
		},
		{
			name: "should delete EventMesh sub if EventMesh sub was modified",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Backend: eventingv1alpha2.Backend{
						EventMeshHash: int64(-9219276050977208880),
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentModeModified",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientDeleteResponse: &types.DeleteResponse{
				Response: types.Response{
					StatusCode: http.StatusNoContent,
					Message:    "",
				},
			},
			wantIsModified: true,
			wantError:      false,
		},
		{
			name: "should fail if delete EventMesh sub return error",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Backend: eventingv1alpha2.Backend{
						EventMeshHash: int64(-9219276050977208880),
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentModeModified",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientDeleteResponse: &types.DeleteResponse{
				Response: types.Response{
					StatusCode: http.StatusInternalServerError,
					Message:    "",
				},
			},
			wantIsModified: true,
			wantError:      true,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			eventMesh := NewEventMesh(&OAuth2ClientCredentials{}, nameMapper, defaultLogger)
			// Set a mock client interface for EventMesh
			mockClient := new(emsclientmocks.PublisherManager)
			mockClient.On("Delete", testcase.givenEventMeshSub.Name).Return(testcase.givenClientDeleteResponse, nil)
			eventMesh.client = mockClient

			// when
			isModified, err := eventMesh.handleEventMeshSubModified(testcase.givenEventMeshSub, testcase.givenKymaSub)

			// then
			require.Equal(t, testcase.wantError, err != nil)
			if !testcase.wantError {
				require.Equal(t, testcase.wantIsModified, isModified)
			}
		})
	}
}

func Test_handleCreateEventMeshSub(t *testing.T) {
	// given
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	nameMapper := backendutils.NewBEBSubscriptionNameMapper("mydomain.com",
		MaxSubscriptionNameLength)

	// cases
	testCases := []struct {
		name                      string
		givenKymaSub              *eventingv1alpha2.Subscription
		givenEventMeshSub         *types.Subscription
		givenClientCreateResponse *types.CreateResponse
		wantError                 bool
	}{
		{
			name: "should be able create EventMesh sub",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Types: []eventingv1alpha2.EventType{
						{
							OriginalType: "test1",
							CleanType:    "test1",
						},
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentMode",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientCreateResponse: &types.CreateResponse{
				Response: types.Response{
					StatusCode: http.StatusAccepted,
					Message:    "",
				},
			},
			wantError: false,
		},
		{
			name: "should fail to create EventMesh sub if server returns error",
			givenKymaSub: &eventingv1alpha2.Subscription{
				Status: eventingv1alpha2.SubscriptionStatus{
					Types: []eventingv1alpha2.EventType{
						{
							OriginalType: "test1",
							CleanType:    "test1",
						},
					},
				},
			},
			givenEventMeshSub: &types.Subscription{
				Name:            "Name1",
				ContentMode:     "ContentModeModified",
				ExemptHandshake: true,
				Qos:             types.QosAtLeastOnce,
				WebhookURL:      "www.kyma-project.io",
			},
			givenClientCreateResponse: &types.CreateResponse{
				Response: types.Response{
					StatusCode: http.StatusInternalServerError,
					Message:    "",
				},
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			eventMesh := NewEventMesh(&OAuth2ClientCredentials{}, nameMapper, defaultLogger)
			// Set a mock client interface for EventMesh
			mockClient := new(emsclientmocks.PublisherManager)
			mockClient.On("Create", testcase.givenEventMeshSub).Return(testcase.givenClientCreateResponse, nil)
			mockClient.On("Get", testcase.givenEventMeshSub.Name).Return(testcase.givenEventMeshSub, &types.Response{
				StatusCode: http.StatusOK,
				Message:    "",
			}, nil)
			eventMesh.client = mockClient

			// when
			_, err := eventMesh.handleCreateEventMeshSub(testcase.givenEventMeshSub, testcase.givenKymaSub)

			// then
			require.Equal(t, testcase.wantError, err != nil)
			if !testcase.wantError {
				require.Empty(t, testcase.givenKymaSub.Status.Types)
			}
		})
	}
}

func Test_handleKymaSubStatusUpdate(t *testing.T) {
	// given
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	// cases
	testCases := []struct {
		name               string
		givenKymaSub       *eventingv1alpha2.Subscription
		givenEventMeshSub  *types.Subscription
		givenTypeInfos     []backendutils.EventTypeInfo
		wantEventTypes     []eventingv1alpha2.EventType
		wantEventMeshTypes []eventingv1alpha2.EventMeshTypes
	}{
		{
			name:         "should be able create EventMesh sub",
			givenKymaSub: &eventingv1alpha2.Subscription{},
			givenEventMeshSub: &types.Subscription{
				Name:                     "Name1",
				ContentMode:              "ContentMode",
				ExemptHandshake:          true,
				Qos:                      types.QosAtLeastOnce,
				WebhookURL:               "www.kyma-project.io",
				SubscriptionStatusReason: "test-reason",
			},
			givenTypeInfos: []backendutils.EventTypeInfo{
				{
					OriginalType:  "test1",
					CleanType:     "test2",
					ProcessedType: "test3",
				},
				{
					OriginalType:  "order1",
					CleanType:     "order2",
					ProcessedType: "order3",
				},
			},
			wantEventTypes: []eventingv1alpha2.EventType{
				{
					OriginalType: "test1",
					CleanType:    "test2",
				},
				{
					OriginalType: "order1",
					CleanType:    "order2",
				},
			},
			wantEventMeshTypes: []eventingv1alpha2.EventMeshTypes{
				{
					OriginalType:  "test1",
					EventMeshType: "test3",
				},
				{
					OriginalType:  "order1",
					EventMeshType: "order3",
				},
			},
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			eventMesh := NewEventMesh(nil, nil, defaultLogger)

			// when
			isChanged, _ := eventMesh.handleKymaSubStatusUpdate(testcase.givenEventMeshSub, testcase.givenEventMeshSub, testcase.givenKymaSub, testcase.givenTypeInfos)

			// then
			require.True(t, isChanged)
			require.Equal(t, testcase.wantEventTypes, testcase.givenKymaSub.Status.Types)
			require.Equal(t, testcase.wantEventMeshTypes, testcase.givenKymaSub.Status.Backend.EmsTypes)
			require.Equal(t, testcase.givenEventMeshSub.SubscriptionStatusReason, testcase.givenKymaSub.Status.Backend.EventMeshSubscriptionStatus.StatusReason)
		})
	}
}

func Test_SyncSubscription(t *testing.T) {
	credentials := &OAuth2ClientCredentials{
		ClientID:     "foo-client-id",
		ClientSecret: "foo-client-secret",
	}
	// given
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	nameMapper := backendutils.NewBEBSubscriptionNameMapper("mydomain.com",
		MaxSubscriptionNameLength)
	eventMesh := NewEventMesh(credentials, nameMapper, defaultLogger)

	// start BEB Mock
	eventMeshMock := startEventMeshMock()
	envConf := env.Config{
		BEBAPIURL:                eventMeshMock.MessagingURL,
		ClientID:                 "client-id",
		ClientSecret:             "client-secret",
		TokenEndpoint:            eventMeshMock.TokenURL,
		WebhookActivationTimeout: 0,
		EventTypePrefix:          eventingtesting.EventTypePrefix,
		BEBNamespace:             "/default/ns",
		Qos:                      string(types.QosAtLeastOnce),
	}

	err = eventMesh.Initialize(envConf)
	require.NoError(t, err)

	subscription := fixtureValidSubscription("some-name", "some-namespace")
	subscription.Status.Backend.EventMeshHash = 0
	subscription.Status.Backend.Ev2hash = 0

	apiRule := eventingtesting.NewAPIRule(subscription,
		eventingtesting.WithPath(),
		eventingtesting.WithService("foo-svc", "foo-host"),
	)

	// cases - reconcile same subscription multiple times
	testCases := []struct {
		name           string
		givenEventType string
		wantIsChanged  bool
	}{
		{
			name:           "should be able to sync first time",
			givenEventType: eventingtesting.OrderCreatedEventTypeNotClean,
			wantIsChanged:  true,
		},
		{
			name:           "should be able to sync second time with same subscription",
			givenEventType: eventingtesting.OrderCreatedEventTypeNotClean,
			wantIsChanged:  false,
		},
		{
			name:           "should be able to sync third time with modified subscription",
			givenEventType: eventingtesting.OrderCreatedV2Event,
			wantIsChanged:  true,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			subscription.Spec.Types[0] = testcase.givenEventType
			changed, err := eventMesh.SyncSubscription(subscription, cleaner.NewEventMeshCleaner(defaultLogger), apiRule)
			require.NoError(t, err)
			require.Equal(t, testcase.wantIsChanged, changed)
		})
	}

	// cleanup
	eventMeshMock.Stop()
}

func Test_handleWebhookAuthChange(t *testing.T) {
	// setup
	mock := startEventMeshMock()
	defer func() { mock.Stop() }()

	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	require.NoError(t, err)

	credentials := &OAuth2ClientCredentials{ClientID: "client-id", ClientSecret: "client-secret"}
	mapper := backendutils.NewBEBSubscriptionNameMapper("domain.com", MaxSubscriptionNameLength)
	config := env.Config{BEBAPIURL: mock.MessagingURL, TokenEndpoint: mock.TokenURL}

	eventMesh := NewEventMesh(credentials, mapper, defaultLogger)
	err = eventMesh.Initialize(config)
	require.NoError(t, err)

	tests := []struct {
		name            string
		givenSameHash   bool // same hash means that the WebhookAuth config did not change
		wantDeleteCount int  // count requests to delete EventMesh Subscription
		wantPatchCount  int  // count requests to update the WebhookAuth config
		wantPutCount    int  // count requests to pause and resume EventMesh Subscription
	}{
		{
			name:            "WebhookAuth config changed",
			givenSameHash:   false,
			wantDeleteCount: 0,
			wantPatchCount:  1,
			wantPutCount:    2, // 1 request for pausing and 1 request for resuming the EventMesh Subscription
		},
		{
			name:            "WebhookAuth config did not change",
			givenSameHash:   true,
			wantDeleteCount: 0,
			wantPatchCount:  0,
			wantPutCount:    0,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mock.Reset()

			// given
			kymaSub := fixtureValidSubscription("test-subscription", "test-namespace")
			typesInfo, err := eventMesh.getProcessedEventTypes(kymaSub, cleaner.NewEventMeshCleaner(defaultLogger))
			require.NoError(t, err)
			require.NotNil(t, typesInfo)
			apiRule := eventingtesting.NewAPIRule(
				kymaSub,
				eventingtesting.WithPath(),
				eventingtesting.WithService("test-service", "http://localhost"),
			)

			// convert Kyma Subscription to EventMesh Subscription
			emSub, err := backendutils.ConvertKymaSubToEventMeshSub(
				kymaSub,
				typesInfo,
				apiRule,
				eventMesh.webhookAuth,
				eventMesh.protocolSettings,
				eventMesh.namespace,
				eventMesh.SubNameMapper,
			)
			require.NoError(t, err)
			require.NotNil(t, emSub)

			if test.givenSameHash {
				hash, hashErr := backendutils.GetWebhookAuthHash(emSub.WebhookAuth)
				require.NoError(t, hashErr)
				require.NotEqual(t, 0, hash)
				kymaSub.Status.Backend.WebhookAuthHash = hash // simulate equal hashes
			}

			// ensure EventMesh subscription is created on the mock server
			emSub, err = eventMesh.handleCreateEventMeshSub(emSub, kymaSub)
			require.NoError(t, err)
			require.NotNil(t, emSub)

			// when
			err = eventMesh.handleWebhookAuthChange(emSub, kymaSub)
			require.NoError(t, err)

			emSubName := mapper.MapSubscriptionName(kymaSub.Name, kymaSub.Namespace)
			deleteURI := fmt.Sprintf("/messaging/events/subscriptions/%s", emSubName)
			patchURI := fmt.Sprintf("/messaging/events/subscriptions/%s", emSubName)
			putURI := fmt.Sprintf("/messaging/events/subscriptions/%s/state", emSubName)

			// then
			require.Equal(t, test.wantDeleteCount, mock.CountRequests(http.MethodDelete, deleteURI))
			require.Equal(t, test.wantPatchCount, mock.CountRequests(http.MethodPatch, patchURI))
			require.Equal(t, test.wantPutCount, mock.CountRequests(http.MethodPut, putURI))
		})
	}
}

// fixtureValidSubscription returns a valid subscription.
func fixtureValidSubscription(name, namespace string) *eventingv1alpha2.Subscription {
	return eventingtesting.NewSubscription(
		name, namespace,
		eventingtesting.WithSinkURL("https://webhook.xxx.com"),
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedEventTypeNotClean),
		eventingtesting.WithWebhookAuthForEventMesh(),
	)
}

func startEventMeshMock() *eventingtesting.EventMeshMock {
	eventMesh := eventingtesting.NewEventMeshMock()
	eventMesh.Start()
	return eventMesh
}
