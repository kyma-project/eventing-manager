package eventmesh

import (
	"context"
	"fmt"
	"testing"
	"time"

	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	kctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/internal/controller/eventing/subscription/validator"
	subscriptionvalidatormocks "github.com/kyma-project/eventing-manager/internal/controller/eventing/subscription/validator/mocks"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	"github.com/kyma-project/eventing-manager/pkg/backend/eventmesh"
	"github.com/kyma-project/eventing-manager/pkg/backend/eventmesh/mocks"
	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	backendutils "github.com/kyma-project/eventing-manager/pkg/backend/utils"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/featureflags"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	"github.com/kyma-project/eventing-manager/test/utils"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

// TestReconciler_Reconcile tests the return values of the Reconcile() method of the reconciler.
// This is important, as it dictates whether the reconciliation should be requeued by Controller Runtime,
// and if so with how much initial delay. Returning error or a `Result{Requeue: true}` would cause the reconciliation to be requeued.
// Everything else is mocked since we are only interested in the logic of the Reconcile method and not the reconciler dependencies.
func TestReconciler_Reconcile(t *testing.T) {
	req := require.New(t)
	col := metrics.NewCollector()

	// A subscription with the correct Finalizer, conditions and status ready for reconciliation with the backend.
	testSub := eventingtesting.NewSubscription("sub1", "test",
		eventingtesting.WithConditions(eventingv1alpha2.MakeSubscriptionConditions()),
		eventingtesting.WithFinalizers([]string{eventingv1alpha2.Finalizer}),
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedEventType),
		eventingtesting.WithValidSink("test", "test-svc"),
		eventingtesting.WithEmsSubscriptionStatus(string(types.SubscriptionStatusActive)),
		eventingtesting.WithMaxInFlight(10),
	)
	// A subscription marked for deletion.
	testSubUnderDeletion := eventingtesting.NewSubscription("sub2", "test",
		eventingtesting.WithNonZeroDeletionTimestamp(),
		eventingtesting.WithFinalizers([]string{eventingv1alpha2.Finalizer}),
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedEventType),
	)

	// A subscription with the correct Finalizer, conditions and status Paused for reconciliation with the backend.
	testSubPaused := eventingtesting.NewSubscription("sub3", "test",
		eventingtesting.WithConditions(eventingv1alpha2.MakeSubscriptionConditions()),
		eventingtesting.WithFinalizers([]string{eventingv1alpha2.Finalizer}),
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedEventType),
		eventingtesting.WithValidSink("test", "test-svc"),
		eventingtesting.WithEmsSubscriptionStatus(string(types.SubscriptionStatusPaused)),
		eventingtesting.WithMaxInFlight(10),
	)

	testService := eventingtesting.NewService("test-svc", "test")

	backendSyncErr := errors.New("backend sync error")
	backendDeleteErr := errors.New("backend delete error")
	validatorErr := errors.New("invalid sink")
	happyValidator := validator.SubscriptionValidatorFunc(func(_ context.Context, _ eventingv1alpha2.Subscription) error { return nil })
	unhappyValidator := validator.SubscriptionValidatorFunc(func(_ context.Context, _ eventingv1alpha2.Subscription) error { return validatorErr })

	testCases := []struct {
		name                 string
		givenSubscription    *eventingv1alpha2.Subscription
		givenReconcilerSetup func() *Reconciler
		wantReconcileResult  kctrl.Result
		wantReconcileError   error
	}{
		{
			name:              "Return nil and default Result{} when there is no error from the reconciler dependencies",
			givenSubscription: testSub,
			givenReconcilerSetup: func() *Reconciler {
				testEnv := setupTestEnvironment(t, testSub, testService)
				testEnv.backend.On("Initialize", mock.Anything).Return(nil)
				testEnv.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				return NewReconciler(
					testEnv.fakeClient,
					testEnv.logger,
					testEnv.recorder,
					testEnv.cfg,
					testEnv.cleaner,
					testEnv.backend,
					testEnv.credentials,
					testEnv.mapper,
					happyValidator,
					col,
					utils.Domain)
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  nil,
		},
		{
			name:              "Return nil and default Result{} when the subscription does not exist on the cluster",
			givenSubscription: testSub,
			givenReconcilerSetup: func() *Reconciler {
				testenv := setupTestEnvironment(t, testService)
				testenv.backend.On("Initialize", mock.Anything).Return(nil)
				return NewReconciler(
					testenv.fakeClient,
					testenv.logger,
					testenv.recorder,
					testenv.cfg,
					testenv.cleaner,
					testenv.backend,
					testenv.credentials,
					testenv.mapper,
					unhappyValidator,
					col,
					utils.Domain)
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  nil,
		},
		{
			name:              "Return error and default Result{} when backend sync returns error",
			givenSubscription: testSub,
			givenReconcilerSetup: func() *Reconciler {
				testenv := setupTestEnvironment(t, testSub, testService)
				testenv.backend.On("Initialize", mock.Anything).Return(nil)
				testenv.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(false, backendSyncErr)
				return NewReconciler(
					testenv.fakeClient,
					testenv.logger,
					testenv.recorder,
					testenv.cfg,
					testenv.cleaner,
					testenv.backend,
					testenv.credentials,
					testenv.mapper,
					happyValidator,
					col,
					utils.Domain)
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  backendSyncErr,
		},
		{
			name:              "Return error and default Result{} when backend delete returns error",
			givenSubscription: testSubUnderDeletion,
			givenReconcilerSetup: func() *Reconciler {
				testenv := setupTestEnvironment(t, testSubUnderDeletion, testService)
				testenv.backend.On("Initialize", mock.Anything).Return(nil)
				testenv.backend.On("DeleteSubscription", mock.Anything).Return(backendDeleteErr)
				return NewReconciler(
					testenv.fakeClient,
					testenv.logger,
					testenv.recorder,
					testenv.cfg,
					testenv.cleaner,
					testenv.backend,
					testenv.credentials,
					testenv.mapper,
					happyValidator,
					col,
					utils.Domain)
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  backendDeleteErr,
		},
		{
			name:              "Return error and default Result{} when validator returns error",
			givenSubscription: testSub,
			givenReconcilerSetup: func() *Reconciler {
				testenv := setupTestEnvironment(t, testSub, testService)
				testenv.backend.On("Initialize", mock.Anything).Return(nil)
				return NewReconciler(
					testenv.fakeClient,
					testenv.logger,
					testenv.recorder,
					testenv.cfg,
					testenv.cleaner,
					testenv.backend,
					testenv.credentials,
					testenv.mapper,
					unhappyValidator,
					col,
					utils.Domain)
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  reconcile.TerminalError(validatorErr),
		},
		{
			name:              "Return nil and RequeueAfter when the EventMesh subscription is Paused",
			givenSubscription: testSubPaused,
			givenReconcilerSetup: func() *Reconciler {
				testenv := setupTestEnvironment(t, testSubPaused, testService)
				testenv.backend.On("Initialize", mock.Anything).Return(nil)
				testenv.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				return NewReconciler(
					testenv.fakeClient,
					testenv.logger,
					testenv.recorder,
					testenv.cfg,
					testenv.cleaner,
					testenv.backend,
					testenv.credentials,
					testenv.mapper,
					happyValidator,
					col,
					utils.Domain)
			},
			wantReconcileResult: kctrl.Result{
				RequeueAfter: requeueAfterDuration,
			},
			wantReconcileError: nil,
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			reconciler := testcase.givenReconcilerSetup()
			r := kctrl.Request{NamespacedName: ktypes.NamespacedName{
				Namespace: testcase.givenSubscription.Namespace,
				Name:      testcase.givenSubscription.Name,
			}}
			res, err := reconciler.Reconcile(context.Background(), r)
			req.Equal(testcase.wantReconcileResult, res)
			req.Equal(testcase.wantReconcileError, err)
		})
	}
}

// TestReconciler_APIRuleConfig ensures that the created APIRule is configured correctly
// whether the Eventing webhook auth is enabled or not.
func TestReconciler_APIRuleConfig(t *testing.T) {
	ctx := context.Background()

	credentials := &eventmesh.OAuth2ClientCredentials{
		CertsURL: "https://domain.com/oauth2/certs",
	}

	subscription := eventingtesting.NewSubscription("some-test-sub", "test",
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithValidSink("test", "some-test-svc"),
		eventingtesting.WithFinalizers([]string{eventingv1alpha2.Finalizer}),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedEventType),
		eventingtesting.WithConditions(eventingv1alpha2.MakeSubscriptionConditions()),
		eventingtesting.WithEmsSubscriptionStatus(string(types.SubscriptionStatusActive)),
		eventingtesting.WithMaxInFlight(10),
	)

	subscriptionValidator := validator.SubscriptionValidatorFunc(func(_ context.Context, _ eventingv1alpha2.Subscription) error { return nil })

	col := metrics.NewCollector()

	testCases := []struct {
		name                            string
		givenSubscription               *eventingv1alpha2.Subscription
		givenReconcilerSetup            func() (*Reconciler, client.Client)
		givenEventingWebhookAuthEnabled bool
		wantReconcileResult             kctrl.Result
		wantReconcileError              error
		jwt                             *apigatewayv2.JwtConfig
		extAuth                         apigatewayv2.ExtAuth
	}{
		{
			name:              "Eventing webhook auth is not enabled",
			givenSubscription: subscription,
			givenReconcilerSetup: func() (*Reconciler, client.Client) {
				testenv := setupTestEnvironment(t, subscription)
				testenv.credentials = credentials
				testenv.backend.On("Initialize", mock.Anything).Return(nil)
				testenv.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				return NewReconciler(
						testenv.fakeClient,
						testenv.logger,
						testenv.recorder,
						testenv.cfg,
						testenv.cleaner,
						testenv.backend,
						testenv.credentials,
						testenv.mapper,
						subscriptionValidator,
						col,
						utils.Domain),
					testenv.fakeClient
			},
			givenEventingWebhookAuthEnabled: false,
			wantReconcileResult:             kctrl.Result{},
			wantReconcileError:              nil,
			extAuth:                         apigatewayv2.ExtAuth{},
		},
		{
			name:              "Eventing webhook auth is enabled",
			givenSubscription: subscription,
			givenReconcilerSetup: func() (*Reconciler, client.Client) {
				testenv := setupTestEnvironment(t, subscription)
				testenv.credentials = credentials
				testenv.backend.On("Initialize", mock.Anything).Return(nil)
				testenv.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				return NewReconciler(
						testenv.fakeClient,
						testenv.logger,
						testenv.recorder,
						testenv.cfg,
						testenv.cleaner,
						testenv.backend,
						testenv.credentials,
						testenv.mapper,
						subscriptionValidator,
						col,
						utils.Domain),
					testenv.fakeClient
			},
			givenEventingWebhookAuthEnabled: true,
			wantReconcileResult:             kctrl.Result{},
			wantReconcileError:              nil,
			jwt: &apigatewayv2.JwtConfig{
				Authentications: []*apigatewayv2.JwtAuthentication{
					{
						JwksUri: credentials.CertsURL,
						Issuer:  "https://domain.com",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			featureflags.SetEventingWebhookAuthEnabled(testcase.givenEventingWebhookAuthEnabled)
			reconciler, cli := testcase.givenReconcilerSetup()
			namespacedName := ktypes.NamespacedName{
				Namespace: testcase.givenSubscription.Namespace,
				Name:      testcase.givenSubscription.Name,
			}

			// when
			res, err := reconciler.Reconcile(context.Background(), kctrl.Request{NamespacedName: namespacedName})
			require.Equal(t, testcase.wantReconcileResult, res)
			require.Equal(t, testcase.wantReconcileError, err)

			sub := &eventingv1alpha2.Subscription{}
			err = cli.Get(ctx, namespacedName, sub)
			require.NoError(t, err)

			namespacedName = ktypes.NamespacedName{
				Namespace: sub.Namespace,
				Name:      sub.Status.Backend.APIRuleName,
			}

			apiRule := &apigatewayv2.APIRule{}
			err = cli.Get(ctx, namespacedName, apiRule)
			require.NoError(t, err)

			// then
			if testcase.givenEventingWebhookAuthEnabled {
				require.Equal(t, testcase.jwt, apiRule.Spec.Rules[0].Jwt)
			} else {
				require.Equal(t, &testcase.extAuth, apiRule.Spec.Rules[0].ExtAuth)
			}
		})
	}
}

// TestReconciler_PreserveBackendHashes ensures that the precomputed EventMesh hashes in the Kyma subscription
// is preserved after reconciliation.
func TestReconciler_PreserveBackendHashes(t *testing.T) {
	ctx := context.Background()
	collector := metrics.NewCollector()
	subscriptionValidator := validator.SubscriptionValidatorFunc(func(_ context.Context, _ eventingv1alpha2.Subscription) error { return nil })

	const (
		ev2hash            = int64(118518533334734)
		eventMeshHash      = int64(748405436686967)
		webhookAuthHash    = int64(118518533334734)
		eventMeshLocalHash = int64(883494500014499)
	)

	testCases := []struct {
		name                   string
		givenSubscription      *eventingv1alpha2.Subscription
		givenReconcilerSetup   func(*eventingv1alpha2.Subscription) (*Reconciler, client.Client)
		wantEv2Hash            int64
		wantEventMeshHash      int64
		wantWebhookAuthHash    int64
		wantEventMeshLocalHash int64
		wantReconcileError     error
	}{
		{
			name: "Preserve hashes if conditions are empty",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription("some-test-sub-0", "test",
					eventingtesting.WithValidSink("test", "some-test-svc-0"),
					eventingtesting.WithConditions(nil),
					eventingtesting.WithBackend(eventingv1alpha2.Backend{
						Ev2hash:            ev2hash,
						EventMeshHash:      eventMeshHash,
						WebhookAuthHash:    webhookAuthHash,
						EventMeshLocalHash: eventMeshLocalHash,
					}),
					eventingtesting.WithMaxInFlight(10),
					eventingtesting.WithSource(eventingtesting.EventSourceClean),
					eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				)
			}(),
			givenReconcilerSetup: func(s *eventingv1alpha2.Subscription) (*Reconciler, client.Client) {
				te := setupTestEnvironment(t, s)
				te.backend.On("Initialize", mock.Anything).Return(nil)
				te.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				return NewReconciler(te.fakeClient, te.logger, te.recorder, te.cfg, te.cleaner,
					te.backend, te.credentials, te.mapper, subscriptionValidator, collector, utils.Domain), te.fakeClient
			},
			wantEv2Hash:            ev2hash,
			wantEventMeshHash:      eventMeshHash,
			wantWebhookAuthHash:    webhookAuthHash,
			wantEventMeshLocalHash: eventMeshLocalHash,
			wantReconcileError:     nil,
		},
		{
			name: "Preserve hashes if conditions are not empty",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				return eventingtesting.NewSubscription("some-test-sub-1", "test",
					eventingtesting.WithValidSink("test", "some-test-svc-1"),
					eventingtesting.WithConditions(eventingv1alpha2.MakeSubscriptionConditions()),
					eventingtesting.WithBackend(eventingv1alpha2.Backend{
						Ev2hash:            ev2hash,
						EventMeshHash:      eventMeshHash,
						WebhookAuthHash:    webhookAuthHash,
						EventMeshLocalHash: eventMeshLocalHash,
					}),
					eventingtesting.WithMaxInFlight(10),
					eventingtesting.WithSource(eventingtesting.EventSourceClean),
					eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				)
			}(),
			givenReconcilerSetup: func(s *eventingv1alpha2.Subscription) (*Reconciler, client.Client) {
				te := setupTestEnvironment(t, s)
				te.backend.On("Initialize", mock.Anything).Return(nil)
				te.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				return NewReconciler(te.fakeClient, te.logger, te.recorder, te.cfg, te.cleaner,
					te.backend, te.credentials, te.mapper, subscriptionValidator, collector, utils.Domain), te.fakeClient
			},
			wantEv2Hash:            ev2hash,
			wantEventMeshHash:      eventMeshHash,
			wantWebhookAuthHash:    webhookAuthHash,
			wantEventMeshLocalHash: eventMeshLocalHash,
			wantReconcileError:     nil,
		},
	}
	featureFlagValues := []bool{true, false}
	for _, tc := range testCases {
		for _, value := range featureFlagValues {
			testcase := tc
			flag := value
			t.Run(fmt.Sprintf("%s [EventingWebhookAuthEnabled=%v]", testcase.name, flag), func(t *testing.T) {
				// given
				featureflags.SetEventingWebhookAuthEnabled(flag)
				reconciler, cli := testcase.givenReconcilerSetup(testcase.givenSubscription)
				reconciler.syncConditionWebhookCallStatus = func(subscription *eventingv1alpha2.Subscription) {}
				namespacedName := ktypes.NamespacedName{
					Namespace: testcase.givenSubscription.Namespace,
					Name:      testcase.givenSubscription.Name,
				}

				// when
				_, err := reconciler.Reconcile(context.Background(), kctrl.Request{NamespacedName: namespacedName})
				require.Equal(t, testcase.wantReconcileError, err)

				// then
				sub := &eventingv1alpha2.Subscription{}
				err = cli.Get(ctx, namespacedName, sub)
				require.NoError(t, err)
				require.Equal(t, testcase.wantEv2Hash, sub.Status.Backend.Ev2hash)
				require.Equal(t, testcase.wantEventMeshHash, sub.Status.Backend.EventMeshHash)
				require.Equal(t, testcase.wantWebhookAuthHash, sub.Status.Backend.WebhookAuthHash)
				require.Equal(t, testcase.wantEventMeshLocalHash, sub.Status.Backend.EventMeshLocalHash)
			})
		}
	}
}

func Test_replaceStatusCondition(t *testing.T) {
	testCases := []struct {
		name              string
		giveSubscription  *eventingv1alpha2.Subscription
		giveCondition     eventingv1alpha2.Condition
		wantStatusChanged bool
		wantStatus        *eventingv1alpha2.SubscriptionStatus
		wantReady         bool
	}{
		{
			name: "Updating a condition marks the status as changed",
			giveSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace",
					eventingtesting.WithNotCleanSource(),
					eventingtesting.WithNotCleanType())
				sub.Status.InitializeConditions()
				return sub
			}(),
			giveCondition: func() eventingv1alpha2.Condition {
				return eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscribed, eventingv1alpha2.ConditionReasonSubscriptionCreated, kcorev1.ConditionTrue, "")
			}(),
			wantStatusChanged: true,
			wantReady:         false,
		},
		{
			name: "All conditions true means status is ready",
			giveSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace",
					eventingtesting.WithNotCleanSource(),
					eventingtesting.WithNotCleanType(),
					eventingtesting.WithWebhookAuthForEventMesh())
				sub.Status.InitializeConditions()
				sub.Status.Ready = false

				// mark all conditions as true
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscribed,
						LastTransitionTime: kmetav1.Now(),
						Status:             kcorev1.ConditionTrue,
					},
					{
						Type:               eventingv1alpha2.ConditionSubscriptionActive,
						LastTransitionTime: kmetav1.Now(),
						Status:             kcorev1.ConditionTrue,
					},
				}
				return sub
			}(),
			giveCondition: func() eventingv1alpha2.Condition {
				return eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscribed, eventingv1alpha2.ConditionReasonSubscriptionCreated, kcorev1.ConditionTrue, "")
			}(),
			wantStatusChanged: true, // readiness changed
			wantReady:         true, // all conditions are true
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			sub := testcase.giveSubscription
			condition := testcase.giveCondition
			statusChanged := replaceStatusCondition(sub, condition)
			assert.Equal(t, testcase.wantStatusChanged, statusChanged)
			assert.Contains(t, sub.Status.Conditions, condition)
			assert.Equal(t, testcase.wantReady, sub.Status.Ready)
		})
	}
}

