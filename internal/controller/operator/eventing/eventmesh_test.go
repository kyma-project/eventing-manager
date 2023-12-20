package eventing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/label"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	eventingmocks "github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	submgrmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager/mocks"
	submgrmocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks"
	"github.com/kyma-project/eventing-manager/test/utils"
)

const (
	defaultEventingWebhookAuthSecretName      = "eventing-webhook-auth"
	defaultEventingWebhookAuthSecretNamespace = "kyma-system"
)

var (
	ErrFailedToStart  = errors.New("failed to start")
	ErrFailedToStop   = errors.New("failed to stop")
	ErrFailedToRemove = errors.New("failed to remove")
	errNotFound       = errors.New("secret not found")
)

//nolint:goerr113 // all tests here need to be fixed, as they use require.ErrorAs and use it wrongly
func Test_reconcileEventMeshSubManager(t *testing.T) {
	t.Parallel()

	// given - common for all test cases.
	givenEventing := utils.NewEventingCR(
		utils.WithEventingCRName("eventing"),
		utils.WithEventingCRNamespace("test-namespace"),
		utils.WithEventMeshBackend("test-namespace/test-secret-name"),
		utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
		utils.WithEventingEventTypePrefix("test-prefix"),
		utils.WithEventingDomain(utils.Domain),
	)

	givenOauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", givenEventing.Namespace)

	givenBackendConfig := &env.BackendConfig{
		EventingWebhookAuthSecretName: "eventing-webhook-auth",
	}

	givenConfigMap := &kcorev1.ConfigMap{
		Data: map[string]string{
			shootInfoConfigMapKeyDomain: utils.Domain,
		},
	}

	ctx := context.Background()

	// define test cases
	testCases := []struct {
		name                              string
		givenIsEventMeshSubManagerStarted bool
		givenShouldRetry                  bool
		givenUpdateTest                   bool
		givenHashBefore                   int64
		givenEventMeshSubManagerMock      func() *submgrmanagermocks.Manager
		givenEventingManagerMock          func() *eventingmocks.Manager
		givenManagerFactoryMock           func(*submgrmanagermocks.Manager) *submgrmocks.ManagerFactory
		givenClientMock                   func() client.Client
		givenKubeClientMock               func() k8s.Client
		wantAssertCheck                   bool
		wantError                         error
		wantHashAfter                     int64
	}{
		{
			name:                              "it should do nothing because syncing OAuth secret failed",
			givenIsEventMeshSubManagerStarted: true,
			givenHashBefore:                   int64(0),
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				return nil
			},
			givenManagerFactoryMock: func(_ *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				return nil
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					nil, errNotFound).Once()
				return mockKubeClient
			},
			wantError:     fmt.Errorf("failed to sync OAuth secret: %w", errNotFound),
			wantHashAfter: int64(0),
		},
		{
			name:                              "it should do nothing because failed sync Publisher Proxy secret",
			givenIsEventMeshSubManagerStarted: true,
			givenHashBefore:                   int64(0),
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				return nil
			},
			givenManagerFactoryMock: func(_ *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				return nil
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewOAuthSecret("test-secret", givenEventing.Namespace), nil).Once()
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(errors.New("failed to apply patch")).Once()
				return mockKubeClient
			},
			wantError: errors.New("failed to sync Publisher Proxy secret: failed to apply patch"),
		},
		{
			name:                              "it should do nothing because subscription manager is already started",
			givenIsEventMeshSubManagerStarted: true,
			givenHashBefore:                   int64(4922936597877296700),
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenManagerFactoryMock: func(_ *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				return nil
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetConfigMap", ctx, mock.Anything, mock.Anything).Return(givenConfigMap, nil).Once()
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Once()
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewOAuthSecret("test-secret", givenEventing.Namespace), nil).Once()
				return mockKubeClient
			},
			wantHashAfter: int64(4922936597877296700),
		},
		{
			name: "it should initialize and start subscription manager because " +
				"subscription manager is not started",
			givenIsEventMeshSubManagerStarted: false,
			givenHashBefore:                   int64(0),
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				eventMeshSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewEventMeshManager", mock.Anything).Return(subManager, nil).Once()
				return subManagerFactoryMock
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetConfigMap", ctx, mock.Anything, mock.Anything).Return(givenConfigMap, nil).Once()
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Once()
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewOAuthSecret("test-secret", givenEventing.Namespace), nil).Once()
				return mockKubeClient
			},
			wantAssertCheck: true,
			wantHashAfter:   int64(4922936597877296700),
		},
		{
			name: "it should retry to start subscription manager when subscription manager was " +
				"successfully initialized but failed to start",
			givenIsEventMeshSubManagerStarted: false,
			givenHashBefore:                   int64(0),
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				eventMeshSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(errors.New("failed to start")).Twice()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewEventMeshManager", mock.Anything).Return(subManager, nil).Once()
				return subManagerFactoryMock
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Twice()
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewOAuthSecret("test-secret", givenEventing.Namespace), nil).Twice()
				return mockKubeClient
			},
			wantAssertCheck:  true,
			givenShouldRetry: true,
			wantError:        errors.New("failed to start"),
			wantHashAfter:    int64(0),
		},
		{
			name:                              "it should update the subscription manager when the backend config changes",
			givenIsEventMeshSubManagerStarted: true,
			givenHashBefore:                   int64(-2279197549452913403),
			givenUpdateTest:                   true,
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				eventMeshSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				eventMeshSubManagerMock.On("Stop", mock.Anything, mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig).Twice()
				return emMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewEventMeshManager", mock.Anything).Return(subManager, nil).Once()
				return subManagerFactoryMock
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Twice()
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewOAuthSecret("test-secret", givenEventing.Namespace), nil).Twice()
				return mockKubeClient
			},
			wantAssertCheck: true,
			wantHashAfter:   int64(4922936597877296700),
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t, givenEventing, givenOauthSecret)
			testEnv.Reconciler.backendConfig = *givenBackendConfig

			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			// get mocks from test-case.
			givenEventMeshSubManagerMock := tc.givenEventMeshSubManagerMock()
			givenManagerFactoryMock := tc.givenManagerFactoryMock(givenEventMeshSubManagerMock)
			givenEventingManagerMock := tc.givenEventingManagerMock()

			// connect mocks with reconciler.
			if tc.givenKubeClientMock != nil {
				testEnv.Reconciler.kubeClient = tc.givenKubeClientMock()
			}

			testEnv.Reconciler.isEventMeshSubManagerStarted = tc.givenIsEventMeshSubManagerStarted
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.eventMeshSubManager = nil
			if givenManagerFactoryMock == nil || tc.givenUpdateTest {
				testEnv.Reconciler.eventMeshSubManager = givenEventMeshSubManagerMock
			}

			// set the backend hash before depending on test
			givenEventing.Status.BackendConfigHash = tc.givenHashBefore

			// when

			eventMeshSecret := utils.NewEventMeshSecret("eventing-backend", givenEventing.Namespace)

			err := testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing, eventMeshSecret, logger)
			if err != nil && tc.givenShouldRetry {
				// This is to test the scenario where initialization of eventMeshSubManager was successful but
				// starting the eventMeshSubManager failed. So on next try it should again try to start the eventMeshSubManager.
				err = testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing, eventMeshSecret, logger)
			}
			if tc.givenUpdateTest {
				// Run reconcile again with newBackendConfig:
				err = testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing, eventMeshSecret, logger)
				require.NoError(t, err)
			}

			// check for backend hash after
			require.Equal(t, tc.wantHashAfter, givenEventing.Status.BackendConfigHash)

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.Equal(t, err.Error(), tc.wantError.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, testEnv.Reconciler.eventMeshSubManager)
				require.True(t, testEnv.Reconciler.isEventMeshSubManagerStarted)
			}

			if tc.wantAssertCheck {
				givenEventMeshSubManagerMock.AssertExpectations(t)
				givenManagerFactoryMock.AssertExpectations(t)
				givenEventingManagerMock.AssertExpectations(t)
			}
		})
	}
}

