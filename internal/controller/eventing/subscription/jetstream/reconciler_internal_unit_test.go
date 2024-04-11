package jetstream

import (
	"context"
	"testing"

	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	kctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	"github.com/kyma-project/eventing-manager/pkg/backend/jetstream"
	backendjetstreammocks "github.com/kyma-project/eventing-manager/pkg/backend/jetstream/mocks"
	"github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	"github.com/kyma-project/eventing-manager/pkg/backend/sink"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

const (
	subscriptionName = "testSubscription"
	namespaceName    = "test"
)

// Test_Reconcile tests the return values of the Reconcile() method of the reconciler.
// This is important, as it dictates whether the reconciliation should be requeued by Controller Runtime,
// and if so with how much initial delay.
// Returning error or a `Result{Requeue: true}` would cause the reconciliation to be requeued.
// Everything else is mocked since we are only interested in the logic of the Reconcile method
// and not the reconciler dependencies.
func Test_Reconcile(t *testing.T) {
	req := require.New(t)

	// A subscription with the correct Finalizer, ready for reconciliation with the backend.
	testSub := eventingtesting.NewSubscription("sub1", namespaceName,
		eventingtesting.WithFinalizers([]string{eventingv1alpha2.Finalizer}),
		eventingtesting.WithSource(eventingtesting.EventSourceClean),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
		eventingtesting.WithMaxInFlight(10),
		eventingtesting.WithSink("http://test.test.svc.cluster.local"),
	)
	// A subscription marked for deletion.
	testSubUnderDeletion := eventingtesting.NewSubscription("sub2", namespaceName,
		eventingtesting.WithNonZeroDeletionTimestamp(),
		eventingtesting.WithFinalizers([]string{eventingv1alpha2.Finalizer}),
		eventingtesting.WithSource(eventingtesting.EventSourceClean),
		eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
	)

	backendSyncErr := errors.New("backend sync error")
	missingSubSyncErr := jetstream.ErrMissingSubscription
	backendDeleteErr := errors.New("backend delete error")
	validatorErr := errors.New("invalid sink")
	happyValidator := sink.ValidatorFunc(func(_ context.Context, s *eventingv1alpha2.Subscription) error { return nil })
	unhappyValidator := sink.ValidatorFunc(func(_ context.Context, s *eventingv1alpha2.Subscription) error { return validatorErr })
	collector := metrics.NewCollector()

	testCases := []struct {
		name                 string
		givenSubscription    *eventingv1alpha2.Subscription
		givenReconcilerSetup func() (*Reconciler, *backendjetstreammocks.Backend)
		wantReconcileResult  kctrl.Result
		wantReconcileError   error
	}{
		{
			name:              "Return nil and default Result{} when there is no error from the reconciler dependencies",
			givenSubscription: testSub,
			givenReconcilerSetup: func() (*Reconciler, *backendjetstreammocks.Backend) {
				testenv := setupTestEnvironment(t, testSub)
				testenv.Backend.On("SyncSubscription", mock.Anything).Return(nil)
				testenv.Backend.On("GetJetStreamSubjects", mock.Anything, mock.Anything, mock.Anything).Return(
					[]string{eventingtesting.JetStreamSubject})
				testenv.Backend.On("GetConfig", mock.Anything).Return(env.NATSConfig{JSStreamName: "sap"})
				return NewReconciler(
						testenv.Client,
						testenv.Backend,
						testenv.Logger,
						testenv.Recorder,
						testenv.Cleaner,
						happyValidator,
						collector),
					testenv.Backend
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  nil,
		},
		{
			name:              "Return nil and default Result{} when the subscription does not exist on the cluster",
			givenSubscription: testSub,
			givenReconcilerSetup: func() (*Reconciler, *backendjetstreammocks.Backend) {
				testenv := setupTestEnvironment(t)
				return NewReconciler(
						testenv.Client,
						testenv.Backend,
						testenv.Logger,
						testenv.Recorder,
						testenv.Cleaner,
						happyValidator,
						collector),
					testenv.Backend
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  nil,
		},
		{
			name:              "Return nil and default Result{} when the subscription has no finalizer",
			givenSubscription: eventingtesting.NewSubscription(subscriptionName, namespaceName),
			givenReconcilerSetup: func() (*Reconciler, *backendjetstreammocks.Backend) {
				testenv := setupTestEnvironment(t, eventingtesting.NewSubscription(subscriptionName, namespaceName))
				return NewReconciler(
						testenv.Client,
						testenv.Backend,
						testenv.Logger,
						testenv.Recorder,
						testenv.Cleaner,
						happyValidator,
						collector),
					testenv.Backend
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  nil,
		},
		{
			name:              "Return error and default Result{} when backend sync returns error",
			givenSubscription: testSub,
			givenReconcilerSetup: func() (*Reconciler, *backendjetstreammocks.Backend) {
				testenv := setupTestEnvironment(t, testSub)
				testenv.Backend.On("SyncSubscription", mock.Anything).Return(backendSyncErr)
				testenv.Backend.On("GetJetStreamSubjects", mock.Anything, mock.Anything, mock.Anything).Return(
					[]string{eventingtesting.JetStreamSubject})
				testenv.Backend.On("GetConfig", mock.Anything).Return(env.NATSConfig{JSStreamName: "sap"})
				return NewReconciler(
						testenv.Client,
						testenv.Backend,
						testenv.Logger,
						testenv.Recorder,
						testenv.Cleaner,
						happyValidator,
						collector),
					testenv.Backend
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  backendSyncErr,
		},
		{
			name: "Return nil and RequeueAfter with requeue duration when " +
				"backend sync returns missingSubscriptionErr",
			givenSubscription: testSub,
			givenReconcilerSetup: func() (*Reconciler, *backendjetstreammocks.Backend) {
				testenv := setupTestEnvironment(t, testSub)
				testenv.Backend.On("SyncSubscription", mock.Anything).Return(missingSubSyncErr)
				testenv.Backend.On("GetJetStreamSubjects", mock.Anything, mock.Anything, mock.Anything).Return(
					[]string{eventingtesting.JetStreamSubject})
				testenv.Backend.On("GetConfig", mock.Anything).Return(env.NATSConfig{JSStreamName: "sap"})
				return NewReconciler(
						testenv.Client,
						testenv.Backend,
						testenv.Logger,
						testenv.Recorder,
						testenv.Cleaner,
						happyValidator,
						collector),
					testenv.Backend
			},
			wantReconcileResult: kctrl.Result{RequeueAfter: requeueDuration},
			wantReconcileError:  nil,
		},
		{
			name:              "Return error and default Result{} when backend delete returns error",
			givenSubscription: testSubUnderDeletion,
			givenReconcilerSetup: func() (*Reconciler, *backendjetstreammocks.Backend) {
				testenv := setupTestEnvironment(t, testSubUnderDeletion)
				testenv.Backend.On("DeleteSubscription", mock.Anything).Return(backendDeleteErr)
				return NewReconciler(
						testenv.Client,
						testenv.Backend,
						testenv.Logger,
						testenv.Recorder,
						testenv.Cleaner,
						happyValidator,
						collector),
					testenv.Backend
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  errFailedToDeleteSub,
		},
		{
			name:              "Return error and default Result{} when validator returns error",
			givenSubscription: testSub,
			givenReconcilerSetup: func() (*Reconciler, *backendjetstreammocks.Backend) {
				testenv := setupTestEnvironment(t, testSub)
				testenv.Backend.On("DeleteSubscriptionsOnly", mock.Anything).Return(nil)
				testenv.Backend.On("GetJetStreamSubjects", mock.Anything, mock.Anything, mock.Anything).Return(
					[]string{eventingtesting.JetStreamSubject})
				testenv.Backend.On("GetConfig", mock.Anything).Return(env.NATSConfig{JSStreamName: "sap"})
				return NewReconciler(
						testenv.Client,
						testenv.Backend,
						testenv.Logger,
						testenv.Recorder,
						testenv.Cleaner,
						unhappyValidator,
						collector),
					testenv.Backend
			},
			wantReconcileResult: kctrl.Result{},
			wantReconcileError:  reconcile.TerminalError(validatorErr),
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			reconciler, mockedBackend := testcase.givenReconcilerSetup()
			request := kctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: testcase.givenSubscription.Namespace,
				Name:      testcase.givenSubscription.Name,
			}}

			// when
			res, err := reconciler.Reconcile(context.Background(), request)

			// then
			req.Equal(testcase.wantReconcileResult, res)
			req.ErrorIs(err, testcase.wantReconcileError)
			if testcase.wantReconcileError == nil {
				mockedBackend.AssertExpectations(t)
			}
		})
	}
}

func Test_handleSubscriptionDeletion(t *testing.T) {
	testCases := []struct {
		name            string
		givenFinalizers []string
		wantDeleteCall  bool
		wantFinalizers  []string
		wantError       error
	}{
		{
			name:            "With no finalizers the NATS subscription should not be deleted",
			givenFinalizers: []string{},
			wantDeleteCall:  false,
			wantFinalizers:  []string{},
			wantError:       nil,
		},
		{
			name: "With eventing finalizer the NATS subscription should be deleted" +
				" and the finalizer should be cleared",
			givenFinalizers: []string{eventingv1alpha2.Finalizer},
			wantDeleteCall:  true,
			wantFinalizers:  []string{},
			wantError:       nil,
		},
		{
			name:            "With wrong finalizer the NATS subscription should not be deleted",
			givenFinalizers: []string{"eventing2.kyma-project.io"},
			wantDeleteCall:  false,
			wantFinalizers:  []string{"eventing2.kyma-project.io"},
			wantError:       nil,
		},
		{
			name:            "With Delete Subscription returning an error, the finalizer still remains",
			givenFinalizers: []string{eventingv1alpha2.Finalizer},
			wantDeleteCall:  true,
			wantFinalizers:  []string{eventingv1alpha2.Finalizer},
			wantError:       errFailedToDeleteSub,
		},
	}

	for _, tC := range testCases {
		testCase := tC
		t.Run(testCase.name, func(t *testing.T) {
			// given
			sub := eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithFinalizers(testCase.givenFinalizers),
			)

			testEnvironment := setupTestEnvironment(t, sub)
			ctx, reconciler, mockedBackend := context.Background(), testEnvironment.Reconciler, testEnvironment.Backend

			if testCase.wantDeleteCall {
				if errors.Is(testCase.wantError, errFailedToDeleteSub) {
					mockedBackend.On("DeleteSubscription", sub).Return(errors.New("deletion error"))
				} else {
					mockedBackend.On("DeleteSubscription", sub).Return(nil)
				}
			}

			// when
			_, err := reconciler.handleSubscriptionDeletion(ctx, sub, reconciler.namedLogger())
			require.ErrorIs(t, err, testCase.wantError)

			// then
			mockedBackend.AssertExpectations(t)

			ensureFinalizerMatch(t, sub, testCase.wantFinalizers)

			// check the changes were made on the kubernetes server
			fetchedSub, err := fetchTestSubscription(ctx, reconciler)
			require.NoError(t, err)
			ensureFinalizerMatch(t, &fetchedSub, testCase.wantFinalizers)

			// clean up finalizers first before deleting sub
			fetchedSub.ObjectMeta.Finalizers = nil
			err = reconciler.Client.Update(ctx, &fetchedSub)
			require.NoError(t, err)
			err = reconciler.Client.Delete(ctx, &fetchedSub)
			require.NoError(t, err)
		})
	}
}

func Test_addFinalizer(t *testing.T) {
	// given
	ctx := context.Background()
	eventingFinalizer := []string{eventingv1alpha2.Finalizer}
	var emptyFinalizer []string
	sub := eventingtesting.NewSubscription(subscriptionName, namespaceName)
	testEnvironment := setupTestEnvironment(t, sub)
	reconciler := testEnvironment.Reconciler
	fakeSub := eventingtesting.NewSubscription("fake", namespaceName)

	testCases := []struct {
		name             string
		givenFinalizers  []string
		wantErrorMessage string
	}{
		{
			name:            "A Subscription should be updated with the eventing finalizer if it is missing",
			givenFinalizers: emptyFinalizer,
		},
		{
			name:            "addFinalizer() should propagate the error returned by the update operation",
			givenFinalizers: eventingFinalizer,
			wantErrorMessage: kerrors.NewNotFound(
				schema.GroupResource{
					Group:    "eventing.kyma-project.io",
					Resource: "subscriptions",
				},
				fakeSub.Name).Error(),
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// in case syncSubscriptionStatus throws an error
			if testcase.wantErrorMessage != "" {
				_, err := reconciler.addFinalizer(ctx, fakeSub)
				require.ErrorContains(t, err, testcase.wantErrorMessage)
				return
			}
			sub.Finalizers = testcase.givenFinalizers
			_, err := reconciler.addFinalizer(ctx, sub)
			require.NoError(t, err)

			fetchedSub, err := fetchTestSubscription(context.Background(), reconciler)
			require.NoError(t, err)

			if testcase.wantErrorMessage != "" {
				ensureFinalizerMatch(t, &fetchedSub, emptyFinalizer)
			} else {
				ensureFinalizerMatch(t, &fetchedSub, eventingFinalizer)
			}
		})
	}
}

func Test_syncSubscriptionStatus(t *testing.T) {
	jetStreamError := errors.New("JetStream is not ready")
	falseNatsSubActiveCondition := eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscriptionActive,
		eventingv1alpha2.ConditionReasonNATSSubscriptionNotActive,
		kcorev1.ConditionFalse, jetStreamError.Error())
	trueNatsSubActiveCondition := eventingv1alpha2.MakeCondition(eventingv1alpha2.ConditionSubscriptionActive,
		eventingv1alpha2.ConditionReasonNATSSubscriptionActive,
		kcorev1.ConditionTrue, "")

	testCases := []struct {
		name           string
		givenSub       *eventingv1alpha2.Subscription
		givenError     error
		wantConditions []eventingv1alpha2.Condition
		wantStatus     bool
	}{
		{
			name: "Subscription with no conditions should stay not ready with false condition",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithConditions([]eventingv1alpha2.Condition{}),
				eventingtesting.WithStatus(true),
			),
			givenError:     jetStreamError,
			wantConditions: []eventingv1alpha2.Condition{falseNatsSubActiveCondition},
			wantStatus:     false,
		},
		{
			name: "Subscription with false ready condition should stay not ready with false condition",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithConditions([]eventingv1alpha2.Condition{falseNatsSubActiveCondition}),
				eventingtesting.WithStatus(false),
			),
			givenError:     jetStreamError,
			wantConditions: []eventingv1alpha2.Condition{falseNatsSubActiveCondition},
			wantStatus:     false,
		},
		{
			name: "Subscription should become ready because because, there is no reconciliation error",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithConditions([]eventingv1alpha2.Condition{falseNatsSubActiveCondition}),
				eventingtesting.WithStatus(false),
			),
			givenError:     nil,
			wantConditions: []eventingv1alpha2.Condition{trueNatsSubActiveCondition},
			wantStatus:     true,
		},
		{
			name: "Subscription should stay with ready condition and status",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithConditions([]eventingv1alpha2.Condition{trueNatsSubActiveCondition}),
				eventingtesting.WithStatus(true),
			),
			givenError:     nil,
			wantConditions: []eventingv1alpha2.Condition{trueNatsSubActiveCondition},
			wantStatus:     true,
		},
		{
			name: "Subscription should become not ready because of the occurred error",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithConditions([]eventingv1alpha2.Condition{trueNatsSubActiveCondition}),
				eventingtesting.WithStatus(true),
			),
			givenError:     jetStreamError,
			wantConditions: []eventingv1alpha2.Condition{falseNatsSubActiveCondition},
			wantStatus:     false,
		},
	}
	for _, tC := range testCases {
		testCase := tC
		t.Run(testCase.name, func(t *testing.T) {
			// given
			sub := testCase.givenSub

			testEnvironment := setupTestEnvironment(t, sub)
			ctx, reconciler := context.Background(), testEnvironment.Reconciler

			// when
			err := reconciler.syncSubscriptionStatus(ctx, sub, testCase.givenError, reconciler.namedLogger())
			require.NoError(t, err)

			// then
			ensureSubscriptionMatchesConditionsAndStatus(t, *sub, testCase.wantConditions, testCase.wantStatus)

			// fetch the sub also from k8s server in order to check whether changes were done both in-memory and on k8s server
			fetchedSub, err := fetchTestSubscription(ctx, reconciler)
			require.NoError(t, err)
			ensureSubscriptionMatchesConditionsAndStatus(t, fetchedSub, testCase.wantConditions, testCase.wantStatus)

			// clean up
			err = reconciler.Client.Delete(ctx, sub)
			require.NoError(t, err)
		})
	}
}

