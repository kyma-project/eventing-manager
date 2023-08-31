package eventing

import (
	"context"
	"errors"
	"testing"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"

	"github.com/kyma-project/eventing-manager/pkg/env"
	managermocks "github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	subscriptionmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks"
	ecsubmanagermocks "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/mocks/ec"
	"github.com/kyma-project/eventing-manager/test/utils"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	defaultEventingWebhookAuthSecretName      = "eventing-webhook-auth"
	defaultEventingWebhookAuthSecretNamespace = "kyma-system"
)

func Test_reconcileEventMeshSubManager(t *testing.T) {
	t.Parallel()

	// given - common for all test cases.
	givenEventing := utils.NewEventingCR(
		utils.WithEventingCRNamespace("test-namespace"),
		utils.WithEventMeshBackend("test-namespace/test-secret-name"),
		utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
		utils.WithEventingEventTypePrefix("test-prefix"),
	)

	givenBackendConfig := &env.BackendConfig{
		EventingWebhookAuthSecretName: "eventing-webhook-auth",
	}
	ctx := context.Background()

	// define test cases
	testCases := []struct {
		name                              string
		givenIsEventMeshSubManagerStarted bool
		givenShouldRetry                  bool
		givenEventMeshSubManagerMock      func() *ecsubmanagermocks.Manager
		givenEventingManagerMock          func() *managermocks.Manager
		givenManagerFactoryMock           func(*ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory
		givenClientMock                   func() client.Client
		givenKubeClientMock               func() k8s.Client
		wantAssertCheck                   bool
		wantError                         error
	}{
		{
			name:                              "it should do nothing because syncing OAuth secret failed",
			givenIsEventMeshSubManagerStarted: true,
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
				eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				return nil
			},
			givenManagerFactoryMock: func(_ *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				return nil
			},
			wantError: errors.New("failed to sync OAuth secret"),
		},
		{
			name:                              "it should do nothing because failed syncing EventMesh secret",
			givenIsEventMeshSubManagerStarted: true,
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
				eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				return nil
			},
			givenManagerFactoryMock: func(_ *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				return nil
			},
			givenClientMock: func() client.Client {
				mockClient := fake.NewClientBuilder().WithObjects().Build()
				oauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", givenEventing.Namespace)
				require.NoError(t, mockClient.Create(ctx, oauthSecret))
				return mockClient
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(nil,
					errors.New("failed getting secret")).Once()
				return mockKubeClient
			},
			wantError: errors.New("failed to get EventMesh secret"),
		},
		{
			name:                              "it should do nothing because failed sync Publisher Proxy secret",
			givenIsEventMeshSubManagerStarted: true,
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
				eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				return nil
			},
			givenManagerFactoryMock: func(_ *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				return nil
			},
			givenClientMock: func() client.Client {
				mockClient := fake.NewClientBuilder().WithObjects().Build()
				oauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", givenEventing.Namespace)
				require.NoError(t, mockClient.Create(ctx, oauthSecret))
				return mockClient
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewEventMeshSecret("test-secret", givenEventing.Namespace), nil).Once()
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(errors.New("failed to apply patch")).Once()
				return mockKubeClient
			},
			wantError: errors.New("failed to sync Publisher Proxy secret"),
		},
		{
			name:                              "it should do nothing because subscription manager is already started",
			givenIsEventMeshSubManagerStarted: true,
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
				eventMeshSubManagerMock.On("Stop", mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				return nil
			},
			givenManagerFactoryMock: func(_ *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				return nil
			},
			givenClientMock: func() client.Client {
				mockClient := fake.NewClientBuilder().WithObjects().Build()
				oauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", givenEventing.Namespace)
				require.NoError(t, mockClient.Create(ctx, oauthSecret))
				return mockClient
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Once()
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewEventMeshSecret("test-secret", givenEventing.Namespace), nil).Once()
				return mockKubeClient
			},
		},
		{
			name: "it should initialize and start subscription manager because " +
				"subscription manager is not started",
			givenIsEventMeshSubManagerStarted: false,
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
				eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				eventMeshSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(nil).Once()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				emMock := new(managermocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenManagerFactoryMock: func(subManager *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				subManagerFactoryMock := new(subscriptionmanagermocks.ManagerFactory)
				subManagerFactoryMock.On("NewEventMeshManager", mock.Anything).Return(subManager, nil).Once()
				return subManagerFactoryMock
			},
			givenClientMock: func() client.Client {
				mockClient := fake.NewClientBuilder().WithObjects().Build()
				oauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", givenEventing.Namespace)
				require.NoError(t, mockClient.Create(ctx, oauthSecret))
				return mockClient
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Once()
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewEventMeshSecret("test-secret", givenEventing.Namespace), nil).Once()
				return mockKubeClient
			},
			wantAssertCheck: true,
		},
		{
			name: "it should retry to start subscription manager when subscription manager was " +
				"successfully initialized but failed to start",
			givenIsEventMeshSubManagerStarted: false,
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
				eventMeshSubManagerMock.On("Init", mock.Anything).Return(nil).Once()
				eventMeshSubManagerMock.On("Start", mock.Anything, mock.Anything).Return(errors.New("failed to start")).Twice()
				return eventMeshSubManagerMock
			},
			givenEventingManagerMock: func() *managermocks.Manager {
				emMock := new(managermocks.Manager)
				emMock.On("GetBackendConfig").Return(givenBackendConfig)
				return emMock
			},
			givenManagerFactoryMock: func(subManager *ecsubmanagermocks.Manager) *subscriptionmanagermocks.ManagerFactory {
				subManagerFactoryMock := new(subscriptionmanagermocks.ManagerFactory)
				subManagerFactoryMock.On("NewEventMeshManager", mock.Anything).Return(subManager, nil).Once()
				return subManagerFactoryMock
			},
			givenClientMock: func() client.Client {
				mockClient := fake.NewClientBuilder().WithObjects().Build()
				oauthSecret := utils.NewOAuthSecret("eventing-webhook-auth", givenEventing.Namespace)
				require.NoError(t, mockClient.Create(ctx, oauthSecret))
				return mockClient
			},
			givenKubeClientMock: func() k8s.Client {
				mockKubeClient := new(k8smocks.Client)
				mockKubeClient.On("PatchApply", ctx, mock.Anything).Return(nil).Twice()
				mockKubeClient.On("GetSecret", ctx, mock.Anything, mock.Anything).Return(
					utils.NewEventMeshSecret("test-secret", givenEventing.Namespace), nil).Twice()
				return mockKubeClient
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
			testEnv.Reconciler.backendConfig = *givenBackendConfig
			// get mocks from test-case.
			givenEventMeshSubManagerMock := tc.givenEventMeshSubManagerMock()
			givenManagerFactoryMock := tc.givenManagerFactoryMock(givenEventMeshSubManagerMock)
			givenEventingManagerMock := tc.givenEventingManagerMock()

			// connect mocks with reconciler.

			if tc.givenKubeClientMock != nil {
				testEnv.Reconciler.kubeClient = tc.givenKubeClientMock()
			}
			if tc.givenClientMock != nil {
				testEnv.Reconciler.Client = tc.givenClientMock()
			}

			testEnv.Reconciler.isEventMeshSubManagerStarted = tc.givenIsEventMeshSubManagerStarted
			testEnv.Reconciler.eventingManager = givenEventingManagerMock
			testEnv.Reconciler.subManagerFactory = givenManagerFactoryMock
			testEnv.Reconciler.eventMeshSubManager = nil
			if givenManagerFactoryMock == nil {
				testEnv.Reconciler.eventMeshSubManager = givenEventMeshSubManagerMock
			}

			// when
			err := testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing)
			if err != nil && tc.givenShouldRetry {
				// This is to test the scenario where initialization of eventMeshSubManager was successful but
				// starting the eventMeshSubManager failed. So on next try it should again try to start the eventMeshSubManager.
				err = testEnv.Reconciler.reconcileEventMeshSubManager(ctx, givenEventing)
			}

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.ErrorAs(t, err, &tc.wantError)
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