func Test_getRequiredConditions(t *testing.T) {
	var emptySubscriptionStatus eventingv1alpha2.SubscriptionStatus
	emptySubscriptionStatus.InitializeConditions()
	expectedConditions := emptySubscriptionStatus.Conditions

	testCases := []struct {
		name                   string
		subscriptionConditions []eventingv1alpha2.Condition
		wantConditions         []eventingv1alpha2.Condition
	}{
		{
			name:                   "should get expected conditions if the subscription has no conditions",
			subscriptionConditions: []eventingv1alpha2.Condition{},
			wantConditions:         expectedConditions,
		},
		{
			name: "should get subscription conditions if the all the expected conditions are present",
			subscriptionConditions: []eventingv1alpha2.Condition{
				{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: eventingv1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionSubscriptionSpecValid, Status: kcorev1.ConditionFalse},
			},
			wantConditions: []eventingv1alpha2.Condition{
				{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: eventingv1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionSubscriptionSpecValid, Status: kcorev1.ConditionFalse},
			},
		},
		{
			name: "should get latest conditions Status compared to the expected condition status",
			subscriptionConditions: []eventingv1alpha2.Condition{
				{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
			},
			wantConditions: []eventingv1alpha2.Condition{
				{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionSubscriptionSpecValid, Status: kcorev1.ConditionUnknown},
			},
		},
		{
			name: "should get rid of unwanted conditions in the subscription, if present",
			subscriptionConditions: []eventingv1alpha2.Condition{
				{Type: "Fake Condition Type", Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
			},
			wantConditions: []eventingv1alpha2.Condition{
				{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
				{Type: eventingv1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionUnknown},
				{Type: eventingv1alpha2.ConditionSubscriptionSpecValid, Status: kcorev1.ConditionUnknown},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotConditions := getRequiredConditions(tc.subscriptionConditions, expectedConditions)
			assert.True(t, eventingv1alpha2.ConditionsEquals(gotConditions, tc.wantConditions))
			assert.Len(t, gotConditions, len(expectedConditions))
		})
	}
}

func Test_syncConditionSubscribed(t *testing.T) {
	currentTime := kmetav1.Now()
	errorMessage := "error message"
	testCases := []struct {
		name              string
		givenSubscription *eventingv1alpha2.Subscription
		givenError        error
		wantCondition     eventingv1alpha2.Condition
	}{
		{
			name: "should replace condition with status false",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Ready = false

				// mark ConditionSubscribed conditions as true
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscribed,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionTrue,
					},
					{
						Type:               eventingv1alpha2.ConditionSubscriptionActive,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionTrue,
					},
				}
				return sub
			}(),
			givenError: errors.New(errorMessage),
			wantCondition: eventingv1alpha2.Condition{
				Type:               eventingv1alpha2.ConditionSubscribed,
				LastTransitionTime: currentTime,
				Status:             kcorev1.ConditionFalse,
				Reason:             eventingv1alpha2.ConditionReasonSubscriptionCreationFailed,
				Message:            errorMessage,
			},
		},
		{
			name: "should replace condition with status true",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Ready = false

				// mark ConditionSubscribed conditions as false
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscribed,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionFalse,
					},
					{
						Type:               eventingv1alpha2.ConditionSubscriptionActive,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionTrue,
					},
				}
				return sub
			}(),
			givenError: nil,
			wantCondition: eventingv1alpha2.Condition{
				Type:               eventingv1alpha2.ConditionSubscribed,
				LastTransitionTime: currentTime,
				Status:             kcorev1.ConditionTrue,
				Reason:             eventingv1alpha2.ConditionReasonSubscriptionCreated,
				Message:            "EventMesh subscription name is: some-namef73aa86661706ae6ba5acf1d32821ce318051d0e",
			},
		},
	}

	reconciler := Reconciler{
		nameMapper: backendutils.NewBEBSubscriptionNameMapper(
			utils.Domain,
			eventmesh.MaxSubscriptionNameLength,
		),
		syncConditionWebhookCallStatus: syncConditionWebhookCallStatus,
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			sub := testcase.givenSubscription

			// when
			reconciler.syncConditionSubscribed(sub, testcase.givenError)

			// then
			newCondition := sub.Status.FindCondition(testcase.wantCondition.Type)
			assert.NotNil(t, newCondition)
			assert.True(t, eventingv1alpha2.ConditionEquals(*newCondition, testcase.wantCondition))
		})
	}
}