func Test_syncEventTypes(t *testing.T) {
	testEnvironment := setupTestEnvironment(t)
	reconciler := testEnvironment.Reconciler

	jsSubjects := []string{eventingtesting.JetStreamSubjectV2}
	eventTypes := []eventingv1alpha2.EventType{
		{
			OriginalType: eventingtesting.OrderCreatedUncleanEvent,
			CleanType:    eventingtesting.OrderCreatedCleanEvent,
		},
	}
	jsTypes := []eventingv1alpha2.JetStreamTypes{
		{
			OriginalType: eventingtesting.OrderCreatedUncleanEvent,
			ConsumerName: "a59e97ceb4883938c193bc0abf6e8bca",
		},
	}
	backendStatus := eventingv1alpha2.Backend{
		Types: jsTypes,
	}
	testEnvironment.Backend.On("GetJetStreamSubjects", mock.Anything, mock.Anything, mock.Anything).Return(jsSubjects)

	testCases := []struct {
		name          string
		givenSub      *eventingv1alpha2.Subscription
		wantSubStatus eventingv1alpha2.SubscriptionStatus
	}{
		{
			name: "A new Subscription must be updated with cleanEventTypes and backend jstypes and return true",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithStatus(false),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedUncleanEvent),
			),
			wantSubStatus: eventingv1alpha2.SubscriptionStatus{
				Ready:   false,
				Types:   eventTypes,
				Backend: backendStatus,
			},
		},
		{
			name: "A subscription with the same cleanEventTypes and jsTypes must return false",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithStatus(true),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedUncleanEvent),
				eventingtesting.WithStatusTypes(eventTypes),
				eventingtesting.WithStatusJSBackendTypes(jsTypes),
			),
			wantSubStatus: eventingv1alpha2.SubscriptionStatus{
				Ready:   true,
				Types:   eventTypes,
				Backend: backendStatus,
			},
		},
		{
			name: "A subscription with the same eventTypes and new jsTypes must return true",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithStatus(true),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedUncleanEvent),
				eventingtesting.WithStatusTypes(eventTypes),
			),
			wantSubStatus: eventingv1alpha2.SubscriptionStatus{
				Ready:   true,
				Types:   eventTypes,
				Backend: backendStatus,
			},
		},
		{
			name: "A subscription with changed eventTypes and the same jsTypes must return true",
			givenSub: eventingtesting.NewSubscription(subscriptionName, namespaceName,
				eventingtesting.WithStatus(true),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedUncleanEvent),
				eventingtesting.WithStatusJSBackendTypes(jsTypes),
			),
			wantSubStatus: eventingv1alpha2.SubscriptionStatus{
				Ready:   true,
				Types:   eventTypes,
				Backend: backendStatus,
			},
		},
	}
	for _, tC := range testCases {
		testCase := tC
		t.Run(testCase.name, func(t *testing.T) {
			// given
			sub := testCase.givenSub

			// when
			require.NoError(t, reconciler.syncEventTypes(sub))

			// then
			require.Equal(t, testCase.wantSubStatus.Types, sub.Status.Types)
			require.Equal(t, testCase.wantSubStatus.Backend, sub.Status.Backend)
			require.Equal(t, testCase.wantSubStatus.Ready, sub.Status.Ready)
		})
	}
}

