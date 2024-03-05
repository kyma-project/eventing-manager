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
	ErrFailedToStart        = errors.New("failed to start")
	ErrFailedToStop         = errors.New("failed to stop")
	ErrFailedToRemove       = errors.New("failed to remove")
	errNotFound             = errors.New("secret not found")
	ErrFailedToApplyPatch   = errors.New("failed to apply patch")
	ErrFailedToSyncPPSecret = errors.New("failed to sync Publisher Proxy secret: failed to apply patch")
)

func Test_reconcileEventMeshSubManager(t *testing.T) {
	t.Parallel()

	// given - common for all test cases.

	givenNamespace := "test-namespace"
	givenOauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", givenNamespace)

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
					utils.NewOAuthSecret("test-secret", givenNamespace), nil).Once()
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(ErrFailedToApplyPatch).Once()
				return mockKubeClient
			},
			wantError: ErrFailedToSyncPPSecret,
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
					utils.NewOAuthSecret("test-secret", givenNamespace), nil).Once()
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
					utils.NewOAuthSecret("test-secret", givenNamespace), nil).Once()
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
				eventMeshSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(ErrFailedToStart).Twice()
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
					utils.NewOAuthSecret("test-secret", givenNamespace), nil).Twice()
				return mockKubeClient
			},
			wantAssertCheck:  true,
			givenShouldRetry: true,
			wantError:        ErrFailedToStart,
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
					utils.NewOAuthSecret("test-secret", givenNamespace), nil).Twice()
				return mockKubeClient
			},
			wantAssertCheck: true,
			wantHashAfter:   int64(4922936597877296700),
		},
	}

	// run test cases
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			givenEventing := utils.NewEventingCR(
				utils.WithEventingCRName("eventing"),
				utils.WithEventingCRNamespace(givenNamespace),
				utils.WithEventMeshBackend("test-namespace/test-secret-name"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			)
			// given
			testEnv := NewMockedUnitTestEnvironment(t, givenEventing, givenOauthSecret)
			testEnv.Reconciler.backendConfig = *givenBackendConfig

			// get mocks from test-case.
			givenEventMeshSubManagerMock := testcase.givenEventMeshSubManagerMock()
			givenManagerFactoryMock := testcase.givenManagerFactoryMock(givenEventMeshSubManagerMock)
			givenEventingManagerMock := testcase.givenEventingManagerMock()

			// connect mocks with reconciler.
			if testcase.givenKubeClientMock != nil {
				testEnv.Reconciler.kubeClient = testcase.givenKubeClientMock()
			}

			testEnv.Reconciler.isEventMeshSubManagerStarted = testcase.givenIsEventMeshSubManagerStarted
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.eventMeshSubManager = nil
			if givenManagerFactoryMock == nil || testcase.givenUpdateTest {
				testEnv.Reconciler.eventMeshSubManager = givenEventMeshSubManagerMock
			}

			// set the backend hash before depending on test
			givenEventing.Status.BackendConfigHash = testcase.givenHashBefore

			// when

			eventMeshSecret := utils.NewEventMeshSecret("eventing-backend", givenEventing.Namespace)

			err := testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing, eventMeshSecret)
			if err != nil && testcase.givenShouldRetry {
				// This is to test the scenario where initialization of eventMeshSubManager was successful but
				// starting the eventMeshSubManager failed. So on next try it should again try to start the eventMeshSubManager.
				err = testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing, eventMeshSecret)
			}
			if testcase.givenUpdateTest {
				// Run reconcile again with newBackendConfig:
				err = testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing, eventMeshSecret)
				require.NoError(t, err)
			}

			// check for backend hash after
			require.Equal(t, testcase.wantHashAfter, givenEventing.Status.BackendConfigHash)

			// then
			if testcase.wantError != nil {
				require.Error(t, err)
				require.ErrorAs(t, err, &testcase.wantError)
			} else {
				require.NoError(t, err)
				require.NotNil(t, testEnv.Reconciler.eventMeshSubManager)
				require.True(t, testEnv.Reconciler.isEventMeshSubManagerStarted)
			}

			if testcase.wantAssertCheck {
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			testEnv := NewMockedUnitTestEnvironment(t, testcase.givenEventing, givenOauthSecret)
			testEnv.Reconciler.backendConfig = *givenBackendConfig

			givenEventMeshSubManagerMock := testcase.givenEventMeshSubManagerMock()
			givenManagerFactoryMock := testcase.givenManagerFactoryMock(givenEventMeshSubManagerMock)
			givenEventingManagerMock := testcase.givenEventingManagerMock()

			var mockClient *k8smocks.Client
			if testcase.givenKubeClientMock != nil {
				testEnv.Reconciler.kubeClient, mockClient = testcase.givenKubeClientMock()
			}

			testEnv.Reconciler.isEventMeshSubManagerStarted = true
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.eventMeshSubManager = nil

			// when
			eventMeshSecret := utils.NewEventMeshSecret("test-secret", testcase.givenEventing.Namespace)
			err := testEnv.Reconciler.reconcileEventMeshSubManager(ctx, testcase.givenEventing, eventMeshSecret)

			// then
			require.NoError(t, err)
			givenEventMeshSubManagerMock.AssertExpectations(t)
			givenEventingManagerMock.AssertExpectations(t)
			givenManagerFactoryMock.AssertExpectations(t)
			mockClient.AssertExpectations(t)
		})
	}
}

