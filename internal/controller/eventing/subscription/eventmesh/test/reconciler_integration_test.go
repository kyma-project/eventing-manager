package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/assert"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	apigateway "github.com/kyma-incubator/api-gateway/api/v1beta1"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"

	eventMeshtypes "github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/object"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
	eventmeshsubmatchers "github.com/kyma-project/eventing-manager/testing/eventmeshsub"
)

const (
	invalidSinkErrMsg = "failed to validate subscription sink URL. " +
		"It is not a valid cluster local svc: Service \"invalid\" not found"
	testName = "test"
)

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	if err := setupSuite(); err != nil {
		panic(err)
	}

	// run tests
	code := m.Run()

	// tear down test env
	if err := tearDownSuite(); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func Test_ValidationWebhook(t *testing.T) {
	t.Parallel()
	var testCases = []struct {
		name                  string
		givenSubscriptionFunc func(namespace string) *eventingv1alpha2.Subscription
		wantError             error
	}{
		{
			name: "should fail to create subscription with invalid event source",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithStandardTypeMatching(),
					eventingtesting.WithSource(""),
					eventingtesting.WithOrderCreatedV1Event(),
					eventingtesting.WithValidSink(namespace, "svc"),
				)
			},
			wantError: GenerateInvalidSubscriptionError(testName,
				eventingv1alpha2.EmptyErrDetail, eventingv1alpha2.SourcePath),
		},
		{
			name: "should fail to create subscription with invalid event types",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithStandardTypeMatching(),
					eventingtesting.WithSource("source"),
					eventingtesting.WithTypes([]string{}),
					eventingtesting.WithValidSink(namespace, "svc"),
				)
			},
			wantError: GenerateInvalidSubscriptionError(testName,
				eventingv1alpha2.EmptyErrDetail, eventingv1alpha2.TypesPath),
		},
		{
			name: "should fail to create subscription with invalid sink",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithStandardTypeMatching(),
					eventingtesting.WithSource("source"),
					eventingtesting.WithOrderCreatedV1Event(),
					eventingtesting.WithSink("https://svc2.test.local"),
				)
			},
			wantError: GenerateInvalidSubscriptionError(testName,
				eventingv1alpha2.SuffixMissingErrDetail, eventingv1alpha2.SinkPath),
		},
		{
			name: "should fail to create subscription with invalid protocol settings",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithStandardTypeMatching(),
					eventingtesting.WithSource("source"),
					eventingtesting.WithOrderCreatedV1Event(),
					eventingtesting.WithInvalidWebhookAuthGrantType(),
					eventingtesting.WithValidSink(namespace, "svc"),
				)
			},
			wantError: GenerateInvalidSubscriptionError(testName,
				eventingv1alpha2.InvalidGrantTypeErrDetail, eventingv1alpha2.ConfigPath),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// create unique namespace for this test run
			testNamespace := getTestNamespace()
			ensureNamespaceCreated(ctx, t, testNamespace)

			// update namespace information in given test assets
			givenSubscription := tc.givenSubscriptionFunc(testNamespace)

			// attempt to create subscription
			ensureK8sResourceNotCreated(ctx, t, givenSubscription, tc.wantError)
		})
	}
}