func Test_updateStatus(t *testing.T) {
	sub := eventingtesting.NewSubscription(subscriptionName, namespaceName, eventingtesting.WithStatus(true))

	ctx := context.Background()
	testEnvironment := setupTestEnvironment(t, sub)
	reconciler := testEnvironment.Reconciler

	testCases := []struct {
		name       string
		oldStatus  eventingv1alpha2.SubscriptionStatus
		newStatus  eventingv1alpha2.SubscriptionStatus
		wantError  error
		wantChange bool
	}{
		{
			name:       "No update should happen in case when the statuses are equal",
			oldStatus:  eventingv1alpha2.SubscriptionStatus{Ready: true},
			newStatus:  eventingv1alpha2.SubscriptionStatus{Ready: true},
			wantChange: false,
		},
		{
			name:       "An Update should happen in case when the statuses are equal",
			oldStatus:  eventingv1alpha2.SubscriptionStatus{Ready: true},
			newStatus:  eventingv1alpha2.SubscriptionStatus{Ready: false},
			wantChange: true,
		},
		{
			name:       "The error message from the update operation should fail the whole function",
			oldStatus:  eventingv1alpha2.SubscriptionStatus{Ready: true},
			newStatus:  eventingv1alpha2.SubscriptionStatus{Ready: false},
			wantChange: true,
			wantError:  errFailedToUpdateStatus,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			resourceVersionBefore := sub.ResourceVersion
			sub.Status = testcase.oldStatus
			newSub := sub.DeepCopy()
			newSub.Status = testcase.newStatus

			// simulate the update error
			if testcase.wantError != nil {
				require.NoError(t, reconciler.Client.Delete(ctx, sub))
			}

			// when
			updateStatusErr := reconciler.updateStatus(ctx, sub, newSub, reconciler.namedLogger())

			// then
			// in case the function should fail
			if testcase.wantError != nil {
				require.ErrorIs(t, updateStatusErr, testcase.wantError)
				return
			}

			// then
			require.NoError(t, updateStatusErr)
			fetchedSub, err := fetchTestSubscription(context.Background(), reconciler)
			require.NoError(t, err)

			require.Equal(t, testcase.wantChange, fetchedSub.ResourceVersion != resourceVersionBefore)
			require.Equal(t, newSub.Status.Ready, fetchedSub.Status.Ready)
			require.Equal(t, newSub.Status.Conditions, fetchedSub.Status.Conditions)
		})
	}
}