func Test_stopEventMeshSubManager(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                              string
		givenEventMeshSubManagerMock      func() *ecsubmanagermocks.Manager
		givenIsEventMeshSubManagerStarted bool
		wantError                         error
		wantAssertCheck                   bool
	}{
		{
			name: "should do nothing when subscription manager is not initialised",
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				return nil
			},
			givenIsEventMeshSubManagerStarted: false,
			wantError:                         nil,
		},
		{
			name: "should return error when subscription manager fails to stop",
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
				managerMock.On("Stop", mock.Anything).Return(errors.New("failed to stop")).Once()
				return managerMock
			},
			givenIsEventMeshSubManagerStarted: true,
			wantError:                         errors.New("failed to stop"),
			wantAssertCheck:                   true,
		},
		{
			name: "should succeed to stop subscription manager",
			givenEventMeshSubManagerMock: func() *ecsubmanagermocks.Manager {
				managerMock := new(ecsubmanagermocks.Manager)
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
	secretFor := func(message, namespace []byte) *corev1.Secret {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.PublisherName,
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
		expectedSecret corev1.Secret
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
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deployment.PublisherName,
					Namespace: "test-namespace",
					Labels: map[string]string{
						deployment.AppLabelKey: deployment.PublisherName,
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
			expectedError: errors.New("message is missing from BEB secret"),
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
			expectedError: errors.New("namespace is missing from BEB secret"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			publisherSecret := secretFor(tc.messagingData, tc.namespaceData)

			gotPublisherSecret, err := getSecretForPublisher(publisherSecret)
			if tc.expectedError != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expectedError.Error(), err.Error(), "invalid error")
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedSecret, *gotPublisherSecret, "invalid publisher secret")
		})
	}
}