func Test_reconcileEventMeshSubManager_ReadClusterDomain(t *testing.T) {
	t.Parallel()

	const (
		namespace = "test-namespace"
	)

	ctx := context.Background()

	givenOauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", namespace)

	givenBackendConfig := &env.BackendConfig{
		EventingWebhookAuthSecretName: "eventing-webhook-auth",
	}

	givenConfigMap := &kcorev1.ConfigMap{
		Data: map[string]string{
			shootInfoConfigMapKeyDomain: utils.Domain,
		},
	}

	testCases := []struct {
		name                         string
		givenEventing                *v1alpha1.Eventing
		givenEventMeshSubManagerMock func() *submgrmanagermocks.Manager
		givenEventingManagerMock     func() *eventingmocks.Manager
		givenManagerFactoryMock      func(*submgrmanagermocks.Manager) *submgrmocks.ManagerFactory
		givenKubeClientMock          func() (k8s.Client, *k8smocks.Client)
	}{
		{
			name: "should not read the domain from the configmap because it is configured in the Eventing CR",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRName("eventing"),
				utils.WithEventingCRNamespace(namespace),
				utils.WithEventMeshBackend("test-namespace/test-secret-name"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewEventMeshManager", mock.Anything).Return(subManager, nil).Once()
				return subManagerFactoryMock
			},
			givenKubeClientMock: func() (k8s.Client, *k8smocks.Client) {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetSecret", mock.Anything, mock.Anything).Return(givenOauthSecret, nil).Once()
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Once()
				return mockKubeClient, mockKubeClient
			},
		},
		{
			name: "should read the domain from the configmap because it is not configured in the Eventing CR",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRName("eventing"),
				utils.WithEventingCRNamespace(namespace),
				utils.WithEventMeshBackend("test-namespace/test-secret-name"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(""),
			),
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
				eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *eventingmocks.Manager {
				emMock := new(eventingmocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenManagerFactoryMock: func(subManager *submgrmanagermocks.Manager) *submgrmocks.ManagerFactory {
				subManagerFactoryMock := new(submgrmocks.ManagerFactory)
				subManagerFactoryMock.On("NewEventMeshManager", mock.Anything).Return(subManager, nil).Once()
				return subManagerFactoryMock
			},
			givenKubeClientMock: func() (k8s.Client, *k8smocks.Client) {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetSecret", mock.Anything, mock.Anything).Return(givenOauthSecret, nil).Once()
				mockKubeClient.On("GetConfigMap", ctx, mock.Anything, mock.Anything).Return(givenConfigMap, nil).Once()
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Once()
				return mockKubeClient, mockKubeClient
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t, tc.givenEventing, givenOauthSecret)
			testEnv.Reconciler.backendConfig = *givenBackendConfig

			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			givenEventMeshSubManagerMock := tc.givenEventMeshSubManagerMock()
			givenManagerFactoryMock := tc.givenManagerFactoryMock(givenEventMeshSubManagerMock)
			givenEventingManagerMock := tc.givenEventingManagerMock()

			var mockClient *k8smocks.Client
			if tc.givenKubeClientMock != nil {
				testEnv.Reconciler.kubeClient, mockClient = tc.givenKubeClientMock()
			}

			testEnv.Reconciler.isEventMeshSubManagerStarted = true
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.eventMeshSubManager = nil

			// when
			eventMeshSecret := utils.NewEventMeshSecret("test-secret", tc.givenEventing.Namespace)
			err := testEnv.Reconciler.reconcileEventMeshSubManager(ctx, tc.givenEventing, eventMeshSecret, logger)

			// then
			require.NoError(t, err)
			givenEventMeshSubManagerMock.AssertExpectations(t)
			givenEventingManagerMock.AssertExpectations(t)
			givenManagerFactoryMock.AssertExpectations(t)
			mockClient.AssertExpectations(t)
		})
	}
}

func Test_stopEventMeshSubManager(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                              string
		givenEventMeshSubManagerMock      func() *submgrmanagermocks.Manager
		givenIsEventMeshSubManagerStarted bool
		wantError                         error
		wantAssertCheck                   bool
	}{
		{
			name: "should do nothing when subscription manager is not initialised",
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				return nil
			},
			givenIsEventMeshSubManagerStarted: false,
			wantError:                         nil,
		},
		{
			name: "should return error when subscription manager fails to stop",
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", mock.Anything).Return(ErrFailedToStop).Once()
				return managerMock
			},
			givenIsEventMeshSubManagerStarted: true,
			wantError:                         ErrFailedToStop,
			wantAssertCheck:                   true,
		},
		{
			name: "should succeed to stop subscription manager",
			givenEventMeshSubManagerMock: func() *submgrmanagermocks.Manager {
				managerMock := new(submgrmanagermocks.Manager)
				managerMock.On("Stop", mock.Anything).Return(nil).Once()
				return managerMock
			},
			givenIsEventMeshSubManagerStarted: true,
			wantError:                         nil,
			wantAssertCheck:                   true,
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
			givenEventMeshSubManagerMock := tc.givenEventMeshSubManagerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.eventMeshSubManager = givenEventMeshSubManagerMock
			testEnv.Reconciler.isEventMeshSubManagerStarted = tc.givenIsEventMeshSubManagerStarted

			// when
			err := testEnv.Reconciler.stopEventMeshSubManager(true, logger)
			// then
			if tc.wantError == nil {
				require.NoError(t, err)
				require.Nil(t, testEnv.Reconciler.eventMeshSubManager)
				require.False(t, testEnv.Reconciler.isEventMeshSubManagerStarted)
			} else {
				require.Equal(t, tc.wantError.Error(), err.Error())
			}

			if tc.wantAssertCheck {
				givenEventMeshSubManagerMock.AssertExpectations(t)
			}
		})
	}
}

