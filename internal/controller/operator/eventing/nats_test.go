package eventing

import (
	"context"
	"errors"
	"testing"
	"time"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/controller/operator/eventing/mocks"
	"github.com/kyma-project/eventing-manager/options"
	"github.com/kyma-project/eventing-manager/pkg/env"
	eventingmocks "github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	submgrmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager/mocks"
	submgrmocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks"
	"github.com/kyma-project/eventing-manager/test/utils"
)

var ErrUseMeInMocks = errors.New("use me in mocks")

func Test_reconcileNATSSubManager(t *testing.T) {
	t.Parallel()

	// given - common for all test cases.
	givenEventing := utils.NewEventingCR(
		utils.WithEventingCRName("eventing"),
		utils.WithEventingCRNamespace("kyma-system"),
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
		givenUpdateTest              bool
		givenHashBefore              int64
		givenNATSSubManagerMock      func() *submgrmanagermocks.Manager
		givenEventingManagerMock     func() *eventingmocks.Manager
		givenNatsConfigHandlerMock   func() *mocks.NatsConfigHandler
		givenManagerFactoryMock      func(*submgrmanagermocks.Manager) *submgrmocks.ManagerFactory
		wantAssertCheck              bool
		wantError                    error
		wantHashAfter                int64
	}{
		{
			name:                         "it should do nothing because subscription manager is already started",
			givenIsNATSSubManagerStarted: true,
			givenHashBefore:              int64(-7550677537009891034),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				jetStreamSubManagerMock := new(submgrmanagermocks.Manager)
				jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Stop", mock.Anything, mock.Anything).Return(nil).Once()
				return jetStreamSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				nchMock := new(mocks.NatsConfigHandler)
				nchMock.On("GetNatsConfig", mock.Anything, mock.Anything).Return(givenNATSConfig, nil)
				return nchMock
			},
			givenManagerFactoryMock: func(_ *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				return nil
			},
			wantHashAfter: int64(-7550677537009891034),
		},
		{
			name: "it should initialize and start subscription manager because " +
				"subscription manager is not started",
			givenIsNATSSubManagerStarted: false,
			givenHashBefore:              int64(0),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				jetStreamSubManagerMock := new(submgrmanagermocks.Manager)
				jetStreamSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				return jetStreamSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				nchMock := new(mocks.NatsConfigHandler)
				nchMock.On("GetNatsConfig", mock.Anything, mock.Anything).Return(givenNATSConfig, nil)
				return nchMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(subManager).Once()
				return subManagerFactoryMock
			},
			wantAssertCheck: true,
			wantHashAfter:   int64(-7550677537009891034),
		},
		{
			name: "it should retry to start subscription manager when subscription manager was " +
				"successfully initialized but failed to start",
			givenIsNATSSubManagerStarted: false,
			givenHashBefore:              int64(-7550677537009891034),
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				jetStreamSubManagerMock := new(submgrmanagermocks.Manager)
				jetStreamSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(ErrUseMeInMocks).Twice()
				return jetStreamSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				nchMock := new(mocks.NatsConfigHandler)
				nchMock.On("GetNatsConfig", mock.Anything, mock.Anything).Return(givenNATSConfig, nil)
				return nchMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(subManager).Once()
				return subManagerFactoryMock
			},
			wantAssertCheck:  true,
			givenShouldRetry: true,
			wantError:        ErrUseMeInMocks,
			wantHashAfter:    int64(-7550677537009891034),
		},
		{
			name:                         "it should update the subscription manager when the backend config changes",
			givenIsNATSSubManagerStarted: true,
			givenHashBefore:              int64(-8550677537009891034),
			givenUpdateTest:              true,
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				jetStreamSubManagerMock := new(submgrmanagermocks.Manager)
				jetStreamSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Stop", mock.Anything, mock.Anything).Return(nil).Once()
				return jetStreamSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig).Twice()
				return emMock
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				nchMock := new(mocks.NatsConfigHandler)
				nchMock.On("GetNatsConfig", mock.Anything, mock.Anything).Return(givenNATSConfig, nil)
				return nchMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(subManager).Once()
				return subManagerFactoryMock
			},
			wantAssertCheck: true,
			wantHashAfter:   int64(-7550677537009891034),
		},
		{
			name: "it should update the subscription manager when the backend config changes" +
				"but subscription manager failed to start",
			givenIsNATSSubManagerStarted: false,
			givenHashBefore:              int64(-8550677537009891034),
			givenUpdateTest:              true,
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				jetStreamSubManagerMock := new(submgrmanagermocks.Manager)
				jetStreamSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				jetStreamSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				return jetStreamSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig).Twice()
				return emMock
			},
			givenNatsConfigHandlerMock: func() *mocks.NatsConfigHandler {
				nchMock := new(mocks.NatsConfigHandler)
				nchMock.On("GetNatsConfig", mock.Anything, mock.Anything).Return(givenNATSConfig, nil)
				return nchMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewJetStreamManager", mock.Anything, mock.Anything).Return(subManager).Once()
				return subManagerFactoryMock
			},
			wantAssertCheck: true,
			wantHashAfter:   int64(-7550677537009891034),
		},
	}

	// run test cases
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t, givenEventing)
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			// get mocks from test-case.
			givenNATSSubManagerMock := testcase.givenNATSSubManagerMock()
			givenManagerFactoryMock := testcase.givenManagerFactoryMock(givenNATSSubManagerMock)
			givenEventingManagerMock := testcase.givenEventingManagerMock()
			givenNatConfigHandlerMock := testcase.givenNatsConfigHandlerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.isNATSSubManagerStarted = testcase.givenIsNATSSubManagerStarted
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.natsConfigHandler = givenNatConfigHandlerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.natsSubManager = nil
			if givenManagerFactoryMock == nil || testcase.givenUpdateTest {
				testEnv.Reconciler.natsSubManager = givenNATSSubManagerMock
			}

			// set the backend hash before depending on test
			givenEventing.Status.BackendConfigHash = testcase.givenHashBefore

			// when
			err := testEnv.Reconciler.reconcileNATSSubManager(givenEventing, logger)
			if err != nil && testcase.givenShouldRetry {
				// This is to test the scenario where initialization of natsSubManager was successful but
				// starting the natsSubManager failed. So on next try it should again try to start the natsSubManager.
				err = testEnv.Reconciler.reconcileNATSSubManager(givenEventing, logger)
			}
			if testcase.givenUpdateTest {
				// Run reconcile again with newBackendConfig:
				err = testEnv.Reconciler.reconcileNATSSubManager(givenEventing, logger)
				require.NoError(t, err)
			}

			// check for backend hash after
			require.Equal(t, testcase.wantHashAfter, givenEventing.Status.BackendConfigHash)

			// then
			if testcase.wantError != nil {
				require.Error(t, err)
				require.Equal(t, testcase.wantError.Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, testEnv.Reconciler.natsSubManager)
				require.True(t, testEnv.Reconciler.isNATSSubManagerStarted)
			}

			if testcase.wantAssertCheck {
				givenNATSSubManagerMock.AssertExpectations(t)
				givenEventingManagerMock.AssertExpectations(t)
				givenNatConfigHandlerMock.AssertExpectations(t)
			}
			if !testcase.givenIsNATSSubManagerStarted {
				givenManagerFactoryMock.AssertExpectations(t)
			}
		})
	}
}