func Test_CreateSubscription(t *testing.T) {
	var testCases = []struct {
		name                     string
		givenSubscriptionFunc    func(namespace string) *eventingv1alpha2.Subscription
		wantSubscriptionMatchers gomegatypes.GomegaMatcher
		wantEventMeshSubMatchers gomegatypes.GomegaMatcher
		wantEventMeshSubCheck    bool
		wantAPIRuleCheck         bool
		wantSubCreatedEventCheck bool
		wantSubActiveEventCheck  bool
	}{
		{
			name: "should fail to create subscription if sink does not exist",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, "invalid")),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionNotReady(),
				eventingtesting.HaveCondition(eventingv1alpha2.MakeCondition(
					eventingv1alpha2.ConditionAPIRuleStatus,
					eventingv1alpha2.ConditionReasonAPIRuleStatusNotReady,
					kcore.ConditionFalse, invalidSinkErrMsg)),
			),
		},
		{
			name: "should succeed to create subscription if types are non-empty",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionFinalizer(eventingv1alpha2.Finalizer),
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1Event),
					}, {
						OriginalType: fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
			wantEventMeshSubMatchers: gomega.And(
				eventmeshsubmatchers.HaveEvents(eventMeshtypes.Events{
					{
						Source: eventingtesting.EventMeshNamespaceNS,
						Type: fmt.Sprintf("%s.%s.%s0", eventingtesting.EventMeshPrefix,
							eventingtesting.ApplicationName, eventingtesting.OrderCreatedV1Event),
					},
					{
						Source: eventingtesting.EventMeshNamespaceNS,
						Type: fmt.Sprintf("%s.%s.%s1", eventingtesting.EventMeshPrefix,
							eventingtesting.ApplicationName, eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
			wantEventMeshSubCheck: true,
			wantAPIRuleCheck:      true,
		},
		{
			name: "should succeed to create subscription with empty protocol and webhook settings",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithNotCleanType(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionActiveCondition(),
			),
			wantEventMeshSubMatchers: gomega.And(
				// should have default values for protocol and webhook auth
				eventmeshsubmatchers.HaveContentMode(emTestEnsemble.envConfig.ContentMode),
				eventmeshsubmatchers.HaveExemptHandshake(emTestEnsemble.envConfig.ExemptHandshake),
				eventmeshsubmatchers.HaveQoS(eventMeshtypes.Qos(emTestEnsemble.envConfig.Qos)),
				eventmeshsubmatchers.HaveWebhookAuth(eventMeshtypes.WebhookAuth{
					ClientID:     "foo-client-id",
					ClientSecret: "foo-client-secret",
					TokenURL:     "foo-token-url",
					Type:         eventMeshtypes.AuthTypeClientCredentials,
					GrantType:    eventMeshtypes.GrantTypeClientCredentials,
				}),
			),
			wantAPIRuleCheck:      true,
			wantEventMeshSubCheck: true,
		},
		{
			name: "should succeed to create subscription with EXACT type matching",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithExactTypeMatching(),
					eventingtesting.WithEventMeshExactType(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionFinalizer(eventingv1alpha2.Finalizer),
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: eventingtesting.EventMeshExactType,
						CleanType:    eventingtesting.EventMeshExactType,
					},
				}),
			),
			wantEventMeshSubMatchers: gomega.And(
				eventmeshsubmatchers.HaveEvents(eventMeshtypes.Events{
					{
						Source: eventingtesting.EventMeshNamespaceNS,
						Type:   eventingtesting.EventMeshExactType,
					},
				}),
			),
			wantEventMeshSubCheck:    true,
			wantAPIRuleCheck:         true,
			wantSubCreatedEventCheck: true,
			wantSubActiveEventCheck:  true,
		},
		{
			name: "should mark a non-clean Subscription as ready",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithNotCleanSource(),
					eventingtesting.WithNotCleanType(),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionFinalizer(eventingv1alpha2.Finalizer),
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: eventingtesting.OrderCreatedV1EventNotClean,
						CleanType:    eventingtesting.OrderCreatedV1Event,
					},
				}),
			),
			wantEventMeshSubMatchers: gomega.And(
				eventmeshsubmatchers.HaveEvents(eventMeshtypes.Events{
					{
						Source: eventingtesting.EventMeshNamespaceNS,
						Type: fmt.Sprintf("%s.%s.%s", eventingtesting.EventMeshPrefix,
							eventingtesting.ApplicationName, eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
			wantEventMeshSubCheck: true,
			wantAPIRuleCheck:      true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)
			ctx := context.Background()

			// create unique namespace for this test run
			testNamespace := getTestNamespace()
			ensureNamespaceCreated(ctx, t, testNamespace)

			// update namespace information in given test assets
			givenSubscription := tc.givenSubscriptionFunc(testNamespace)

			// create a subscriber service
			subscriberSvc := eventingtesting.NewSubscriberSvc(givenSubscription.Name, testNamespace)
			ensureK8sResourceCreated(ctx, t, subscriberSvc)

			// create subscription
			ensureK8sResourceCreated(ctx, t, givenSubscription)

			// check if the subscription is as required
			getSubscriptionAssert(ctx, g, givenSubscription).Should(tc.wantSubscriptionMatchers)

			if tc.wantAPIRuleCheck {
				// check if an APIRule was created for the subscription
				getAPIRuleForASvcAssert(ctx, g, subscriberSvc).Should(eventingtesting.HaveNotEmptyAPIRule())
			}

			if tc.wantEventMeshSubCheck {
				emSub := getEventMeshSubFromMock(givenSubscription.Name, givenSubscription.Namespace)
				g.Expect(emSub).ShouldNot(gomega.BeNil())
				g.Expect(emSub).Should(tc.wantEventMeshSubMatchers)
			}

			if tc.wantSubCreatedEventCheck {
				message := eventingv1alpha2.CreateMessageForConditionReasonSubscriptionCreated(
					emTestEnsemble.nameMapper.MapSubscriptionName(givenSubscription.Name, givenSubscription.Namespace))
				subscriptionCreatedEvent := kcore.Event{
					Reason:  string(eventingv1alpha2.ConditionReasonSubscriptionCreated),
					Message: message,
					Type:    kcore.EventTypeNormal,
				}
				ensureK8sEventReceived(t, subscriptionCreatedEvent, givenSubscription.Namespace)
			}

			if tc.wantSubActiveEventCheck {
				subscriptionActiveEvent := kcore.Event{
					Reason:  string(eventingv1alpha2.ConditionReasonSubscriptionActive),
					Message: "",
					Type:    kcore.EventTypeNormal,
				}
				ensureK8sEventReceived(t, subscriptionActiveEvent, givenSubscription.Namespace)
			}
		})
	}
}