// TestGetSecretForPublisher verifies the successful and failing retrieval
// of secrets.
func Test_GetSecretForPublisher(t *testing.T) {
	secretFor := func(message, namespace []byte) *kcorev1.Secret {
		secret := &kcorev1.Secret{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      eventing.PublisherName,
				Namespace: "test-namespace",
			},
		}

		secret.Data = make(map[string][]byte)

		if len(message) > 0 {
			secret.Data["messaging"] = message
		}
		if len(namespace) > 0 {
			secret.Data["namespace"] = namespace
		}

		return secret
	}

	testCases := []struct {
		name           string
		messagingData  []byte
		namespaceData  []byte
		expectedSecret *kcorev1.Secret
		expectedError  error
	}{
		{
			name: "with valid message and namespace data",
			messagingData: []byte(`[
									  {
										"broker": {
										  "type": "sapmgw"
										},
										"oa2": {
										  "clientid": "clientid",
										  "clientsecret": "clientsecret",
										  "granttype": "client_credentials",
										  "tokenendpoint": "https://token"
										},
										"protocol": [
										  "amqp10ws"
										],
										"uri": "wss://amqp"
									  },
									  {
										"broker": {
										  "type": "sapmgw"
										},
										"oa2": {
										  "clientid": "clientid",
										  "clientsecret": "clientsecret",
										  "granttype": "client_credentials",
										  "tokenendpoint": "https://token"
										},
										"protocol": [
										  "amqp10ws"
										],
										"uri": "wss://amqp"
									  },
									  {
										"broker": {
										  "type": "saprestmgw"
										},
										"oa2": {
										  "clientid": "rest-clientid",
										  "clientsecret": "rest-client-secret",
										  "granttype": "client_credentials",
										  "tokenendpoint": "https://rest-token"
										},
										"protocol": [
										  "httprest"
										],
										"uri": "https://rest-messaging"
									  }
									] `),
			namespaceData: []byte("valid/namespace"),
			expectedSecret: &kcorev1.Secret{
				TypeMeta: kmetav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: kcorev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      eventing.PublisherName,
					Namespace: "test-namespace",
					Labels: map[string]string{
						label.KeyName: label.ValueEventingPublisherProxy,
					},
				},
				StringData: map[string]string{
					"client-id":        "rest-clientid",
					"client-secret":    "rest-client-secret",
					"token-endpoint":   "https://rest-token?grant_type=client_credentials&response_type=token",
					"ems-publish-host": "https://rest-messaging",
					"ems-publish-url":  "https://rest-messaging/sap/ems/v1/events",
					"beb-namespace":    "valid/namespace",
				},
			},
		},
		{
			name:          "with empty message data",
			namespaceData: []byte("valid/namespace"),
			expectedError: ErrEMSecretMessagingMissing,
		},
		{
			name: "with empty namespace data",
			messagingData: []byte(`[
									  {
										"broker": {
										  "type": "sapmgw"
										},
										"oa2": {
										  "clientid": "clientid",
										  "clientsecret": "clientsecret",
										  "granttype": "client_credentials",
										  "tokenendpoint": "https://token"
										},
										"protocol": [
										  "amqp10ws"
										],
										"uri": "wss://amqp"
									  },
									  {
										"broker": {
										  "type": "sapmgw"
										},
										"oa2": {
										  "clientid": "clientid",
										  "clientsecret": "clientsecret",
										  "granttype": "client_credentials",
										  "tokenendpoint": "https://token"
										},
										"protocol": [
										  "amqp10ws"
										],
										"uri": "wss://amqp"
									  },
									  {
										"broker": {
										  "type": "saprestmgw"
										},
										"oa2": {
										  "clientid": "rest-clientid",
										  "clientsecret": "rest-client-secret",
										  "granttype": "client_credentials",
										  "tokenendpoint": "https://rest-token"
										},
										"protocol": [
										  "httprest"
										],
										"uri": "https://rest-messaging"
									  }
									]`),
			expectedError: ErrEMSecretNamespaceMissing,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			publisherSecret := secretFor(tc.messagingData, tc.namespaceData)

			gotPublisherSecret, err := getSecretForPublisher(publisherSecret)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.expectedError.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedSecret, gotPublisherSecret, "invalid publisher secret")
		})
	}
}

