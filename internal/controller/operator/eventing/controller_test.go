package eventing

import (
	"context"
	"fmt"
	"testing"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	eventingcontrollermocks "github.com/kyma-project/eventing-manager/internal/controller/operator/eventing/mocks"
	eventingmocks "github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	submgrmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager/mocks"
	"github.com/kyma-project/eventing-manager/pkg/watcher"
	watchermocks "github.com/kyma-project/eventing-manager/pkg/watcher/mocks"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
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
		givenEventing   *operatorv1alpha1.Eventing
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t, testcase.givenEventing)
			testEnv.Reconciler.allowedEventingCR = givenAllowedEventingCR
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			// when
			result, err := testEnv.Reconciler.handleEventingCRAllowedCheck(context.Background(), testcase.givenEventing, logger)

			// then
			require.NoError(t, err)
			require.Equal(t, testcase.wantCheckResult, result)

			// if the Eventing CR is not allowed then check if the CR status is correctly updated or not.
			gotEventing, err := testEnv.GetEventing(testcase.givenEventing.Name, testcase.givenEventing.Namespace)
			require.NoError(t, err)
			if !testcase.wantCheckResult {
				// check eventing.status.state
				require.Equal(t, operatorv1alpha1.StateError, gotEventing.Status.State)

				// check eventing.status.conditions
				wantConditions := []kmetav1.Condition{
					{
						Type:               string(operatorv1alpha1.ConditionPublisherProxyReady),
						Status:             kmetav1.ConditionFalse,
						LastTransitionTime: kmetav1.Now(),
						Reason:             string(operatorv1alpha1.ConditionReasonForbidden),
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
		givenEventing                *operatorv1alpha1.Eventing
		givenNATSSubManagerMock      func() *submgrmanagermocks.Manager
		givenEventMeshSubManagerMock func() *submgrmanagermocks.Manager
		givenEventingManagerMock     func() *eventingmocks.Manager
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
				testutils.WithStatusActiveBackend(operatorv1alpha1.NatsBackendType),
				testutils.WithStatusState(operatorv1alpha1.StateReady),
				testutils.WithStatusConditions([]kmetav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				return new(submgrmanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				return new(submgrmanagermocks.Manager)
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				return new(eventingmocks.Manager)
			},
			wantError:                 nil,
			wantEventingState:         operatorv1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should stop NATS because backend is changed to EventMesh",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test"),
				testutils.WithStatusActiveBackend(operatorv1alpha1.NatsBackendType),
				testutils.WithStatusState(operatorv1alpha1.StateReady),
				testutils.WithStatusConditions([]kmetav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", true).Return(nil).Once()
				return managerMock
			},
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				return new(submgrmanagermocks.Manager)
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("DeletePublisherProxyResources", mock.Anything,
					mock.Anything).Return(nil).Once()
				return emMock
			},
			wantError:                 nil,
			wantEventingState:         operatorv1alpha1.StateProcessing,
			wantEventingConditionsLen: 0,
			wantNATSStopped:           true,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should return error because it failed to stop NATS backend",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test"),
				testutils.WithStatusActiveBackend(operatorv1alpha1.NatsBackendType),
				testutils.WithStatusState(operatorv1alpha1.StateReady),
				testutils.WithStatusConditions([]kmetav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", true).Return(ErrFailedToStop).Once()
				return managerMock
			},
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				return new(submgrmanagermocks.Manager)
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				return new(eventingmocks.Manager)
			},
			wantError:                 ErrFailedToStop,
			wantEventingState:         operatorv1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should stop EventMesh because backend is changed to NATS",
			givenEventing: testutils.NewEventingCR(
				testutils.WithNATSBackend(),
				testutils.WithStatusActiveBackend(operatorv1alpha1.EventMeshBackendType),
				testutils.WithStatusState(operatorv1alpha1.StateReady),
				testutils.WithStatusConditions([]kmetav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				return new(submgrmanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", true).Return(nil).Once()
				return managerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("DeletePublisherProxyResources", mock.Anything,
					mock.Anything).Return(nil).Once()
				return emMock
			},
			wantError:                 nil,
			wantEventingState:         operatorv1alpha1.StateProcessing,
			wantEventingConditionsLen: 0,
			wantNATSStopped:           false,
			wantEventMeshStopped:      true,
		},
		{
			name: "it should return error because it failed to stop EventMesh backend",
			givenEventing: testutils.NewEventingCR(
				testutils.WithNATSBackend(),
				testutils.WithStatusActiveBackend(operatorv1alpha1.EventMeshBackendType),
				testutils.WithStatusState(operatorv1alpha1.StateReady),
				testutils.WithStatusConditions([]kmetav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				return new(submgrmanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", true).Return(ErrFailedToStop).Once()
				return managerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				return new(eventingmocks.Manager)
			},
			wantError:                 ErrFailedToStop,
			wantEventingState:         operatorv1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should return error because it failed to remove EPP resources",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test"),
				testutils.WithStatusActiveBackend(operatorv1alpha1.NatsBackendType),
				testutils.WithStatusState(operatorv1alpha1.StateReady),
				testutils.WithStatusConditions([]kmetav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", true).Return(nil).Once()
				return managerMock
			},
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				return new(submgrmanagermocks.Manager)
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("DeletePublisherProxyResources", mock.Anything,
					mock.Anything).Return(ErrFailedToRemove).Once()
				return emMock
			},
			wantError:                 ErrFailedToRemove,
			wantEventingState:         operatorv1alpha1.StateProcessing,
			wantEventingConditionsLen: 0,
			wantNATSStopped:           true,
			wantEventMeshStopped:      false,
		},
	}

	// run test cases
	for _, testCase := range testCases {
		testcase := testCase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)
			testEnv.Reconciler.isNATSSubManagerStarted = true
			testEnv.Reconciler.isEventMeshSubManagerStarted = true

			mockNatsWatcher := new(watchermocks.Watcher)
			if testcase.wantNATSStopped {
				mockNatsWatcher.On("Stop").Once()
			}
			testEnv.Reconciler.natsWatchers[testcase.givenEventing.Namespace] = mockNatsWatcher

			// get mocks from test-case.
			givenNATSSubManagerMock := testcase.givenNATSSubManagerMock()
			givenEventMeshSubManagerMock := testcase.givenEventMeshSubManagerMock()
			givenEventingManagerMock := testcase.givenEventingManagerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.natsSubManager = givenNATSSubManagerMock
			testEnv.Reconciler.eventMeshSubManager = givenEventMeshSubManagerMock
			testEnv.Reconciler.eventingManager = givenEventingManagerMock

			// when
			err := testEnv.Reconciler.handleBackendSwitching(context.TODO(), testcase.givenEventing, logger)

			// then
			if testcase.wantError != nil {
				require.Error(t, err)
				require.Equal(t, testcase.wantError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			// check CR status.
			require.Equal(t, testcase.wantEventingState, testcase.givenEventing.Status.State)
			require.Len(t, testcase.givenEventing.Status.Conditions, testcase.wantEventingConditionsLen)

			// check assertions for mocks.
			givenNATSSubManagerMock.AssertExpectations(t)
			givenEventMeshSubManagerMock.AssertExpectations(t)

			// NATS
			if testcase.wantNATSStopped {
				require.Nil(t, testEnv.Reconciler.natsSubManager)
				require.False(t, testEnv.Reconciler.isNATSSubManagerStarted)
				givenEventingManagerMock.AssertExpectations(t)
			} else {
				require.NotNil(t, testEnv.Reconciler.natsSubManager)
				require.True(t, testEnv.Reconciler.isNATSSubManagerStarted)
			}

			// EventMesh
			if testcase.wantEventMeshStopped {
				require.Nil(t, testEnv.Reconciler.eventMeshSubManager)
				require.False(t, testEnv.Reconciler.isEventMeshSubManagerStarted)
				givenEventingManagerMock.AssertExpectations(t)
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
			watchErr:     ErrUseMeInMocks,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			testEnv.Reconciler.natsCRWatchStarted = testcase.watchStarted

			// Create a fake Eventing CR
			eventing := testutils.NewEventingCR(
				testutils.WithEventingCRName("test-name"),
				testutils.WithEventingCRNamespace("test-namespace"),
			)

			// Create mock watcher and controller
			natsWatcher := new(watchermocks.Watcher)
			natsWatcher.On("IsStarted").Return(testcase.watchStarted)
			mockController := new(eventingcontrollermocks.Controller)
			if !testcase.watchStarted {
				natsWatcher.On("Start").Once()
				natsWatcher.On("GetEventsChannel").Return(make(<-chan event.GenericEvent)).Once()

				mockController.On("Watch", mock.Anything, mock.Anything, mock.Anything).Return(testcase.watchErr).Once()
			}
			testEnv.Reconciler.natsWatchers[eventing.Namespace] = natsWatcher
			testEnv.Reconciler.controller = mockController

			// when
			err := testEnv.Reconciler.startNATSCRWatch(eventing)

			// then
			require.Equal(t, testcase.watchErr, err)
			require.NotNil(t, testEnv.Reconciler.natsWatchers[eventing.Namespace])
			if testcase.watchErr != nil {
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

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			testEnv.Reconciler.natsCRWatchStarted = testcase.natsCRWatchStarted

			// Create a fake Watcher
			natsWatcher := new(watchermocks.Watcher)
			natsWatcher.On("Stop").Times(1)

			eventing := testutils.NewEventingCR(
				testutils.WithEventingCRName("test-name"),
				testutils.WithEventingCRNamespace("test-namespace"),
			)

			testEnv.Reconciler.natsWatchers[eventing.Namespace] = natsWatcher

			testEnv.Reconciler.stopNATSCRWatch(eventing)

			// Check the results
			require.Nil(t, testcase.watchNatsWatcher)
			require.False(t, testcase.natsCRWatchStarted)
		})
	}
}