func Test_UpdateSubscription(t *testing.T) {
	t.Parallel()
	var testCases = []struct {
		name                           string
		givenSubscriptionFunc          func(namespace string) *eventingv1alpha2.Subscription
		givenUpdateSubscriptionFunc    func(namespace string) *eventingv1alpha2.Subscription
		wantSubscriptionMatchers       gomegatypes.GomegaMatcher
		wantUpdateSubscriptionMatchers gomegatypes.GomegaMatcher
	}{
		{
			name: "should succeed to update subscription when event type is added",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
			givenUpdateSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantUpdateSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1Event),
					}, {
						OriginalType: fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
		},
		{
			name: "should succeed to update subscription when event types are updated",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1Event),
					}, {
						OriginalType: fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
			givenUpdateSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0alpha", eventingtesting.OrderCreatedV1EventNotClean),
						fmt.Sprintf("%s1alpha", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantUpdateSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: fmt.Sprintf("%s0alpha", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s0alpha", eventingtesting.OrderCreatedV1Event),
					}, {
						OriginalType: fmt.Sprintf("%s1alpha", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s1alpha", eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
		},
		{
			name: "should succeed to update subscription when event type is deleted",
			givenSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1Event),
					}, {
						OriginalType: fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s1", eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
			givenUpdateSubscriptionFunc: func(namespace string) *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription(testName, namespace,
					eventingtesting.WithDefaultSource(),
					eventingtesting.WithTypes([]string{
						fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
					}),
					eventingtesting.WithWebhookAuthForEventMesh(),
					eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
				)
			},
			wantUpdateSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionActiveCondition(),
				eventingtesting.HaveCleanEventTypes([]eventingv1alpha2.EventType{
					{
						OriginalType: fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1EventNotClean),
						CleanType:    fmt.Sprintf("%s0", eventingtesting.OrderCreatedV1Event),
					},
				}),
			),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := gomega.NewGomegaWithT(t)
			ctx := context.Background()

			// create unique namespace for this test run
			testNamespace := getTestNamespace()
			ensureNamespaceCreated(ctx, t, testNamespace)

			// update namespace information in given test assets
			givenSubscription := tc.givenSubscriptionFunc(testNamespace)
			givenUpdateSubscription := tc.givenUpdateSubscriptionFunc(testNamespace)

			// create a subscriber service
			subscriberSvc := eventingtesting.NewSubscriberSvc(givenSubscription.Name, testNamespace)
			ensureK8sResourceCreated(ctx, t, subscriberSvc)

			// create subscription
			ensureK8sResourceCreated(ctx, t, givenSubscription)
			createdSubscription := givenSubscription.DeepCopy()
			// check if the created subscription is correct
			getSubscriptionAssert(ctx, g, createdSubscription).Should(tc.wantSubscriptionMatchers)

			// update subscription
			givenUpdateSubscription.ResourceVersion = createdSubscription.ResourceVersion
			ensureK8sSubscriptionUpdated(ctx, t, givenUpdateSubscription)

			// check if the updated subscription is correct
			getSubscriptionAssert(ctx, g, givenSubscription).Should(tc.wantUpdateSubscriptionMatchers)
		})
	}
}

func Test_DeleteSubscription(t *testing.T) {
	t.Parallel()

	// given
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	// create unique namespace for this test run
	testNamespace := getTestNamespace()
	ensureNamespaceCreated(ctx, t, testNamespace)
	subName := fmt.Sprintf("test-sink-%s", testNamespace)

	givenSubscription := eventingtesting.NewSubscription(subName, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(testNamespace, subName)),
	)

	// phase 1: Create a Subscription with ready APIRule and ready status.
	// create a subscriber service
	subscriberSvc := eventingtesting.NewSubscriberSvc(subName, testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvc)
	// create subscription
	ensureK8sResourceCreated(ctx, t, givenSubscription)
	createdSubscription := givenSubscription.DeepCopy()

	// wait until the APIRule is assigned to the created subscription
	getSubscriptionAssert(ctx, g, createdSubscription).Should(eventingtesting.HaveNoneEmptyAPIRuleName())

	// fetch the APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule := &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription.Status.Backend.APIRuleName, Namespace: createdSubscription.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule).Should(eventingtesting.HaveNotEmptyAPIRule())
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule)

	// check if corresponding subscription on EventMesh server exists
	emSub := getEventMeshSubFromMock(givenSubscription.Name, givenSubscription.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())

	// when
	// phase 2: Delete the Subscription from k8s
	// delete the Subscription and wait until its deleted.
	ensureK8sResourceDeleted(ctx, t, createdSubscription)
	getSubscriptionAssert(ctx, g, createdSubscription).Should(eventingtesting.IsAnEmptySubscription())

	// then
	// check if k8s event was triggered for Subscription deletion
	subscriptionDeletedEvent := kcore.Event{
		Reason:  string(eventingv1alpha2.ConditionReasonSubscriptionDeleted),
		Message: "",
		Type:    kcore.EventTypeWarning,
	}
	ensureK8sEventReceived(t, subscriptionDeletedEvent, givenSubscription.Namespace)

	// check if corresponding subscription on EventMesh server was also deleted
	emSub = getEventMeshSubFromMock(givenSubscription.Name, givenSubscription.Namespace)
	g.Expect(emSub).Should(gomega.BeNil())
}