func Test_getOAuth2ClientCredentials(t *testing.T) {
	testCases := []struct {
		name             string
		givenSecret      *kcorev1.Secret
		wantError        bool
		wantClientID     []byte
		wantClientSecret []byte
		wantTokenURL     []byte
		wantCertsURL     []byte
	}{
		{
			name:        "secret does not exist",
			givenSecret: nil,
			wantError:   true,
		},
		{
			name: "secret exists with missing data",
			givenSecret: &kcorev1.Secret{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      defaultEventingWebhookAuthSecretName,
					Namespace: defaultEventingWebhookAuthSecretNamespace,
				},
				Data: map[string][]byte{
					secretKeyClientID: []byte("test-client-id-0"),
					// missing data
				},
			},
			wantError: true,
		},
		{
			name: "secret exists with all data",
			givenSecret: &kcorev1.Secret{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      defaultEventingWebhookAuthSecretName,
					Namespace: defaultEventingWebhookAuthSecretNamespace,
				},
				Data: map[string][]byte{
					secretKeyClientID:     []byte("test-client-id-0"),
					secretKeyClientSecret: []byte("test-client-secret-0"),
					secretKeyTokenURL:     []byte("test-token-url-0"),
					secretKeyCertsURL:     []byte("test-certs-url-0"),
				},
			},
			wantError:        false,
			wantClientID:     []byte("test-client-id-0"),
			wantClientSecret: []byte("test-client-secret-0"),
			wantTokenURL:     []byte("test-token-url-0"),
			wantCertsURL:     []byte("test-certs-url-0"),
		},
	}

	l, e := logger.New("json", "info")
	require.NoError(t, e)

	for _, testcase := range testCases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()

			kubeClient := new(k8smocks.Client)

			if tc.givenSecret != nil {
				kubeClient.On("GetSecret", mock.Anything, mock.Anything).Return(tc.givenSecret, nil).Once()
			} else {
				kubeClient.On("GetSecret", mock.Anything, mock.Anything).Return(nil, errNotFound).Once()
			}

			r := Reconciler{
				kubeClient: kubeClient,
				logger:     l,
				backendConfig: env.BackendConfig{
					EventingWebhookAuthSecretName:      defaultEventingWebhookAuthSecretName,
					EventingWebhookAuthSecretNamespace: defaultEventingWebhookAuthSecretNamespace,
				},
			}

			// when
			credentials, err := r.getOAuth2ClientCredentials(ctx, defaultEventingWebhookAuthSecretNamespace)

			// then
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.Equal(t, tc.wantClientID, credentials.clientID)
			require.Equal(t, tc.wantClientSecret, credentials.clientSecret)
			require.Equal(t, tc.wantTokenURL, credentials.tokenURL)
			require.Equal(t, tc.wantCertsURL, credentials.certsURL)
		})
	}
}