func Test_syncConditionSubscriptionActive(t *testing.T) {
	currentTime := kmetav1.Now()

	testCases := []struct {
		name              string
		givenSubscription *eventingv1alpha2.Subscription
		givenIsSubscribed bool
		wantCondition     eventingv1alpha2.Condition
	}{
		{
			name: "should replace condition with status false",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Ready = false
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					Status: "Paused",
				}

				// mark ConditionSubscriptionActive conditions as true
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscribed,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionTrue,
					},
					{
						Type:               eventingv1alpha2.ConditionSubscriptionActive,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionTrue,
					},
				}
				return sub
			}(),
			givenIsSubscribed: false,
			wantCondition: eventingv1alpha2.Condition{
				Type:               eventingv1alpha2.ConditionSubscriptionActive,
				LastTransitionTime: currentTime,
				Status:             kcorev1.ConditionFalse,
				Reason:             eventingv1alpha2.ConditionReasonSubscriptionNotActive,
				Message:            "Waiting for subscription to be active",
			},
		},
		{
			name: "should replace condition with status true",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Ready = false
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{}

				// mark ConditionSubscriptionActive conditions as false
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscribed,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionFalse,
					},
					{
						Type:               eventingv1alpha2.ConditionSubscriptionActive,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionFalse,
					},
				}
				return sub
			}(),
			givenIsSubscribed: true,
			wantCondition: eventingv1alpha2.Condition{
				Type:               eventingv1alpha2.ConditionSubscriptionActive,
				LastTransitionTime: currentTime,
				Status:             kcorev1.ConditionTrue,
				Reason:             eventingv1alpha2.ConditionReasonSubscriptionActive,
			},
		},
	}

	logger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	if err != nil {
		t.Fatalf(`failed to initiate logger, %v`, err)
	}

	reconciler := Reconciler{
		nameMapper: backendutils.NewBEBSubscriptionNameMapper(utils.Domain, eventmesh.MaxSubscriptionNameLength),
		logger:     logger,
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			sub := testcase.givenSubscription
			log := backendutils.LoggerWithSubscription(reconciler.namedLogger(), sub)

			// when
			reconciler.syncConditionSubscriptionActive(sub, testcase.givenIsSubscribed, log)

			// then
			newCondition := sub.Status.FindCondition(testcase.wantCondition.Type)
			assert.NotNil(t, newCondition)
			assert.True(t, eventingv1alpha2.ConditionEquals(*newCondition, testcase.wantCondition))
		})
	}
}