func Test_FixingSinkAndApiRule(t *testing.T) {
	t.Parallel()
	// common given test assets
	givenSubscriptionWithoutSinkFunc := func(namespace, name string) *eventingv1alpha2.Subscription {
		return eventingtesting.NewSubscription(name, namespace,
			eventingtesting.WithDefaultSource(),
			eventingtesting.WithOrderCreatedV1Event(),
			eventingtesting.WithWebhookAuthForEventMesh(),
			// The following sink is invalid because it has an invalid svc name
			eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, "invalid")),
		)
	}

	wantSubscriptionWithoutSinkMatchers := gomega.And(
		eventingtesting.HaveCondition(eventingv1alpha2.MakeCondition(
			eventingv1alpha2.ConditionAPIRuleStatus,
			eventingv1alpha2.ConditionReasonAPIRuleStatusNotReady,
			kcore.ConditionFalse,
			invalidSinkErrMsg,
		)),
	)

	givenUpdateSubscriptionWithSinkFunc := func(namespace, name, sinkFormat, path string) *eventingv1alpha2.Subscription {
		return eventingtesting.NewSubscription(name, namespace,
			eventingtesting.WithDefaultSource(),
			eventingtesting.WithOrderCreatedV1Event(),
			eventingtesting.WithWebhookAuthForEventMesh(),
			eventingtesting.WithSink(fmt.Sprintf(sinkFormat, name, namespace, path)),
		)
	}

	wantUpdateSubscriptionWithSinkMatchers := gomega.And(
		eventingtesting.HaveSubscriptionReady(),
		eventingtesting.HaveSubscriptionActiveCondition(),
		eventingtesting.HaveAPIRuleTrueStatusCondition(),
	)

	// test cases
	var testCases = []struct {
		name            string
		givenSinkFormat string
	}{
		{
			name:            "should succeed to fix invalid sink with Url and port in subscription",
			givenSinkFormat: "https://%s.%s.svc.cluster.local:8080%s",
		},
		{
			name:            "should succeed to fix invalid sink with Url without port in subscription",
			givenSinkFormat: "https://%s.%s.svc.cluster.local%s",
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := gomega.NewGomegaWithT(t)
			ctx := context.Background()

			// create unique namespace for this test run
			testNamespace := getTestNamespace()
			ensureNamespaceCreated(ctx, t, testNamespace)
			subName := fmt.Sprintf("test-sink-%s", testNamespace)
			sinkPath := "/path1"

			// update namespace information in given test assets
			givenSubscription := givenSubscriptionWithoutSinkFunc(testNamespace, subName)
			givenUpdateSubscription := givenUpdateSubscriptionWithSinkFunc(testNamespace, subName, tc.givenSinkFormat, sinkPath)

			// create a subscriber service
			subscriberSvc := eventingtesting.NewSubscriberSvc(subName, testNamespace)
			ensureK8sResourceCreated(ctx, t, subscriberSvc)

			// create subscription
			ensureK8sResourceCreated(ctx, t, givenSubscription)
			createdSubscription := givenSubscription.DeepCopy()
			// check if the created subscription is correct
			getSubscriptionAssert(ctx, g, createdSubscription).Should(wantSubscriptionWithoutSinkMatchers)

			// update subscription with valid sink
			givenUpdateSubscription.ResourceVersion = createdSubscription.ResourceVersion
			ensureK8sSubscriptionUpdated(ctx, t, givenUpdateSubscription)

			// check if an APIRule was created for the subscription
			getAPIRuleForASvcAssert(ctx, g, subscriberSvc).Should(eventingtesting.HaveNotEmptyAPIRule())

			// check if the created APIRule is as required
			apiRules, err := getAPIRulesList(ctx, subscriberSvc)
			assert.NoError(t, err)
			apiRuleUpdated := filterAPIRulesForASvc(apiRules, subscriberSvc)
			getAPIRuleAssert(ctx, g, &apiRuleUpdated).Should(gomega.And(
				eventingtesting.HaveNotEmptyHost(),
				eventingtesting.HaveNotEmptyAPIRule(),
				eventingtesting.HaveAPIRuleSpecRules(
					acceptableMethods,
					object.OAuthHandlerNameJWT,
					certsURL,
					sinkPath,
				),
				eventingtesting.HaveAPIRuleOwnersRefs(givenSubscription.UID),
			))

			// update the status of the APIRule to ready (mocking APIGateway controller)
			ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, &apiRuleUpdated)

			// check if the updated subscription is correct
			getSubscriptionAssert(ctx, g, givenSubscription).Should(wantUpdateSubscriptionWithSinkMatchers)

			// check if the reconciled subscription has API rule in the status
			assert.Equal(t, givenSubscription.Status.Backend.APIRuleName, apiRuleUpdated.Name)

			// check if the EventMesh mock received requests
			_, postRequests, _ := countEventMeshRequests(
				emTestEnsemble.nameMapper.MapSubscriptionName(givenSubscription.Name, givenSubscription.Namespace),
				eventingtesting.EventMeshOrderCreatedV1Type)

			assert.GreaterOrEqual(t, postRequests, 1)
		})
	}
}