func Test_updateSubscription(t *testing.T) {
	sub := eventingtesting.NewSubscription(subscriptionName, namespaceName,
		eventingtesting.WithStatusTypes([]eventingv1alpha2.EventType{
			{OriginalType: "order.created.v1", CleanType: "order.created.v1"},
		}),
	)
	subWithDifferentStatus := sub.DeepCopy()
	subWithDifferentStatus.Status.Ready = !sub.Status.Ready

	ctx := context.Background()
	testCases := []struct {
		name             string
		actualSub        *eventingv1alpha2.Subscription
		desiredSub       *eventingv1alpha2.Subscription
		wantErrorMessage string
		wantChange       bool
		wantSubStatus    *eventingv1alpha2.SubscriptionStatus
	}{
		{
			name:       "When Subscription cannot be fetched from the api server the function should return an error",
			actualSub:  sub,
			desiredSub: sub,
			wantErrorMessage: kerrors.NewNotFound(
				eventingv1alpha2.SubscriptionGroupVersionResource().GroupResource(),
				subscriptionName).Error(),
		},
		{
			name:       "updateSubscriptionStatus() should be idempotent",
			actualSub:  sub,
			desiredSub: sub,
			wantChange: false,
		},
		{
			name:       "Subscription's status should be updated on difference",
			actualSub:  sub,
			desiredSub: subWithDifferentStatus,
			wantChange: true,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			testEnvironment := setupTestEnvironment(t)
			if testcase.wantErrorMessage == "" {
				sub.ResourceVersion = ""
				testEnvironment = setupTestEnvironment(t, sub)
			}
			reconciler := testEnvironment.Reconciler

			resourceVersionBefore := sub.ResourceVersion

			// when
			err := reconciler.updateSubscriptionStatus(ctx, testcase.desiredSub, reconciler.namedLogger())

			// then
			// in case the subscription doesn't exist
			if testcase.wantErrorMessage != "" {
				require.ErrorContains(t, err, testcase.wantErrorMessage)
				return
			}

			// then
			fetchedSub, err := fetchTestSubscription(context.Background(), reconciler)
			require.NoError(t, err)

			require.Equal(t, testcase.wantChange, fetchedSub.ResourceVersion != resourceVersionBefore)
			ensureFinalizerMatch(t, &fetchedSub, testcase.desiredSub.Finalizers)
			require.Equal(t, testcase.desiredSub.Status.Ready, fetchedSub.Status.Ready)
			require.Equal(t, testcase.desiredSub.Status.Conditions, fetchedSub.Status.Conditions)

			// clean up
			require.NoError(t, reconciler.Client.Delete(ctx, sub))
		})
	}
}