func Test_syncConditionWebhookCallStatus(t *testing.T) {
	currentTime := kmetav1.Now()

	testCases := []struct {
		name              string
		givenSubscription *eventingv1alpha2.Subscription
		givenIsSubscribed bool
		wantCondition     eventingv1alpha2.Condition
	}{
		{
			name: "should replace condition with status false if it throws error to check lastDelivery",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Ready = false

				// mark ConditionWebhookCallStatus conditions as true
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscriptionActive,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionTrue,
					},
					{
						Type:               eventingv1alpha2.ConditionWebhookCallStatus,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionTrue,
					},
				}
				// set EventMeshSubscriptionStatus
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   "invalid",
					LastFailedDelivery:       "invalid",
					LastFailedDeliveryReason: "",
				}
				return sub
			}(),
			givenIsSubscribed: false,
			wantCondition: eventingv1alpha2.Condition{
				Type:               eventingv1alpha2.ConditionWebhookCallStatus,
				LastTransitionTime: currentTime,
				Status:             kcorev1.ConditionFalse,
				Reason:             eventingv1alpha2.ConditionReasonWebhookCallStatus,
				Message:            `failed to parse LastFailedDelivery: parsing time "invalid" as "2006-01-02T15:04:05Z07:00": cannot parse "invalid" as "2006"`,
			},
		},
		{
			name: "should replace condition with status false if lastDelivery was not okay",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Ready = false

				// mark ConditionWebhookCallStatus conditions as false
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscribed,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionFalse,
					},
					{
						Type:               eventingv1alpha2.ConditionWebhookCallStatus,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionFalse,
					},
				}
				// set EventMeshSubscriptionStatus
				// LastFailedDelivery is latest
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   time.Now().Format(time.RFC3339),
					LastFailedDelivery:       time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					LastFailedDeliveryReason: "abc",
				}
				return sub
			}(),
			givenIsSubscribed: true,
			wantCondition: eventingv1alpha2.Condition{
				Type:               eventingv1alpha2.ConditionWebhookCallStatus,
				LastTransitionTime: currentTime,
				Status:             kcorev1.ConditionFalse,
				Reason:             eventingv1alpha2.ConditionReasonWebhookCallStatus,
				Message:            "abc",
			},
		},
		{
			name: "should replace condition with status true if lastDelivery was okay",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Ready = false

				// mark ConditionWebhookCallStatus conditions as false
				sub.Status.Conditions = []eventingv1alpha2.Condition{
					{
						Type:               eventingv1alpha2.ConditionSubscribed,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionFalse,
					},
					{
						Type:               eventingv1alpha2.ConditionWebhookCallStatus,
						LastTransitionTime: currentTime,
						Status:             kcorev1.ConditionFalse,
					},
				}
				// set EventMeshSubscriptionStatus
				// LastSuccessfulDelivery is latest
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					LastFailedDelivery:       time.Now().Format(time.RFC3339),
					LastFailedDeliveryReason: "",
				}
				return sub
			}(),
			givenIsSubscribed: true,
			wantCondition: eventingv1alpha2.Condition{
				Type:               eventingv1alpha2.ConditionWebhookCallStatus,
				LastTransitionTime: currentTime,
				Status:             kcorev1.ConditionTrue,
				Reason:             eventingv1alpha2.ConditionReasonWebhookCallStatus,
				Message:            "",
			},
		},
	}

	logger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	if err != nil {
		t.Fatalf(`failed to initiate logger, %v`, err)
	}

	reconciler := Reconciler{
		logger:                         logger,
		syncConditionWebhookCallStatus: syncConditionWebhookCallStatus,
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			sub := testcase.givenSubscription

			// when
			reconciler.syncConditionWebhookCallStatus(sub)

			// then
			newCondition := sub.Status.FindCondition(testcase.wantCondition.Type)
			assert.NotNil(t, newCondition)
			assert.True(t, eventingv1alpha2.ConditionEquals(*newCondition, testcase.wantCondition))
		})
	}
}