// Test_SinkChangeAndAPIRule tests the Subscription sink change scenario.
// The reconciler should update the EventMesh subscription webhookURL by creating a new APIRule
// when the sink is changed.
func Test_SinkChangeAndAPIRule(t *testing.T) {
	t.Parallel()

	// given
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	// create unique namespace for this test run
	testNamespace := getTestNamespace()
	ensureNamespaceCreated(ctx, t, testNamespace)
	subName := fmt.Sprintf("test-sink-%s", testNamespace)

	givenSubscription := eventingtesting.NewSubscription(subName, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(testNamespace, subName)),
	)

	// phase 1: Create a Subscription with ready APIRule and ready status.
	// create a subscriber service
	subscriberSvc := eventingtesting.NewSubscriberSvc(subName, testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvc)
	// create subscription
	ensureK8sResourceCreated(ctx, t, givenSubscription)
	createdSubscription := givenSubscription.DeepCopy()

	// wait until the APIRule is assigned to the created subscription
	getSubscriptionAssert(ctx, g, createdSubscription).Should(eventingtesting.HaveNoneEmptyAPIRuleName())

	// fetch the APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule := &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription.Status.Backend.APIRuleName, Namespace: createdSubscription.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule).Should(eventingtesting.HaveNotEmptyAPIRule())
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule)

	// check if the EventMesh Subscription has the correct webhook URL
	emSub := getEventMeshSubFromMock(givenSubscription.Name, givenSubscription.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())
	g.Expect(emSub).Should(eventmeshsubmatchers.HaveWebhookURL(fmt.Sprintf("https://%s/", *apiRule.Spec.Host)))

	// phase 2: Update the Subscription sink and check if new APIRule is created.
	// create a subscriber service
	subscriberSvcNew := eventingtesting.NewSubscriberSvc(fmt.Sprintf("%s2", subName), testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvcNew)

	// update subscription sink
	updatedSubscription := createdSubscription.DeepCopy()
	eventingtesting.SetSink(subscriberSvcNew.Namespace, subscriberSvcNew.Name, updatedSubscription)
	ensureK8sSubscriptionUpdated(ctx, t, updatedSubscription)
	// wait until the APIRule details are updated in Subscription.
	getSubscriptionAssert(ctx, g, updatedSubscription).Should(eventingtesting.HaveSubscriptionReady())
	getSubscriptionAssert(ctx, g, updatedSubscription).ShouldNot(eventingtesting.HaveAPIRuleName(apiRule.Name))

	// fetch the new APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule = &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: updatedSubscription.Status.Backend.APIRuleName, Namespace: updatedSubscription.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule).Should(eventingtesting.HaveNotEmptyAPIRule())
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule)

	// check if the EventMesh Subscription has the correct webhook URL
	emSub = getEventMeshSubFromMock(givenSubscription.Name, givenSubscription.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())
	g.Expect(emSub).Should(eventmeshsubmatchers.HaveWebhookURL(fmt.Sprintf("https://%s/", *apiRule.Spec.Host)))
}

// Test_APIRuleReUseAfterUpdatingSink tests two Subscriptions using different sinks
// which are then changed to use same sink, and they should use same APIRule.
func Test_APIRuleReUseAfterUpdatingSink(t *testing.T) {
	t.Parallel()

	// given
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	// create unique namespace for this test run
	testNamespace := getTestNamespace()
	ensureNamespaceCreated(ctx, t, testNamespace)

	// phase 1: Create the first Subscription with ready APIRule and ready status.
	// create a subscriber service
	sub1Name := fmt.Sprintf("test-sink-%s", testNamespace)
	subscriberSvc1 := eventingtesting.NewSubscriberSvc(sub1Name, testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvc1)
	// create subscription
	givenSubscription1 := eventingtesting.NewSubscription(sub1Name, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURLWithPath(testNamespace, sub1Name, "path1")),
	)
	ensureK8sResourceCreated(ctx, t, givenSubscription1)
	createdSubscription1 := givenSubscription1.DeepCopy()

	// wait until the APIRule is assigned to the created subscription
	getSubscriptionAssert(ctx, g, createdSubscription1).Should(eventingtesting.HaveNoneEmptyAPIRuleName())

	// fetch the APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule1 := &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription1.Status.Backend.APIRuleName, Namespace: createdSubscription1.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule1).Should(eventingtesting.HaveNotEmptyAPIRule())
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule1)

	// check if the EventMesh Subscription has the correct webhook URL
	emSub := getEventMeshSubFromMock(givenSubscription1.Name, givenSubscription1.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())
	g.Expect(emSub).Should(eventmeshsubmatchers.HaveWebhookURL(fmt.Sprintf("https://%s/path1", *apiRule1.Spec.Host)))

	// phase 2: Create the second Subscription (different sink) with ready APIRule and ready status.
	// create a subscriber service
	sub2Name := fmt.Sprintf("test-sink-%s-2", testNamespace)
	subscriberSvc2 := eventingtesting.NewSubscriberSvc(sub2Name, testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvc2)
	// create subscription
	givenSubscription2 := eventingtesting.NewSubscription(sub2Name, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURLWithPath(testNamespace, sub2Name, "path2")),
	)
	ensureK8sResourceCreated(ctx, t, givenSubscription2)
	createdSubscription2 := givenSubscription2.DeepCopy()

	// wait until the APIRule is assigned to the created subscription
	getSubscriptionAssert(ctx, g, createdSubscription2).Should(eventingtesting.HaveNoneEmptyAPIRuleName())
	getSubscriptionAssert(ctx, g, createdSubscription2).ShouldNot(eventingtesting.HaveAPIRuleName(apiRule1.Name))

	// fetch the APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule2 := &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription2.Status.Backend.APIRuleName, Namespace: createdSubscription2.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule2).Should(eventingtesting.HaveNotEmptyAPIRule())
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule2)

	// check if the EventMesh Subscription has the correct webhook URL
	emSub = getEventMeshSubFromMock(givenSubscription2.Name, givenSubscription2.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())
	g.Expect(emSub).Should(eventmeshsubmatchers.HaveWebhookURL(fmt.Sprintf("https://%s/path2", *apiRule2.Spec.Host)))

	// phase 3: Update the Subscription 2 to use same sink as in Subscription 1. The subscription 2 should then
	// re-use APIRule from Subscription 1.
	// update subscription 2 sink
	updatedSubscription2 := createdSubscription2.DeepCopy()
	updatedSubscription2.Spec.Sink = eventingtesting.ValidSinkURLWithPath(testNamespace, sub1Name, "path2")
	ensureK8sSubscriptionUpdated(ctx, t, updatedSubscription2)
	// wait until the APIRule details are updated in Subscription.
	getSubscriptionAssert(ctx, g, updatedSubscription2).Should(gomega.And(
		eventingtesting.HaveSubscriptionReady(),
		eventingtesting.HaveAPIRuleName(apiRule1.Name),
	))

	// check if the EventMesh Subscription has the correct webhook URL
	emSub = getEventMeshSubFromMock(givenSubscription2.Name, givenSubscription2.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())
	g.Expect(emSub).Should(eventmeshsubmatchers.HaveWebhookURL(fmt.Sprintf("https://%s/path2", *apiRule1.Spec.Host)))

	// fetch the re-used APIRule
	getAPIRuleAssert(ctx, g, apiRule1).Should(gomega.And(
		eventingtesting.HaveNotEmptyAPIRule(),
		eventingtesting.HaveAPIRuleOwnersRefs(createdSubscription1.UID, createdSubscription2.UID),
		eventingtesting.HaveAPIRuleSpecRules(
			acceptableMethods,
			object.OAuthHandlerNameJWT,
			certsURL,
			"/path1",
		),
		eventingtesting.HaveAPIRuleSpecRules(
			acceptableMethods,
			object.OAuthHandlerNameJWT,
			certsURL,
			"/path2",
		),
	))

	// phase 4: check that the unused APIRule is deleted.
	ensureAPIRuleNotFound(ctx, t, apiRule2)
}