// helper functions and structs

// TestEnvironment provides mocked resources for tests.
type TestEnvironment struct {
	Client     client.Client
	Backend    *backendjetstreammocks.Backend
	Reconciler *Reconciler
	Logger     *logger.Logger
	Recorder   *record.FakeRecorder
	Cleaner    cleaner.Cleaner
}

// setupTestEnvironment is a TestEnvironment constructor.
func setupTestEnvironment(t *testing.T, objs ...client.Object) *TestEnvironment {
	t.Helper()
	mockedBackend := &backendjetstreammocks.Backend{}
	fakeClient := createFakeClientBuilder(t).WithObjects(objs...).WithStatusSubresource(objs...).Build()
	recorder := &record.FakeRecorder{}

	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	if err != nil {
		t.Fatalf("initialize logger failed: %v", err)
	}
	jsCleaner := cleaner.NewJetStreamCleaner(defaultLogger)
	defaultSinkValidator := sink.NewValidator(fakeClient, recorder)

	reconciler := Reconciler{
		Backend:       mockedBackend,
		Client:        fakeClient,
		logger:        defaultLogger,
		recorder:      recorder,
		sinkValidator: defaultSinkValidator,
		cleaner:       jsCleaner,
		collector:     metrics.NewCollector(),
	}

	return &TestEnvironment{
		Client:     fakeClient,
		Backend:    mockedBackend,
		Reconciler: &reconciler,
		Logger:     defaultLogger,
		Recorder:   recorder,
		Cleaner:    jsCleaner,
	}
}