//nolint:dupl // quite similar to the eventmesh test
func Test_stopNATSSubManager(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                         string
		givenNATSSubManagerMock      func() *submgrmanagermocks.Manager
		givenIsNATSSubManagerStarted bool
		wantError                    error
		wantAssertCheck              bool
	}{
		{
			name: "should do nothing when subscription manager is not initialised",
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				return nil
			},
			givenIsNATSSubManagerStarted: false,
			wantError:                    nil,
		},
		{
			name: "should return error when subscription manager fails to stop",
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", mock.Anything).Return(ErrUseMeInMocks).Once()
				return managerMock
			},
			givenIsNATSSubManagerStarted: true,
			wantError:                    ErrUseMeInMocks,
			wantAssertCheck:              true,
		},
		{
			name: "should succeed to stop subscription manager",
			givenNATSSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			// get mocks from test-case.
			givenNATSSubManagerMock := testcase.givenNATSSubManagerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.natsSubManager = givenNATSSubManagerMock
			testEnv.Reconciler.isNATSSubManagerStarted = testcase.givenIsNATSSubManagerStarted

			// when
			err := testEnv.Reconciler.stopNATSSubManager(true, logger)
			// then
			if testcase.wantError == nil {
				require.NoError(t, err)
				require.Nil(t, testEnv.Reconciler.natsSubManager)
				require.False(t, testEnv.Reconciler.isNATSSubManagerStarted)
			} else {
				require.Equal(t, testcase.wantError.Error(), err.Error())
			}

			if testcase.wantAssertCheck {
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
			expectedError:      ErrUseMeInMocks,
		},
	}

	opts := &options.Options{}
	require.NoError(t, opts.Parse())
	// run test cases
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, testcase.eventing.Namespace).Return(&natsv1alpha1.NATSList{
				Items: testcase.givenNatsResources,
			}, testcase.expectedError)

			natsConfigHandler := NatsConfigHandlerImpl{
				kubeClient: kubeClient,
				opts:       opts,
			}

			// when
			natsConfig, err := natsConfigHandler.GetNatsConfig(ctx, *testcase.eventing)

			// then
			require.Equal(t, testcase.expectedError, err)
			require.Equal(t, testcase.expectedConfig, natsConfig)
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
			wantErr:             ErrCannotBuildNATSURL,
		},
		{
			name:                "NATS resource does not exist",
			givenNatsResources:  nil,
			givenNamespace:      "test-namespace",
			want:                "",
			getNATSResourcesErr: ErrCannotBuildNATSURL,
			wantErr:             ErrCannotBuildNATSURL,
		},
	}

	// run test cases
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, testcase.givenNamespace).Return(&natsv1alpha1.NATSList{
				Items: testcase.givenNatsResources,
			}, testcase.getNATSResourcesErr)

			natsConfigHandler := NatsConfigHandlerImpl{
				kubeClient: kubeClient,
			}

			// when
			url, err := natsConfigHandler.getNATSUrl(ctx, testcase.givenNamespace)

			// then
			require.Equal(t, testcase.wantErr, err)
			require.Equal(t, testcase.want, url)
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
			expectedError:      ErrCannotBuildNATSURL,
		},
	}

	// run test cases
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, testcase.eventing.Namespace).Return(&natsv1alpha1.NATSList{
				Items: testcase.givenNatsResources,
			}, testcase.expectedError)

			natsConfigHandler := NatsConfigHandlerImpl{
				kubeClient: kubeClient,
			}

			// when
			natsConfig := env.NATSConfig{}
			err := natsConfigHandler.setURLToNatsConfig(ctx, testcase.eventing, &natsConfig)

			// then
			require.Equal(t, testcase.expectedError, err)
			require.Equal(t, testcase.expectedConfig, testcase.expectedConfig)
		})
	}
}