func Test_checkStatusActive(t *testing.T) {
	currentTime := time.Now()
	testCases := []struct {
		name         string
		subscription *eventingv1alpha2.Subscription
		wantStatus   bool
		wantError    error
	}{
		{
			name: "should return active since the EventMeshSubscriptionStatus is active",
			subscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					Status: string(types.SubscriptionStatusActive),
				}
				return sub
			}(),
			wantStatus: true,
			wantError:  nil,
		},
		{
			name: "should return active if the EventMeshSubscriptionStatus is active and the FailedActivation time is set",
			subscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Backend.FailedActivation = currentTime.Format(time.RFC3339)
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					Status: string(types.SubscriptionStatusActive),
				}
				return sub
			}(),
			wantStatus: true,
			wantError:  nil,
		},
		{
			name: "should return not active if the EventMeshSubscriptionStatus is inactive",
			subscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					Status: string(types.SubscriptionStatusPaused),
				}
				return sub
			}(),
			wantStatus: false,
			wantError:  nil,
		},
		{
			name: `should return not active if the EventMeshSubscriptionStatus is inactive and the FailedActivation time is set`,
			subscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Backend.FailedActivation = currentTime.Format(time.RFC3339)
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					Status: string(types.SubscriptionStatusPaused),
				}
				return sub
			}(),
			wantStatus: false,
			wantError:  nil,
		},
		{
			name: "should error if timed out waiting after retrying",
			subscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				sub.Status.InitializeConditions()
				sub.Status.Backend.FailedActivation = currentTime.Add(time.Minute * -1).Format(time.RFC3339)
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					Status: string(types.SubscriptionStatusPaused),
				}
				return sub
			}(),
			wantStatus: false,
			wantError:  errors.New("timeout waiting for the subscription to be active: some-name"),
		},
	}

	r := Reconciler{}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			gotStatus, err := r.checkStatusActive(testcase.subscription)
			require.Equal(t, testcase.wantStatus, gotStatus)
			if testcase.wantError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, testcase.wantError.Error())
			}
		})
	}
}