// Test_APIRuleExistsAfterDeletingSub tests that if two Subscriptions are using same sinks (i.e. APIRule)
// then deleting one of the subscription should not delete the shared APIRule.
func Test_APIRuleExistsAfterDeletingSub(t *testing.T) {
	t.Parallel()

	// given
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	// create unique namespace for this test run
	testNamespace := getTestNamespace()
	ensureNamespaceCreated(ctx, t, testNamespace)

	// phase 1: Create two Subscriptions with same sink.
	// create a subscriber service
	sub1Name := fmt.Sprintf("test-sink-%s", testNamespace)
	sub2Name := fmt.Sprintf("test-sink-%s-2", testNamespace)
	subscriberSvc := eventingtesting.NewSubscriberSvc(sub1Name, testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvc)
	// create subscriptions
	givenSubscription1 := eventingtesting.NewSubscription(sub1Name, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURLWithPath(testNamespace, sub1Name, "path1")),
	)
	givenSubscription2 := eventingtesting.NewSubscription(sub2Name, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURLWithPath(testNamespace, sub1Name, "path2")),
	)

	ensureK8sResourceCreated(ctx, t, givenSubscription1)
	ensureK8sResourceCreated(ctx, t, givenSubscription2)
	createdSubscription1 := givenSubscription1.DeepCopy()
	createdSubscription2 := givenSubscription2.DeepCopy()

	// wait until the APIRule is assigned to the created subscriptions
	getSubscriptionAssert(ctx, g, createdSubscription1).Should(eventingtesting.HaveNoneEmptyAPIRuleName())
	getSubscriptionAssert(ctx, g, createdSubscription2).Should(eventingtesting.HaveNoneEmptyAPIRuleName())

	// verify that both subscriptions have same APIRule name
	getSubscriptionAssert(ctx, g, createdSubscription2).Should(eventingtesting.HaveAPIRuleName(
		createdSubscription1.Status.Backend.APIRuleName))

	// fetch the APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule1 := &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription1.Status.Backend.APIRuleName, Namespace: createdSubscription1.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule1).Should(gomega.And(
		eventingtesting.HaveNotEmptyAPIRule(),
		eventingtesting.HaveAPIRuleOwnersRefs(createdSubscription1.UID, createdSubscription2.UID),
		eventingtesting.HaveAPIRuleSpecRules(
			acceptableMethods,
			object.OAuthHandlerNameJWT,
			certsURL,
			"/path1",
		),
		eventingtesting.HaveAPIRuleSpecRules(
			acceptableMethods,
			object.OAuthHandlerNameJWT,
			certsURL,
			"/path2",
		),
	))
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule1)

	// phase 2: Delete the second Subscription and verify that the shared APIRule is not deleted.
	// delete the subscription and wait until deleted
	ensureK8sResourceDeleted(ctx, t, createdSubscription2)
	getSubscriptionAssert(ctx, g, createdSubscription2).Should(eventingtesting.IsAnEmptySubscription())

	// fetch the APIRule again and check
	apiRule1 = &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription1.Status.Backend.APIRuleName, Namespace: createdSubscription1.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule1).Should(gomega.And(
		eventingtesting.HaveNotEmptyAPIRule(),
		eventingtesting.HaveNotEmptyHost(),
		eventingtesting.HaveNotEmptyAPIRule(),
		eventingtesting.HaveAPIRuleOwnersRefs(createdSubscription1.UID),
		eventingtesting.HaveAPIRuleSpecRules(
			acceptableMethods,
			object.OAuthHandlerNameJWT,
			certsURL,
			"/path1",
		),
	))
	// ensure that the deleted Subscription is removed as Owner from the APIRule
	getAPIRuleAssert(ctx, g, apiRule1).ShouldNot(gomega.And(
		eventingtesting.HaveAPIRuleOwnersRefs(createdSubscription1.UID),
		eventingtesting.HaveAPIRuleSpecRules(
			acceptableMethods,
			object.OAuthHandlerNameJWT,
			certsURL,
			"/path2",
		),
	))
}

