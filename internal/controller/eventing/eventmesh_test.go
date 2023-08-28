package eventing

import (
	"context"
	"errors"
	"testing"

	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetSecretForPublisher verifies the successful and failing retrieval
// of secrets.
func TestGetSecretForPublisher(t *testing.T) {
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
	const (
		defaultWebhookTokenEndpoint               = "http://domain.com/token"
		defaultEventingWebhookAuthSecretName      = "eventing-webhook-auth"
		defaultEventingWebhookAuthSecretNamespace = "kyma-system"
	)

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
				cfg: env.BackendConfig{
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
			name:             "should return false when credentials are not initialized",
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
			name: "hould return true when credentials are initialized",
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
				r.credentials = *tc.givenCredentials
			}

			// when
			result := r.isOauth2CredentialsInitialized()

			// then
			require.Equal(t, tc.wantResult, result)
		})
	}
}
