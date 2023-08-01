package eventing

import (
	"errors"
	"testing"

	"github.com/kyma-project/eventing-manager/pkg/env"
	managermocks "github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	subscriptionmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks"
	ecsubmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks/ec"
	"github.com/kyma-project/eventing-manager/test/utils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_reconcileNATSSubManager(t *testing.T) {
	t.Parallel()

	// given - common for all test cases.
	givenEventing := utils.NewEventingCR(
		utils.WithEventingStreamData("Memory", "650M", 99, 98),
		utils.WithEventingEventTypePrefix("one.two.three"),
	)

	givenNATSConfig := env.NATSConfig{
		URL:                     "test123",
		MaxReconnects:           10,
		ReconnectWait:           11,
		EventTypePrefix:         "12",
		MaxIdleConns:            13,
		MaxConnsPerHost:         14,
		MaxIdleConnsPerHost:     15,
		IdleConnTimeout:         16,
		JSStreamName:            "17",
		JSSubjectPrefix:         "18",
		JSStreamStorageType:     "19",
		JSStreamReplicas:        20,
		JSStreamRetentionPolicy: "21",
		JSStreamMaxMessages:     22,
		JSStreamMaxBytes:        "23",
		JSStreamMaxMsgsPerTopic: 24,
		JSStreamDiscardPolicy:   "25",
		JSConsumerDeliverPolicy: "26",
	}

	givenBackendConfig := &env.BackendConfig{
		DefaultSubscriptionConfig: env.DefaultSubscriptionConfig{
			MaxInFlightMessages:   55,
			DispatcherRetryPeriod: 56,
			DispatcherMaxRetries:  57,
		},
	}

	// define test cases
	testCases := []struct {
		name                         string
		givenIsNATSSubManagerStarted bool
		givenNATSSubManagerMock      func() *ecsubmanagermocks.Manager
		givenEventingManagerMock     func() *managermocks.Manager
		givenManagerFactoryMock      func(*ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory
		wantAssertCheck              bool
	}{
		{
			name:                         "should do nothing if NATSSubManager is already started",
			givenIsNATSSubManagerStarted: true,
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				return nil
			},
			givenManagerFactoryMock: func(_ *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				return nil
			},
		},
		{
			name:                         "should init and start NATSSubManager",
			givenIsNATSSubManagerStarted: false,
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				jetStreamSubManagerMock := new(ecsubmanagermocks.Manager)
				jetStreamSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				return jetStreamSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				emMock := new(managermocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				emMock.On("GetNATSConfig").Return(givenNATSConfig)
				return emMock
			},
			givenManagerFactoryMock: func(subManager *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				subManagerFactoryMock := new(subscriptionmanagermocks.ManagerFactory)
				subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(subManager).Once()
				return subManagerFactoryMock
			},
			wantAssertCheck: true,
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

			// get mocks from test-case.
			givenNATSSubManagerMock := tc.givenNATSSubManagerMock()
			givenManagerFactoryMock := tc.givenManagerFactoryMock(givenNATSSubManagerMock)
			givenEventingManagerMock := tc.givenEventingManagerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.isNATSSubManagerStarted = tc.givenIsNATSSubManagerStarted
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.NATSSubManager = nil
			if givenManagerFactoryMock == nil {
				testEnv.Reconciler.NATSSubManager = givenNATSSubManagerMock
			}

			// when
			err := testEnv.Reconciler.reconcileNATSSubManager(givenEventing, logger)

			// then
			require.NoError(t, err)
			require.NotNil(t, testEnv.Reconciler.NATSSubManager)
			require.True(t, testEnv.Reconciler.isNATSSubManagerStarted)
			if tc.wantAssertCheck {
				givenNATSSubManagerMock.AssertExpectations(t)
				givenManagerFactoryMock.AssertExpectations(t)
				givenEventingManagerMock.AssertExpectations(t)
			}
		})
	}
}

func Test_stopNATSSubManager(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                         string
		givenNATSSubManagerMock      func() *ecsubmanagermocks.Manager
		givenIsNATSSubManagerStarted bool
		wantError                    error
		wantAssertCheck              bool
	}{
		{
			name: "should do nothing if NATSSubManager is not initialised",
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				return nil
			},
			givenIsNATSSubManagerStarted: false,
			wantError:                    nil,
		},
		{
			name: "should return error if NATSSubManager fails to stop",
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
				managerMock.On("Stop", mock.Anything).Return(errors.New("failed to stop")).Once()
				return managerMock
			},
			givenIsNATSSubManagerStarted: true,
			wantError:                    errors.New("failed to stop"),
			wantAssertCheck:              true,
		},
		{
			name: "should succeed to stop",
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
				managerMock.On("Stop", mock.Anything).Return(nil).Once()
				return managerMock
			},
			givenIsNATSSubManagerStarted: true,
			wantError:                    nil,
			wantAssertCheck:              true,
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

			// get mocks from test-case.
			givenNATSSubManagerMock := tc.givenNATSSubManagerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.NATSSubManager = givenNATSSubManagerMock
			testEnv.Reconciler.isNATSSubManagerStarted = tc.givenIsNATSSubManagerStarted

			// when
			err := testEnv.Reconciler.stopNATSSubManager(true, logger)
			// then
			if tc.wantError == nil {
				require.NoError(t, err)
				require.Nil(t, testEnv.Reconciler.NATSSubManager)
				require.False(t, testEnv.Reconciler.isNATSSubManagerStarted)
			} else {
				require.Equal(t, tc.wantError.Error(), err.Error())
			}

			if tc.wantAssertCheck {
				givenNATSSubManagerMock.AssertExpectations(t)
			}
		})
	}
}
