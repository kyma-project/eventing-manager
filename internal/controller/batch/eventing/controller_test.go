package eventing

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/kyma-project/eventing-manager/pkg/watcher"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/batch/v1alpha1"
	submanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager/mocks"
	watchmock "github.com/kyma-project/eventing-manager/pkg/watcher/mocks"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func Test_handleEventingCRAllowedCheck(t *testing.T) {
	t.Parallel()

	givenAllowedEventingCR := testutils.NewEventingCR(
		testutils.WithEventingCRName("eventing"),
		testutils.WithEventingCRNamespace("kyma-system"),
	)

	// define test cases
	testCases := []struct {
		name            string
		givenEventing   *eventingv1alpha1.Eventing
		wantCheckResult bool
	}{
		{
			name: "should allow Eventing CR if name and namespace is correct",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("eventing"),
				testutils.WithEventingCRNamespace("kyma-system"),
			),
			wantCheckResult: true,
		},
		{
			name: "should not allow Eventing CR if name is incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("not-allowed-name"),
				testutils.WithEventingCRNamespace("kyma-system"),
			),
			wantCheckResult: false,
		},
		{
			name: "should not allow Eventing CR if namespace is incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("eventing"),
				testutils.WithEventingCRNamespace("not-allowed-namespace"),
			),
			wantCheckResult: false,
		},
		{
			name: "should not allow Eventing CR if name and namespace, both are incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("not-allowed-name"),
				testutils.WithEventingCRNamespace("not-allowed-namespace"),
			),
			wantCheckResult: false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t, tc.givenEventing)
			testEnv.Reconciler.allowedEventingCR = givenAllowedEventingCR
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			// when
			result, err := testEnv.Reconciler.handleEventingCRAllowedCheck(testEnv.Context, tc.givenEventing, logger)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantCheckResult, result)

			// if the Eventing CR is not allowed then check if the CR status is correctly updated or not.
			gotEventing, err := testEnv.GetEventing(tc.givenEventing.Name, tc.givenEventing.Namespace)
			require.NoError(t, err)
			if !tc.wantCheckResult {
				// check eventing.status.state
				require.Equal(t, eventingv1alpha1.StateError, gotEventing.Status.State)

				// check eventing.status.conditions
				wantConditions := []metav1.Condition{
					{
						Type:               string(eventingv1alpha1.ConditionPublisherProxyReady),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(eventingv1alpha1.ConditionReasonForbidden),
						Message: fmt.Sprintf("Only a single Eventing CR with name: %s and namespace: %s "+
							"is allowed to be created in a Kyma cluster.", givenAllowedEventingCR.Name,
							givenAllowedEventingCR.Namespace),
					},
				}
				require.True(t, natsv1alpha1.ConditionsEquals(wantConditions, gotEventing.Status.Conditions))
			}
		})
	}
}