func Test_isOauth2CredentialsInitialized(t *testing.T) {
	testCases := []struct {
		name             string
		givenCredentials *oauth2Credentials
		wantResult       bool
	}{
		{
			name:             "should return false when oauth2credentials are not initialized",
			givenCredentials: nil,
			wantResult:       false,
		},
		{
			name: "should return false when only clientID is initialized",
			givenCredentials: &oauth2Credentials{
				clientID: []byte("test"),
			},
			wantResult: false,
		},
		{
			name: "should return false when only clientSecret is initialized",
			givenCredentials: &oauth2Credentials{
				clientSecret: []byte("test"),
			},
			wantResult: false,
		},
		{
			name: "should return false when only certsURL is initialized",
			givenCredentials: &oauth2Credentials{
				certsURL: []byte("http://kyma-project.io"),
			},
			wantResult: false,
		},
		{
			name: "should return false when only tokenURL is initialized",
			givenCredentials: &oauth2Credentials{
				tokenURL: []byte("http://kyma-project.io"),
			},
			wantResult: false,
		},
		{
			name: "hould return true when oauth2credentials are initialized",
			givenCredentials: &oauth2Credentials{
				clientID:     []byte("test"),
				clientSecret: []byte("test"),
				certsURL:     []byte("http://kyma-project.io"),
				tokenURL:     []byte("http://kyma-project.io"),
			},
			wantResult: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := Reconciler{}
			if tc.givenCredentials != nil {
				r.oauth2credentials = *tc.givenCredentials
			}

			// when
			result := r.isOauth2CredentialsInitialized()

			// then
			require.Equal(t, tc.wantResult, result)
		})
	}
}

