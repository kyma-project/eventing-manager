package eventing

import (
	"errors"
	"testing"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	ecsubmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks/ec"
	"github.com/kyma-project/eventing-manager/test/utils"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_handleBackendSwitching(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                         string
		givenEventing                *v1alpha1.Eventing
		givenNATSSubManagerMock      func() *ecsubmanagermocks.Manager
		givenEventMeshSubManagerMock func() *ecsubmanagermocks.Manager
		wantNATSStopped              bool
		wantEventMeshStopped         bool
		wantEventingState            string
		wantEventingConditionsLen    int
		wantError                    error
	}{
		{
			name: "it should do nothing because backend is not changed",
			givenEventing: utils.NewEventingCR(
				utils.WithNATSBackend(),
				utils.WithStatusActiveBackend(v1alpha1.NatsBackendType),
				utils.WithStatusState(v1alpha1.StateReady),
				utils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			wantError:                 nil,
			wantEventingState:         v1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should stop NATS because backend is changed to EventMesh",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test"),
				utils.WithStatusActiveBackend(v1alpha1.NatsBackendType),
				utils.WithStatusState(v1alpha1.StateReady),
				utils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
				managerMock.On("Stop", true).Return(nil).Once()
				return managerMock
			},
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			wantError:                 nil,
			wantEventingState:         v1alpha1.StateProcessing,
			wantEventingConditionsLen: 0,
			wantNATSStopped:           true,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should return error because it failed to stop NATS backend",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test"),
				utils.WithStatusActiveBackend(v1alpha1.NatsBackendType),
				utils.WithStatusState(v1alpha1.StateReady),
				utils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
				managerMock.On("Stop", true).Return(errors.New("failed to stop")).Once()
				return managerMock
			},
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			wantError:                 errors.New("failed to stop"),
			wantEventingState:         v1alpha1.StateReady,
			wantEventingConditionsLen: 1,
			wantNATSStopped:           false,
			wantEventMeshStopped:      false,
		},
		{
			name: "it should stop EventMesh because backend is changed to NATS",
			givenEventing: utils.NewEventingCR(
				utils.WithNATSBackend(),
				utils.WithStatusActiveBackend(v1alpha1.EventMeshBackendType),
				utils.WithStatusState(v1alpha1.StateReady),
				utils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
				managerMock.On("Stop", true).Return(nil).Once()
				return managerMock
			},
			wantError:                 nil,
			wantEventingState:         v1alpha1.StateProcessing,
			wantEventingConditionsLen: 0,
			wantNATSStopped:           false,
			wantEventMeshStopped:      true,
		},
		{
			name: "it should return error because it failed to stop EventMesh backend",
			givenEventing: utils.NewEventingCR(
				utils.WithNATSBackend(),
				utils.WithStatusActiveBackend(v1alpha1.EventMeshBackendType),
				utils.WithStatusState(v1alpha1.StateReady),
				utils.WithStatusConditions([]metav1.Condition{{Type: "Available"}}),
			),
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
				managerMock.On("Stop", true).Return(errors.New("failed to stop")).Once()
				return managerMock
			},
			wantError:                 errors.New("failed to stop"),
			wantEventingState:         v1alpha1.StateReady,
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
		})
	}
}