// Test_APIRuleRecreateAfterManualDelete tests that the APIRule is re-created by the reconciler
// when it is deleted manuallySubscription sink change scenario.
func Test_APIRuleRecreateAfterManualDelete(t *testing.T) {
	t.Parallel()

	// given
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	// create unique namespace for this test run
	testNamespace := getTestNamespace()
	ensureNamespaceCreated(ctx, t, testNamespace)
	subName := fmt.Sprintf("test-sink-%s", testNamespace)

	givenSubscription := eventingtesting.NewSubscription(subName, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(testNamespace, subName)),
	)

	// phase 1: Create a Subscription with ready APIRule and ready status.
	// create a subscriber service
	subscriberSvc := eventingtesting.NewSubscriberSvc(subName, testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvc)
	// create subscription
	ensureK8sResourceCreated(ctx, t, givenSubscription)
	createdSubscription := givenSubscription.DeepCopy()

	// wait until the APIRule is assigned to the created subscription
	getSubscriptionAssert(ctx, g, createdSubscription).Should(eventingtesting.HaveNoneEmptyAPIRuleName())

	// fetch the APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule := &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription.Status.Backend.APIRuleName, Namespace: createdSubscription.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule).Should(eventingtesting.HaveNotEmptyAPIRule())
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule)

	// phase 2: Delete the APIRule from k8s
	// delete the APIRule and wait until its deleted.
	ensureK8sResourceDeleted(ctx, t, apiRule)
	ensureAPIRuleNotFound(ctx, t, apiRule)

	// phase 3: Check if APIRule is re-created by the reconciler
	getSubscriptionAssert(ctx, g, createdSubscription).Should(eventingtesting.HaveNoneEmptyAPIRuleName())
	apiRule = &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription.Status.Backend.APIRuleName, Namespace: createdSubscription.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule).Should(eventingtesting.HaveNotEmptyAPIRule())
}

// Test_EventMeshSubRecreateAfterManualDelete tests that the EventMesh subscription is re-created by the reconciler
// when it is deleted manually on the EventMesh server.
func Test_EventMeshSubRecreateAfterManualDelete(t *testing.T) {
	t.Parallel()

	// given
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()

	// create unique namespace for this test run
	testNamespace := getTestNamespace()
	ensureNamespaceCreated(ctx, t, testNamespace)
	subName := fmt.Sprintf("test-sink-%s", testNamespace)

	givenSubscription := eventingtesting.NewSubscription(subName, testNamespace,
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithOrderCreatedV1Event(),
		eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(testNamespace, subName)),
	)

	// phase 1: Create a Subscription with ready APIRule and ready status.
	// create a subscriber service
	subscriberSvc := eventingtesting.NewSubscriberSvc(subName, testNamespace)
	ensureK8sResourceCreated(ctx, t, subscriberSvc)
	// create subscription
	ensureK8sResourceCreated(ctx, t, givenSubscription)
	createdSubscription := givenSubscription.DeepCopy()

	// wait until the APIRule is assigned to the created subscription
	getSubscriptionAssert(ctx, g, createdSubscription).Should(eventingtesting.HaveNoneEmptyAPIRuleName())

	// fetch the APIRule and update the status of the APIRule to ready (mocking APIGateway controller)
	// and wait until the created Subscription becomes ready
	apiRule := &apigateway.APIRule{ObjectMeta: kmeta.ObjectMeta{
		Name: createdSubscription.Status.Backend.APIRuleName, Namespace: createdSubscription.Namespace}}
	getAPIRuleAssert(ctx, g, apiRule).Should(eventingtesting.HaveNotEmptyAPIRule())
	ensureAPIRuleStatusUpdatedWithStatusReady(ctx, t, apiRule)

	// check if corresponding subscription on EventMesh server exists
	emSub := getEventMeshSubFromMock(createdSubscription.Name, createdSubscription.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())

	// when
	// phase 2: Delete the EventMesh Subscription manually
	// delete the EventMesh Subscription and confirm if its deleted.
	emTestEnsemble.eventMeshMock.Subscriptions.DeleteSubscriptionsByName(
		emTestEnsemble.nameMapper.MapSubscriptionName(givenSubscription.Name, givenSubscription.Namespace))
	g.Expect(getEventMeshSubFromMock(givenSubscription.Name, givenSubscription.Namespace)).Should(gomega.BeNil())

	// then
	// trigger reconciliation of Subscription by adding a label
	createdSubscription.Labels = map[string]string{"reconcile": "true"}
	ensureK8sSubscriptionUpdated(ctx, t, createdSubscription)

	// wait until Subscription is ready
	getSubscriptionAssert(ctx, g, createdSubscription).Should(eventingtesting.HaveSubscriptionReady())

	// check if corresponding subscription on EventMesh server was recreated
	emSub = getEventMeshSubFromMock(createdSubscription.Name, createdSubscription.Namespace)
	g.Expect(emSub).ShouldNot(gomega.BeNil())
}