func createFakeClientBuilder(t *testing.T) *fake.ClientBuilder {
	t.Helper()
	err := eventingv1alpha2.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	return fake.NewClientBuilder().WithScheme(scheme.Scheme)
}

func ensureFinalizerMatch(t *testing.T, subscription *eventingv1alpha2.Subscription, wantFinalizers []string) {
	t.Helper()
	if len(wantFinalizers) == 0 {
		require.Empty(t, subscription.ObjectMeta.Finalizers)
	} else {
		require.Equal(t, wantFinalizers, subscription.ObjectMeta.Finalizers)
	}
}

func fetchTestSubscription(ctx context.Context, r *Reconciler) (eventingv1alpha2.Subscription, error) {
	var fetchedSub eventingv1alpha2.Subscription
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      subscriptionName,
		Namespace: namespaceName,
	}, &fetchedSub)
	return fetchedSub, err
}

func ensureSubscriptionMatchesConditionsAndStatus(t *testing.T, subscription eventingv1alpha2.Subscription, wantConditions []eventingv1alpha2.Condition, wantStatus bool) {
	t.Helper()
	require.Equal(t, len(wantConditions), len(subscription.Status.Conditions))
	comparisonResult := eventingv1alpha2.ConditionsEquals(wantConditions, subscription.Status.Conditions)
	require.True(t, comparisonResult)
	require.Equal(t, wantStatus, subscription.Status.Ready)
}