func Test_getOAuth2ClientCredentials(t *testing.T) {
	testCases := []struct {
		name             string
		givenSecrets     []*corev1.Secret
		wantError        bool
		wantClientID     []byte
		wantClientSecret []byte
		wantTokenURL     []byte
		wantCertsURL     []byte
	}{
		{
			name:         "secret does not exist",
			givenSecrets: nil,
			wantError:    true,
		},
		{
			name: "secret exists with missing data",
			givenSecrets: []*corev1.Secret{
				// required secret
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultEventingWebhookAuthSecretName,
						Namespace: defaultEventingWebhookAuthSecretNamespace,
					},
					Data: map[string][]byte{
						secretKeyClientID: []byte("test-client-id-0"),
						// missing data
					},
				},
			},
			wantError: true,
		},
		{
			name: "secret exists with all data",
			givenSecrets: []*corev1.Secret{
				// required secret
				{
					ObjectMeta: metav1.ObjectMeta{
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
			r := Reconciler{
				Client: fake.NewClientBuilder().WithObjects().Build(),
				logger: l,
				backendConfig: env.BackendConfig{
					EventingWebhookAuthSecretName:      defaultEventingWebhookAuthSecretName,
					EventingWebhookAuthSecretNamespace: defaultEventingWebhookAuthSecretNamespace,
				},
			}
			if len(tc.givenSecrets) > 0 {
				for _, secret := range tc.givenSecrets {
					err := r.Client.Create(ctx, secret)
					require.NoError(t, err)
				}
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
		givenSecret       *corev1.Secret
		mockKubeClient    func() *k8smocks.Client
		wantErr           bool
		wantDesiredSecret *corev1.Secret
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
			givenSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
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
			name:        "pathApply should fail",
			givenSecret: utils.NewEventMeshSecret("valid", "test-namespace"),
			mockKubeClient: func() *k8smocks.Client {
				kubeClient := new(k8smocks.Client)
				kubeClient.On("PatchApply", mock.Anything, mock.Anything).Return(errors.New("fake error")).Once()
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
		givenSecret                    *corev1.Secret
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
			givenSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
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
			givenSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
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
			givenSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
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
			testEnv := NewMockedUnitTestEnvironment(t)
			testEnv.Reconciler.oauth2credentials = oauth2Credentials{}
			eventMeshSubManagerMock := new(ecsubmanagermocks.Manager)
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

			if tc.givenEventing != nil {
				require.NoError(t, testEnv.Reconciler.Client.Create(ctx, tc.givenEventing))
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