func TestWithEventMeshServerErrors(t *testing.T) {
	t.Parallel()

	givenSubscriptionFunc := func(namespace string) *eventingv1alpha2.Subscription {
		return eventingtesting.NewSubscription(testName, namespace,
			eventingtesting.WithDefaultSource(),
			eventingtesting.WithOrderCreatedV1Event(),
			eventingtesting.WithSinkURL(eventingtesting.ValidSinkURL(namespace, testName)),
		)
	}

	var testCases = []struct {
		name                     string
		givenCreateResponseFunc  func(w http.ResponseWriter, _ eventMeshtypes.Subscription)
		wantSubscriptionMatchers gomegatypes.GomegaMatcher
		wantEventMeshSubMatchers gomegatypes.GomegaMatcher
	}{
		{
			name: "should not be ready when when EventMesh server is not able to create new EventMesh subscriptions",
			givenCreateResponseFunc: func(w http.ResponseWriter, _ eventMeshtypes.Subscription) {
				// ups ... server returns 500
				w.WriteHeader(http.StatusInternalServerError)
				s := eventMeshtypes.Response{
					StatusCode: http.StatusInternalServerError,
					Message:    "sorry, but this mock does not let you create a EventMesh subscription",
				}
				err := json.NewEncoder(w).Encode(s)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionNotReady(),
				eventingtesting.HaveCondition(eventingv1alpha2.MakeCondition(
					eventingv1alpha2.ConditionSubscribed,
					eventingv1alpha2.ConditionReasonSubscriptionCreationFailed,
					kcore.ConditionFalse,
					"failed to get subscription from EventMesh: create subscription failed: 500; 500 Internal"+
						" Server Error;{\"message\":\"sorry, but this mock does not let you create a"+
						" EventMesh subscription\"}\n",
				),
				),
			),
			wantEventMeshSubMatchers: gomega.And(
				gomega.BeNil(),
			),
		},
		{
			name: "should not be ready when EventMesh server subscription is paused",
			givenCreateResponseFunc: func(w http.ResponseWriter, sub eventMeshtypes.Subscription) {
				sub.SubscriptionStatus = eventMeshtypes.SubscriptionStatusPaused
				subKey := getEventMeshKeyForMock(sub.Name)
				emTestEnsemble.eventMeshMock.Subscriptions.PutSubscription(subKey, &sub)
				eventingtesting.EventMeshCreateSuccess(w)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionNotReady(),
				eventingtesting.HaveCondition(eventingv1alpha2.MakeCondition(
					eventingv1alpha2.ConditionSubscriptionActive,
					eventingv1alpha2.ConditionReasonSubscriptionNotActive,
					kcore.ConditionFalse,
					"Waiting for subscription to be active"),
				),
			),
			wantEventMeshSubMatchers: gomega.And(
				eventmeshsubmatchers.HaveStatusPaused(),
			),
		},
		{
			name: "when EventMesh server subscription webhook is unauthorized",
			givenCreateResponseFunc: func(w http.ResponseWriter, sub eventMeshtypes.Subscription) {
				sub.SubscriptionStatus = eventMeshtypes.SubscriptionStatusActive
				sub.LastSuccessfulDelivery = time.Now().Format(time.RFC3339)                   // "now",
				sub.LastFailedDelivery = time.Now().Add(10 * time.Second).Format(time.RFC3339) // "now + 10s"
				sub.LastFailedDeliveryReason = "Webhook endpoint response code: 401"

				subKey := getEventMeshKeyForMock(sub.Name)
				emTestEnsemble.eventMeshMock.Subscriptions.PutSubscription(subKey, &sub)
				eventingtesting.EventMeshCreateSuccess(w)
			},
			wantSubscriptionMatchers: gomega.And(
				eventingtesting.HaveSubscriptionNotReady(),
				eventingtesting.HaveCondition(eventingv1alpha2.MakeCondition(
					eventingv1alpha2.ConditionWebhookCallStatus,
					eventingv1alpha2.ConditionReasonWebhookCallStatus,
					kcore.ConditionFalse,
					"Webhook endpoint response code: 401"),
				),
			),
			wantEventMeshSubMatchers: gomega.And(
				eventmeshsubmatchers.HaveStatusActive(),
				eventmeshsubmatchers.HaveNonEmptyLastFailedDeliveryReason(),
			),
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewGomegaWithT(t)
			ctx := context.Background()

			// create unique namespace for this test run
			testNamespace := getTestNamespace()
			ensureNamespaceCreated(ctx, t, testNamespace)

			// given
			// update namespace information in given test assets
			givenSubscription := givenSubscriptionFunc(testNamespace)

			// override create request response in EventMesh mock
			subKey := getEventMeshSubKeyForMock(givenSubscription.Name, givenSubscription.Namespace)
			emTestEnsemble.eventMeshMock.AddCreateResponseOverride(subKey, tc.givenCreateResponseFunc)

			// when
			// create a subscriber service
			subscriberSvc := eventingtesting.NewSubscriberSvc(testName, testNamespace)
			ensureK8sResourceCreated(ctx, t, subscriberSvc)
			// create subscription
			ensureK8sResourceCreated(ctx, t, givenSubscription)
			createdSubscription := givenSubscription.DeepCopy()

			// then
			// wait until the subscription shows the condition with not-ready status
			getSubscriptionAssert(ctx, g, createdSubscription).Should(tc.wantSubscriptionMatchers)

			// check subscription on EventMesh server
			emSub := getEventMeshSubFromMock(createdSubscription.Name, createdSubscription.Namespace)
			g.Expect(emSub).Should(tc.wantEventMeshSubMatchers)

			// delete the subscription to not provoke more reconciliation requests
			ensureK8sResourceDeleted(ctx, t, createdSubscription)
		})
	}
}