func Test_checkLastFailedDelivery(t *testing.T) {
	testCases := []struct {
		name              string
		givenSubscription *eventingv1alpha2.Subscription
		wantResult        bool
		wantError         error
	}{
		{
			name: "should return false if there is no lastFailedDelivery",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				// set EventMeshSubscriptionStatus
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   "",
					LastFailedDelivery:       "",
					LastFailedDeliveryReason: "",
				}
				return sub
			}(),
			wantResult: false,
			wantError:  nil,
		},
		{
			name: "should return error if LastFailedDelivery is invalid",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				// set EventMeshSubscriptionStatus
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   "",
					LastFailedDelivery:       "invalid",
					LastFailedDeliveryReason: "",
				}
				return sub
			}(),
			wantResult: true,
			wantError:  errors.New(`failed to parse LastFailedDelivery: parsing time "invalid" as "2006-01-02T15:04:05Z07:00": cannot parse "invalid" as "2006"`),
		},
		{
			name: "should return error if LastFailedDelivery is valid but LastSuccessfulDelivery is invalid",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				// set EventMeshSubscriptionStatus
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   "invalid",
					LastFailedDelivery:       time.Now().Format(time.RFC3339),
					LastFailedDeliveryReason: "",
				}
				return sub
			}(),
			wantResult: true,
			wantError:  errors.New(`failed to parse LastSuccessfulDelivery: parsing time "invalid" as "2006-01-02T15:04:05Z07:00": cannot parse "invalid" as "2006"`),
		},
		{
			name: "should return true if last delivery of event was failed",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				// set EventMeshSubscriptionStatus
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   time.Now().Format(time.RFC3339),
					LastFailedDelivery:       time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					LastFailedDeliveryReason: "",
				}
				return sub
			}(),
			wantResult: true,
			wantError:  nil,
		},
		{
			name: "should return false if last delivery of event was success",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace")
				// set EventMeshSubscriptionStatus
				sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
					LastSuccessfulDelivery:   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					LastFailedDelivery:       time.Now().Format(time.RFC3339),
					LastFailedDeliveryReason: "",
				}
				return sub
			}(),
			wantResult: false,
			wantError:  nil,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			result, err := checkLastFailedDelivery(testcase.givenSubscription)
			require.Equal(t, testcase.wantResult, result)
			if testcase.wantError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, testcase.wantError.Error())
			}
		})
	}
}