func Test_SyncPublisherProxySecret(t *testing.T) {
	testCases := []struct {
		name              string
		givenSecret       *kcorev1.Secret
		mockKubeClient    func() *k8smocks.Client
		wantErr           bool
		wantDesiredSecret *kcorev1.Secret
	}{
		{
			name:        "valid secret",
			givenSecret: utils.NewEventMeshSecret("valid", "test-namespace"),
			mockKubeClient: func() *k8smocks.Client {
				kubeClient := new(k8smocks.Client)
				kubeClient.On("PatchApply", mock.Anything, mock.Anything).Return(nil).Once()
				return kubeClient
			},
			wantErr: false,
		},
		{
			name: "invalid secret",
			givenSecret: &kcorev1.Secret{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"client_id": []byte("test-client-id"),
					// missing client_secret
				},
			},
			mockKubeClient: func() *k8smocks.Client {
				kubeClient := new(k8smocks.Client)
				kubeClient.On("PatchApply", mock.Anything, mock.Anything).Return(nil).Once()
				return kubeClient
			},
			wantErr: true,
		},
		// patchApply error
		{
			name:        "patchApply should fail",
			givenSecret: utils.NewEventMeshSecret("valid", "test-namespace"),
			mockKubeClient: func() *k8smocks.Client {
				kubeClient := new(k8smocks.Client)
				kubeClient.On("PatchApply", mock.Anything, mock.Anything).Return(ErrUseMeInMocks).Once()
				return kubeClient
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			r := &Reconciler{
				kubeClient: tc.mockKubeClient(),
			}

			// when
			_, err := r.SyncPublisherProxySecret(context.Background(), tc.givenSecret)

			// then
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func Test_syncOauth2ClientIDAndSecret(t *testing.T) {
	testCases := []struct {
		name                           string
		givenEventing                  *v1alpha1.Eventing
		givenSecret                    *kcorev1.Secret
		givenCredentials               *oauth2Credentials
		givenSubManagerStarted         bool
		shouldEventMeshSubManagerExist bool
		wantErr                        bool
		wantCredentials                *oauth2Credentials
		wantAssertCheck                bool
	}{
		{
			name: "oauth2 credentials not initialized",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace("test-namespace"),
				utils.WithEventMeshBackend("test-namespace/test-secret-name"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
			),
			givenSecret: &kcorev1.Secret{
				ObjectMeta: kmetav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      defaultEventingWebhookAuthSecretName,
				},
				Data: map[string][]byte{
					secretKeyClientID:     []byte("test-client-id"),
					secretKeyClientSecret: []byte("test-client-secret"),
					secretKeyTokenURL:     []byte("test-token-url"),
					secretKeyCertsURL:     []byte("test-certs-url"),
				},
			},
			givenSubManagerStarted:         true,
			shouldEventMeshSubManagerExist: true,
			wantCredentials: &oauth2Credentials{
				clientID:     []byte("test-client-id"),
				clientSecret: []byte("test-client-secret"),
				tokenURL:     []byte("test-token-url"),
				certsURL:     []byte("test-certs-url"),
			},
		},
		{
			name: "oauth2 credentials changed",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace("test-namespace"),
				utils.WithEventMeshBackend("test-namespace/test-secret-name"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
			),
			givenSecret: &kcorev1.Secret{
				ObjectMeta: kmetav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      defaultEventingWebhookAuthSecretName,
				},
				Data: map[string][]byte{
					secretKeyClientID:     []byte("test-client-id-changed"),
					secretKeyClientSecret: []byte("test-client-secret-changed"),
					secretKeyTokenURL:     []byte("test-token-url-changed"),
					secretKeyCertsURL:     []byte("test-certs-url-changed"),
				},
			},
			givenCredentials: &oauth2Credentials{
				clientID:     []byte("test-client-id"),
				clientSecret: []byte("test-client-secret"),
				tokenURL:     []byte("test-token-url"),
				certsURL:     []byte("test-certs-url"),
			},
			givenSubManagerStarted:         false,
			shouldEventMeshSubManagerExist: false,
			wantErr:                        false,
			wantCredentials: &oauth2Credentials{
				clientID:     []byte("test-client-id-changed"),
				clientSecret: []byte("test-client-secret-changed"),
				tokenURL:     []byte("test-token-url-changed"),
				certsURL:     []byte("test-certs-url-changed"),
			},
			wantAssertCheck: true,
		},
		{
			name: "no change in oauth2 credentials",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace("test-namespace"),
				utils.WithEventMeshBackend("test-namespace/test-secret-name"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
			),
			givenSecret: &kcorev1.Secret{
				ObjectMeta: kmetav1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      defaultEventingWebhookAuthSecretName,
				},
				Data: map[string][]byte{
					secretKeyClientID:     []byte("test-client-id"),
					secretKeyClientSecret: []byte("test-client-secret"),
					secretKeyTokenURL:     []byte("test-token-url"),
					secretKeyCertsURL:     []byte("test-certs-url"),
				},
			},
			givenCredentials: &oauth2Credentials{
				clientID:     []byte("test-client-id"),
				clientSecret: []byte("test-client-secret"),
				tokenURL:     []byte("test-token-url"),
				certsURL:     []byte("test-certs-url"),
			},
			givenSubManagerStarted:         true,
			shouldEventMeshSubManagerExist: true,
			wantErr:                        false,
			wantCredentials: &oauth2Credentials{
				clientID:     []byte("test-client-id"),
				clientSecret: []byte("test-client-secret"),
				tokenURL:     []byte("test-token-url"),
				certsURL:     []byte("test-certs-url"),
			},
		},
		{
			name: "oauth2 credentials not found",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRName("test-eventing"),
				utils.WithEventingCRNamespace("test-namespace"),
				utils.WithEventMeshBackend("beb"),
			),
			givenSubManagerStarted:         false,
			shouldEventMeshSubManagerExist: false,
			wantErr:                        true,
			wantAssertCheck:                true,
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t, tc.givenEventing)
			testEnv.Reconciler.oauth2credentials = oauth2Credentials{}
			eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
			eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
			testEnv.Reconciler.eventMeshSubManager = eventMeshSubManagerMock
			testEnv.Reconciler.backendConfig = env.BackendConfig{
				EventingWebhookAuthSecretNamespace: tc.givenEventing.Namespace,
				EventingWebhookAuthSecretName:      defaultEventingWebhookAuthSecretName,
			}
			testEnv.Reconciler.isEventMeshSubManagerStarted = true

			if tc.givenSecret != nil {
				require.NoError(t, testEnv.Reconciler.Client.Create(ctx, tc.givenSecret))
			}

			if tc.givenCredentials != nil {
				testEnv.Reconciler.oauth2credentials.clientID = tc.givenCredentials.clientID
				testEnv.Reconciler.oauth2credentials.clientSecret = tc.givenCredentials.clientSecret
				testEnv.Reconciler.oauth2credentials.tokenURL = tc.givenCredentials.tokenURL
				testEnv.Reconciler.oauth2credentials.certsURL = tc.givenCredentials.certsURL
			}
			// when
			err := testEnv.Reconciler.syncOauth2ClientIDAndSecret(ctx, tc.givenEventing)

			// then
			if tc.wantAssertCheck {
				eventMeshSubManagerMock.AssertExpectations(t)
			}
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.givenSubManagerStarted, testEnv.Reconciler.isEventMeshSubManagerStarted)
			require.Equal(t, tc.shouldEventMeshSubManagerExist, testEnv.Reconciler.eventMeshSubManager != nil)
			require.Equal(t, *tc.wantCredentials, testEnv.Reconciler.oauth2credentials)
		})
	}
}
