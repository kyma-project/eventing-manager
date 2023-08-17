package eventing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/controller/eventing/mocks"
	"github.com/kyma-project/eventing-manager/pkg/env"
	managermocks "github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	subscriptionmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks"
	ecsubmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks/ec"
	"github.com/kyma-project/eventing-manager/test/utils"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
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

	givenNATSConfig := &env.NATSConfig{
		URL:                     "http://eventing-nats.svc.cluster.local",
		MaxReconnects:           10,
		ReconnectWait:           100,
		MaxIdleConns:            5,
		MaxConnsPerHost:         10,
		MaxIdleConnsPerHost:     10,
		IdleConnTimeout:         100,
		JSStreamName:            "kyma",
		JSSubjectPrefix:         "kyma",
		JSStreamRetentionPolicy: "Interest",
		JSStreamMaxMessages:     100000,
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
		givenShouldRetry             bool
		givenNATSSubManagerMock      func() *ecsubmanagermocks.Manager
		givenEventingManagerMock     func() *managermocks.Manager
		givenNatsConfigHandlerMock   func() *mocks.NatsConfigHandler
		givenManagerFactoryMock      func(*ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory
		wantAssertCheck              bool
		wantError                    error
	}{
		{
			name:                         "it should do nothing because subscription manager is already started",
			givenIsNATSSubManagerStarted: true,
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				return new(ecsubmanagermocks.Manager)
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				return nil
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				return nil
			},
			givenManagerFactoryMock: func(_ *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				return nil
			},
		},
		{
			name: "it should initialize and start subscription manager because " +
				"subscription manager is not started",
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
				return emMock
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				nchMock := new(mocks.NatsConfigHandler)
				nchMock.On("GetNatsConfig", mock.Anything, mock.Anything).Return(givenNATSConfig, nil)
				return nchMock
			},
			givenManagerFactoryMock: func(subManager *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				subManagerFactoryMock := new(subscriptionmanagermocks.ManagerFactory)
				subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(subManager).Once()
				return subManagerFactoryMock
			},
			wantAssertCheck: true,
		},
		{
			name: "it should retry to start subscription manager when subscription manager was " +
				"successfully initialized but failed to start",
			givenIsNATSSubManagerStarted: false,
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				jetStreamSubManagerMock := new(ecsubmanagermocks.Manager)
				jetStreamSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(errors.New("failed to start")).Twice()
				return jetStreamSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				emMock := new(managermocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				nchMock := new(mocks.NatsConfigHandler)
				nchMock.On("GetNatsConfig", mock.Anything, mock.Anything).Return(givenNATSConfig, nil)
				return nchMock
			},
			givenManagerFactoryMock: func(subManager *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				subManagerFactoryMock := new(subscriptionmanagermocks.ManagerFactory)
				subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(subManager).Once()
				return subManagerFactoryMock
			},
			wantAssertCheck:  true,
			givenShouldRetry: true,
			wantError:        errors.New("failed to start"),
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
			givenNatConfigHandlerMock := tc.givenNatsConfigHandlerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.isNATSSubManagerStarted = tc.givenIsNATSSubManagerStarted
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.natsConfigHandler = givenNatConfigHandlerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.natsSubManager = nil
			if givenManagerFactoryMock == nil {
				testEnv.Reconciler.natsSubManager = givenNATSSubManagerMock
			}

			// when
			err := testEnv.Reconciler.reconcileNATSSubManager(givenEventing, logger)
			if err != nil && tc.givenShouldRetry {
				// This is to test the scenario where initialization of natsSubManager was successful but
				// starting the natsSubManager failed. So on next try it should again try to start the natsSubManager.
				err = testEnv.Reconciler.reconcileNATSSubManager(givenEventing, logger)
			}

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.Equal(t, tc.wantError.Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, testEnv.Reconciler.natsSubManager)
				require.True(t, testEnv.Reconciler.isNATSSubManagerStarted)
			}

			if tc.wantAssertCheck {
				givenNATSSubManagerMock.AssertExpectations(t)
				givenManagerFactoryMock.AssertExpectations(t)
				givenEventingManagerMock.AssertExpectations(t)
				givenNATSSubManagerMock.AssertExpectations(t)
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
			name: "should do nothing when subscription manager is not initialised",
			givenNATSSubManagerMock: func() *ecsubmanagermocks.Manager {
				return nil
			},
			givenIsNATSSubManagerStarted: false,
			wantError:                    nil,
		},
		{
			name: "should return error when subscription manager fails to stop",
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
			name: "should succeed to stop subscription manager",
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

func Test_GetNatsConfig(t *testing.T) {
	// Define a list of test cases
	testCases := []struct {
		name               string
		eventing           *v1alpha1.Eventing
		expectedConfig     *env.NATSConfig
		givenNatsResources []natsv1alpha1.NATS
		expectedError      error
	}{
		{
			name: "Update NATSConfig",
			eventing: utils.NewEventingCR(
				utils.WithEventingCRName("test-eventing"),
				utils.WithEventingCRNamespace("test-namespace"),
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("File", "700Mi", 2, 1000),
				utils.WithEventingEventTypePrefix("test-prefix"),
			),
			givenNatsResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSCRName("test-nats"),
					natstestutils.WithNATSCRNamespace("test-namespace"),
				),
			},
			expectedConfig: &env.NATSConfig{
				URL:                     "nats://test-nats.test-namespace.svc.cluster.local:4222",
				EventTypePrefix:         "test-prefix",
				JSStreamStorageType:     "File",
				JSStreamReplicas:        2,
				JSStreamMaxBytes:        "700Mi",
				JSStreamMaxMsgsPerTopic: 1000,
				MaxReconnects:           10,
				ReconnectWait:           3 * time.Second,
				MaxIdleConns:            50,
				MaxConnsPerHost:         50,
				MaxIdleConnsPerHost:     50,
				IdleConnTimeout:         10 * time.Second,
				JSStreamName:            "sap",
				JSSubjectPrefix:         "",
				JSStreamRetentionPolicy: "interest",
				JSStreamDiscardPolicy:   "new",
				JSConsumerDeliverPolicy: "new",
				JSStreamMaxMessages:     -1,
			},
			expectedError: nil,
		},
		{
			name: "Error getting NATS URL",
			eventing: utils.NewEventingCR(
				utils.WithEventingCRName("test-eventing"),
				utils.WithEventingCRNamespace("test-namespace"),
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "700Mi", 2, 1000),
			),
			givenNatsResources: nil,
			expectedConfig:     nil,
			expectedError:      fmt.Errorf("failed to get NATS URL"),
		},
	}

	opts := &options.Options{}
	require.NoError(t, opts.Parse())
	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, tc.eventing.Namespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNatsResources,
			}, tc.expectedError)

			natsConfigHandler := NatsConfigHandlerImpl{
				kubeClient: kubeClient,
				opts:       opts,
			}

			// when
			natsConfig, err := natsConfigHandler.GetNatsConfig(ctx, *tc.eventing)

			// then
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expectedConfig, natsConfig)
		})
	}
}