func Test_validateSubscription(t *testing.T) {
	// given
	subscription := eventingtesting.NewSubscription("test-subscription", "test",
		eventingtesting.WithConditions(eventingv1alpha2.MakeSubscriptionConditions()),
		eventingtesting.WithFinalizers([]string{eventingv1alpha2.Finalizer}),
		eventingtesting.WithDefaultSource(),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedEventType),
		eventingtesting.WithValidSink("test", "test-svc"),
		eventingtesting.WithEmsSubscriptionStatus(string(types.SubscriptionStatusActive)),
		eventingtesting.WithMaxInFlight(10),
	)

	// Set up the test environment.
	testEnv := setupTestEnvironment(t, subscription)
	testEnv.backend.On("Initialize", mock.Anything).Return(nil)
	testEnv.backend.On("SyncSubscription", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

	// Set up the SubscriptionValidator mock.
	validatorMock := &subscriptionvalidatormocks.SubscriptionValidator{}
	validatorMock.On("Validate", mock.Anything, mock.Anything).Return(nil)

	reconciler := NewReconciler(
		testEnv.fakeClient,
		testEnv.logger,
		testEnv.recorder,
		testEnv.cfg,
		testEnv.cleaner,
		testEnv.backend,
		testEnv.credentials,
		testEnv.mapper,
		validatorMock,
		metrics.NewCollector(),
		utils.Domain,
	)

	// when
	request := kctrl.Request{NamespacedName: ktypes.NamespacedName{Namespace: subscription.Namespace, Name: subscription.Name}}
	res, err := reconciler.Reconcile(context.Background(), request)

	// then
	require.Equal(t, kctrl.Result{}, res)
	require.NoError(t, err)
	validatorMock.AssertExpectations(t)
	testEnv.backend.AssertExpectations(t)
}