func Test_handleBackendSwitching(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                         string
		givenEventing                *eventingv1alpha1.Eventing
		givenNATSSubManagerMock      func() *submanagermocks.Manager
		givenEventMeshSubManagerMock func() *submanagermocks.Manager
		wantNATSStopped              bool
		wantEventMeshStopped         bool
		wantEventingState            string
		wantEventingConditionsLen    int
		wantError                    error
	}{
		{
			name: "it should do nothing because backend is not changed",
			givenEventing: testutils.NewEventingCR(
				testutils.WithNATSBackend(),
				testutils.WithStatusActiveBackend(eventingv1alpha1.NatsBackendType),
				testutils.WithStatusState(eventingv1alpha1.StateReady),
				testutils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submanagermocks.Manager {
				return new(submanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *submanagermocks.Manager {
				return new(submanagermocks.Manager)
			},
			wantError:                 nil,
			wantEventingState:         eventingv1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should stop NATS because backend is changed to EventMesh",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test"),
				testutils.WithStatusActiveBackend(eventingv1alpha1.NatsBackendType),
				testutils.WithStatusState(eventingv1alpha1.StateReady),
				testutils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submanagermocks.Manager {
				managerMock := new(submanagermocks.Manager)
				managerMock.On("Stop", true).Return(nil).Once()
				return managerMock
			},
			givenEventMeshSubManagerMock: func() *submanagermocks.Manager {
				return new(submanagermocks.Manager)
			},
			wantError:                 nil,
			wantEventingState:         eventingv1alpha1.StateProcessing,
			wantEventingConditionsLen: 0,
			wantNATSStopped:           true,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should return error because it failed to stop NATS backend",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test"),
				testutils.WithStatusActiveBackend(eventingv1alpha1.NatsBackendType),
				testutils.WithStatusState(eventingv1alpha1.StateReady),
				testutils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submanagermocks.Manager {
				managerMock := new(submanagermocks.Manager)
				managerMock.On("Stop", true).Return(errors.New("failed to stop")).Once()
				return managerMock
			},
			givenEventMeshSubManagerMock: func() *submanagermocks.Manager {
				return new(submanagermocks.Manager)
			},
			wantError:                 errors.New("failed to stop"),
			wantEventingState:         eventingv1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should stop EventMesh because backend is changed to NATS",
			givenEventing: testutils.NewEventingCR(
				testutils.WithNATSBackend(),
				testutils.WithStatusActiveBackend(eventingv1alpha1.EventMeshBackendType),
				testutils.WithStatusState(eventingv1alpha1.StateReady),
				testutils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submanagermocks.Manager {
				return new(submanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *submanagermocks.Manager {
				managerMock := new(submanagermocks.Manager)
				managerMock.On("Stop", true).Return(nil).Once()
				return managerMock
			},
			wantError:                 nil,
			wantEventingState:         eventingv1alpha1.StateProcessing,
			wantEventingConditionsLen: 0,
			wantNATSStopped:           false,
			wantEventMeshStopped:      true,
		},
		{
			name: "it should return error because it failed to stop EventMesh backend",
			givenEventing: testutils.NewEventingCR(
				testutils.WithNATSBackend(),
				testutils.WithStatusActiveBackend(eventingv1alpha1.EventMeshBackendType),
				testutils.WithStatusState(eventingv1alpha1.StateReady),
				testutils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submanagermocks.Manager {
				return new(submanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *submanagermocks.Manager {
				managerMock := new(submanagermocks.Manager)
				managerMock.On("Stop", true).Return(errors.New("failed to stop")).Once()
				return managerMock
			},
			wantError:                 errors.New("failed to stop"),
			wantEventingState:         eventingv1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)
			testEnv.Reconciler.isNATSSubManagerStarted = true
			testEnv.Reconciler.isEventMeshSubManagerStarted = true

			mockNatsWatcher := new(watchmock.Watcher)
			if tc.wantNATSStopped {
				mockNatsWatcher.On("Stop").Once()
			}
			testEnv.Reconciler.natsWatchers[tc.givenEventing.Namespace] = mockNatsWatcher

			// get mocks from test-case.
			givenNATSSubManagerMock := tc.givenNATSSubManagerMock()
			givenEventMeshSubManagerMock := tc.givenEventMeshSubManagerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.natsSubManager = givenNATSSubManagerMock
			testEnv.Reconciler.eventMeshSubManager = givenEventMeshSubManagerMock

			// when
			err := testEnv.Reconciler.handleBackendSwitching(tc.givenEventing, logger)

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.Equal(t, tc.wantError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			// check CR status.
			require.Equal(t, tc.wantEventingState, tc.givenEventing.Status.State)
			require.Len(t, tc.givenEventing.Status.Conditions, tc.wantEventingConditionsLen)

			// check assertions for mocks.
			givenNATSSubManagerMock.AssertExpectations(t)
			givenEventMeshSubManagerMock.AssertExpectations(t)

			// NATS
			if tc.wantNATSStopped {
				require.Nil(t, testEnv.Reconciler.natsSubManager)
				require.False(t, testEnv.Reconciler.isNATSSubManagerStarted)
			} else {
				require.NotNil(t, testEnv.Reconciler.natsSubManager)
				require.True(t, testEnv.Reconciler.isNATSSubManagerStarted)
			}

			// EventMesh
			if tc.wantEventMeshStopped {
				require.Nil(t, testEnv.Reconciler.eventMeshSubManager)
				require.False(t, testEnv.Reconciler.isEventMeshSubManagerStarted)
			} else {
				require.NotNil(t, testEnv.Reconciler.eventMeshSubManager)
				require.True(t, testEnv.Reconciler.isEventMeshSubManagerStarted)
			}
			mockNatsWatcher.AssertExpectations(t)
		})
	}
}

func Test_startNatsCRWatch(t *testing.T) {
	testCases := []struct {
		name         string
		watchStarted bool
		watchErr     error
	}{
		{
			name:         "NATS CR watch not started",
			watchStarted: false,
			watchErr:     nil,
		},
		{
			name:         "NATS CR watch already started",
			watchStarted: true,
			watchErr:     nil,
		},
		{
			name:         "NATS watcher error",
			watchStarted: false,
			watchErr:     errors.New("NATS watcher error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			testEnv.Reconciler.natsCRWatchStarted = tc.watchStarted

			// Create a fake Eventing CR
			eventing := testutils.NewEventingCR(
				testutils.WithEventingCRName("test-name"),
				testutils.WithEventingCRNamespace("test-namespace"),
			)

			// Create mock watcher and controller
			natsWatcher := new(watchmock.Watcher)
			natsWatcher.On("IsStarted").Return(tc.watchStarted)
			mockController := new(watchmock.Controller)
			if !tc.watchStarted {
				natsWatcher.On("Start").Once()
				natsWatcher.On("GetEventsChannel").Return(make(<-chan event.GenericEvent)).Once()

				mockController.On("Watch", mock.Anything, mock.Anything, mock.Anything).Return(tc.watchErr).Once()
			}
			testEnv.Reconciler.natsWatchers[eventing.Namespace] = natsWatcher
			testEnv.Reconciler.controller = mockController

			// when
			err := testEnv.Reconciler.startNATSCRWatch(eventing)

			// then
			require.Equal(t, tc.watchErr, err)
			require.NotNil(t, testEnv.Reconciler.natsWatchers[eventing.Namespace])
			if tc.watchErr != nil {
				require.False(t, testEnv.Reconciler.natsWatchers[eventing.Namespace].IsStarted())
			}
		})
	}
}

func Test_stopNatsCRWatch(t *testing.T) {
	testCases := []struct {
		name               string
		natsCRWatchStarted bool
		watchNatsWatcher   watcher.Watcher
	}{
		{
			name:               "NATS CR watch stopped",
			watchNatsWatcher:   nil,
			natsCRWatchStarted: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			testEnv.Reconciler.natsCRWatchStarted = tc.natsCRWatchStarted

			// Create a fake Watcher
			natsWatcher := new(watchmock.Watcher)
			natsWatcher.On("Stop").Times(1)

			eventing := testutils.NewEventingCR(
				testutils.WithEventingCRName("test-name"),
				testutils.WithEventingCRNamespace("test-namespace"),
			)

			testEnv.Reconciler.natsWatchers[eventing.Namespace] = natsWatcher

			testEnv.Reconciler.stopNATSCRWatch(eventing)

			// Check the results
			require.Equal(t, tc.watchNatsWatcher, nil)
			require.Equal(t, tc.natsCRWatchStarted, false)
		})
	}
}
