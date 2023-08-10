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
		URL:                     "http://eventing-nats.svc.cluster.local",
		MaxReconnects:           10,
		ReconnectWait:           100,
		EventTypePrefix:         "sap.kyma.custom",
		MaxIdleConns:            5,
		MaxConnsPerHost:         10,
		MaxIdleConnsPerHost:     10,
		IdleConnTimeout:         100,
		JSStreamName:            "kyma",
		JSSubjectPrefix:         "kyma",
		JSStreamStorageType:     "File",
		JSStreamReplicas:        3,
		JSStreamRetentionPolicy: "Interest",
		JSStreamMaxMessages:     100000,
		JSStreamMaxBytes:        "700Mi",
		JSStreamMaxMsgsPerTopic: 10000,
		JSStreamDiscardPolicy:   "DiscardNew",
		JSConsumerDeliverPolicy: "DeliverNew",
	}

	givenBackendConfig := &env.BackendConfig{
		DefaultSubscriptionConfig: env.DefaultSubscriptionConfig{
			MaxInFlightMessages:   6,
			DispatcherRetryPeriod: 30,
			DispatcherMaxRetries:  100,
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
			name:                         "it should do nothing because NATSSubManager is already started",
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
			name: "it should initialize and start subscription manager because " +
				"natsSubManager is not started",
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
			testEnv.Reconciler.natsSubManager = nil
			if givenManagerFactoryMock == nil {
				testEnv.Reconciler.natsSubManager = givenNATSSubManagerMock
			}

			// when
			err := testEnv.Reconciler.reconcileNATSSubManager(givenEventing, logger)

			// then
			require.NoError(t, err)
			require.NotNil(t, testEnv.Reconciler.natsSubManager)
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
			testEnv.Reconciler.natsSubManager = givenNATSSubManagerMock
			testEnv.Reconciler.isNATSSubManagerStarted = tc.givenIsNATSSubManagerStarted

			// when
			err := testEnv.Reconciler.stopNATSSubManager(true, logger)
			// then
			if tc.wantError == nil {
				require.NoError(t, err)
				require.Nil(t, testEnv.Reconciler.natsSubManager)
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