func Test_getNATSUrl(t *testing.T) {
	testCases := []struct {
		name                string
		givenNatsResources  []natsv1alpha1.NATS
		givenNamespace      string
		want                string
		getNATSResourcesErr error
		wantErr             error
	}{
		{
			name: "NATS resource exists",
			givenNatsResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSCRName("test-nats"),
					natstestutils.WithNATSCRNamespace("test-namespace"),
				),
			},
			givenNamespace: "test-namespace",
			want:           "nats://test-nats.test-namespace.svc.cluster.local:4222",
			wantErr:        nil,
		},
		{
			name:                "NATS resource doesn't exist",
			givenNatsResources:  []natsv1alpha1.NATS{},
			givenNamespace:      "test-namespace",
			want:                "",
			getNATSResourcesErr: nil,
			wantErr:             fmt.Errorf("NATS CR is not found to build NATS server URL"),
		},
		{
			name:                "NATS resource does not exist",
			givenNatsResources:  nil,
			givenNamespace:      "test-namespace",
			want:                "",
			getNATSResourcesErr: fmt.Errorf("NATS CR is not found to build NATS server URL"),
			wantErr:             fmt.Errorf("NATS CR is not found to build NATS server URL"),
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, tc.givenNamespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNatsResources,
			}, tc.getNATSResourcesErr)

			natsConfigHandler := NatsConfigHandlerImpl{
				kubeClient: kubeClient,
			}

			// when
			url, err := natsConfigHandler.getNATSUrl(ctx, tc.givenNamespace)

			// then
			require.Equal(t, tc.wantErr, err)
			require.Equal(t, tc.want, url)
		})
	}
}

func Test_UpdateNatsConfig(t *testing.T) {
	t.Parallel()
	// Define a list of test cases
	testCases := []struct {
		name               string
		eventing           *v1alpha1.Eventing
		expectedConfig     env.NATSConfig
		givenNatsResources []natsv1alpha1.NATS
		expectedError      error
	}{
		{
			name: "Update NATSConfig",
			eventing: utils.NewEventingCR(
				utils.WithEventingCRName("test-eventing"),
				utils.WithEventingCRNamespace("test-namespace"),
			),
			givenNatsResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSCRName("test-nats"),
					natstestutils.WithNATSCRNamespace("test-namespace"),
				),
			},
			expectedConfig: env.NATSConfig{
				URL: "nats://test-nats.test-namespace.svc.cluster.local:4222",
			},
			expectedError: nil,
		},
		{
			name: "Error getting NATS URL",
			eventing: utils.NewEventingCR(
				utils.WithEventingCRName("test-eventing"),
				utils.WithEventingCRNamespace("test-namespace"),
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "700Mi", 2, 1000),
			),
			givenNatsResources: nil,
			expectedError:      fmt.Errorf("failed to get NATS URL"),
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, tc.eventing.Namespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNatsResources,
			}, tc.expectedError)

			natsConfigHandler := NatsConfigHandlerImpl{
				kubeClient: kubeClient,
			}

			// when
			natsConfig := env.NATSConfig{}
			err := natsConfigHandler.setUrlToNatsConfig(ctx, tc.eventing, &natsConfig)

			// then
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expectedConfig, tc.expectedConfig)
		})
	}
}