//nolint:dupl // quite similar to the natsmanager test
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			// get mocks from test-case.
			givenEventMeshSubManagerMock := testcase.givenEventMeshSubManagerMock()

			// connect mocks with reconciler.
			testEnv.Reconciler.eventMeshSubManager = givenEventMeshSubManagerMock
			testEnv.Reconciler.isEventMeshSubManagerStarted = testcase.givenIsEventMeshSubManagerStarted

			// when
			err := testEnv.Reconciler.stopEventMeshSubManager(true, logger)
			// then
			if testcase.wantError == nil {
				require.NoError(t, err)
				require.Nil(t, testEnv.Reconciler.eventMeshSubManager)
				require.False(t, testEnv.Reconciler.isEventMeshSubManagerStarted)
			} else {
				require.Equal(t, testcase.wantError.Error(), err.Error())
			}

			if testcase.wantAssertCheck {
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			publisherSecret := secretFor(testcase.messagingData, testcase.namespaceData)

			gotPublisherSecret, err := getSecretForPublisher(publisherSecret)
			if testcase.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrEventMeshSecretMalformatted)
				require.ErrorContains(t, err, tc.expectedError.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, testcase.expectedSecret, gotPublisherSecret, "invalid publisher secret")
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

	infoLog, err := logger.New("json", "info")
	require.NoError(t, err)

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			ctx := context.Background()

			kubeClient := new(k8smocks.Client)

			if testcase.givenSecret != nil {
				kubeClient.On("GetSecret", mock.Anything, mock.Anything).Return(testcase.givenSecret, nil).Once()
			} else {
				kubeClient.On("GetSecret", mock.Anything, mock.Anything).Return(nil, errNotFound).Once()
			}

			reconciler := Reconciler{
				kubeClient: kubeClient,
				logger:     infoLog,
				backendConfig: env.BackendConfig{
					EventingWebhookAuthSecretName:      defaultEventingWebhookAuthSecretName,
					EventingWebhookAuthSecretNamespace: defaultEventingWebhookAuthSecretNamespace,
				},
			}

			// when
			credentials, err := reconciler.getOAuth2ClientCredentials(ctx, defaultEventingWebhookAuthSecretNamespace)

			// then
			if testcase.wantError {
				require.Error(t, err)
				return
			}
			require.Equal(t, testcase.wantClientID, credentials.clientID)
			require.Equal(t, testcase.wantClientSecret, credentials.clientSecret)
			require.Equal(t, testcase.wantTokenURL, credentials.tokenURL)
			require.Equal(t, testcase.wantCertsURL, credentials.certsURL)
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			reconciler := Reconciler{}
			if testcase.givenCredentials != nil {
				reconciler.oauth2credentials = *testcase.givenCredentials
			}

			// when
			result := reconciler.isOauth2CredentialsInitialized()

			// then
			require.Equal(t, testcase.wantResult, result)
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			r := &Reconciler{
				kubeClient: testcase.mockKubeClient(),
			}

			// when
			_, err := r.SyncPublisherProxySecret(context.Background(), testcase.givenSecret)

			// then
			if testcase.wantErr {
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			testEnv := NewMockedUnitTestEnvironment(t, testcase.givenEventing)
			testEnv.Reconciler.oauth2credentials = oauth2Credentials{}
			eventMeshSubManagerMock := new(submgrmanagermocks.Manager)
			eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
			testEnv.Reconciler.eventMeshSubManager = eventMeshSubManagerMock
			testEnv.Reconciler.backendConfig = env.BackendConfig{
				EventingWebhookAuthSecretNamespace: testcase.givenEventing.Namespace,
				EventingWebhookAuthSecretName:      defaultEventingWebhookAuthSecretName,
			}
			testEnv.Reconciler.isEventMeshSubManagerStarted = true

			if testcase.givenSecret != nil {
				require.NoError(t, testEnv.Reconciler.Client.Create(ctx, testcase.givenSecret))
			}

			if testcase.givenCredentials != nil {
				testEnv.Reconciler.oauth2credentials.clientID = testcase.givenCredentials.clientID
				testEnv.Reconciler.oauth2credentials.clientSecret = testcase.givenCredentials.clientSecret
				testEnv.Reconciler.oauth2credentials.tokenURL = testcase.givenCredentials.tokenURL
				testEnv.Reconciler.oauth2credentials.certsURL = testcase.givenCredentials.certsURL
			}
			// when
			err := testEnv.Reconciler.syncOauth2ClientIDAndSecret(ctx, testcase.givenEventing)

			// then
			if testcase.wantAssertCheck {
				eventMeshSubManagerMock.AssertExpectations(t)
			}
			if testcase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testcase.givenSubManagerStarted, testEnv.Reconciler.isEventMeshSubManagerStarted)
			require.Equal(t, testcase.shouldEventMeshSubManagerExist, testEnv.Reconciler.eventMeshSubManager != nil)
			require.Equal(t, *testcase.wantCredentials, testEnv.Reconciler.oauth2credentials)
		})
	}
}

// Test_IsMalfromattedSecret verifies that the function IsMalformattedSecretErr asses correctly
// if a given error is a malformatted secret error.
func Test_IsMalfromattedSecret(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		givenErr   error
		wantResult bool
	}{
		{
			name:       "should return true when error is ErrMalformedSecret",
			givenErr:   ErrEventMeshSecretMalformatted,
			wantResult: true,
		}, {
			name:       "should return true when error is a wrapped ErrMalformedSecret",
			givenErr:   newMalformattedSecretErr("this error will wrap ErrMalformedSecret"),
			wantResult: true,
		}, {
			name:       "should return false when error is not ErrMalformedSecret",
			givenErr:   fmt.Errorf("this is not a malformed secret error"),
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Parallel()
		t.Run(tc.name, func(t *testing.T) {
			// when
			result := IsMalformattedSecretErr(tc.givenErr)

			// then
			require.Equal(t, tc.wantResult, result)
		})
	}
}