// helper functions and structs

// testEnvironment provides mocked resources for tests.
type testEnvironment struct {
	fakeClient  client.Client
	backend     *mocks.Backend
	logger      *logger.Logger
	recorder    *record.FakeRecorder
	cfg         env.Config
	credentials *eventmesh.OAuth2ClientCredentials
	mapper      backendutils.NameMapper
	cleaner     cleaner.Cleaner
}

// setupTestEnvironment is a testEnvironment constructor.
func setupTestEnvironment(t *testing.T, objs ...client.Object) *testEnvironment {
	t.Helper()
	mockedBackend := &mocks.Backend{}
	fakeClient := createFakeClientBuilder(t).WithObjects(objs...).WithStatusSubresource(objs...).Build()
	recorder := &record.FakeRecorder{}
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	if err != nil {
		t.Fatalf("initialize logger failed: %v", err)
	}
	emptyConfig := env.Config{}
	credentials := &eventmesh.OAuth2ClientCredentials{}
	nameMapper := backendutils.NewBEBSubscriptionNameMapper(utils.Domain, eventmesh.MaxSubscriptionNameLength)
	eventMeshCleaner := cleaner.NewEventMeshCleaner(nil)

	return &testEnvironment{
		fakeClient:  fakeClient,
		backend:     mockedBackend,
		logger:      defaultLogger,
		recorder:    recorder,
		cfg:         emptyConfig,
		credentials: credentials,
		mapper:      nameMapper,
		cleaner:     eventMeshCleaner,
	}
}

func createFakeClientBuilder(t *testing.T) *fake.ClientBuilder {
	t.Helper()
	err := eventingv1alpha2.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = apigatewayv2.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = kcorev1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	return fake.NewClientBuilder().WithScheme(scheme.Scheme)
}
