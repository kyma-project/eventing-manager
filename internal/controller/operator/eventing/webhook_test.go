package eventing

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/pkg/env"
)

func Test_ReconcileWebhooksWithCABundle(t *testing.T) {
	t.Parallel()
	// given
	ctx := context.Background()
	dummyCABundle := make([]byte, 20)
	_, err := rand.Read(dummyCABundle)
	require.NoError(t, err)
	newCABundle := make([]byte, 20)
	_, err = rand.Read(newCABundle)
	require.NoError(t, err)

	testCases := []struct {
		name             string
		givenObjects     []client.Object
		wantMutatingWH   *admissionv1.MutatingWebhookConfiguration
		wantValidatingWH *admissionv1.ValidatingWebhookConfiguration
		wantError        error
	}{
		{
			name:      "secret does not exist",
			wantError: errObjectNotFound,
		},
		{
			name: "secret exits but mutatingWH does not exist",
			givenObjects: []client.Object{
				getSecretWithTLSSecret(nil),
			},
			wantError: errObjectNotFound,
		},
		{
			name: "mutatingWH exists, validatingWH does not exist",
			givenObjects: []client.Object{
				getSecretWithTLSSecret(nil),
				getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{},
					},
				}),
			},
			wantError: errObjectNotFound,
		},
		{
			name: "mutatingWH, validatingWH exists but does not contain webhooks",
			givenObjects: []client.Object{
				getSecretWithTLSSecret(nil),
				getMutatingWebhookConfig(nil),
				getValidatingWebhookConfig(nil),
			},
			wantError: errInvalidObject,
		},
		{
			name: "validatingWH does not contain webhooks",
			givenObjects: []client.Object{
				getSecretWithTLSSecret(nil),
				getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{},
					},
				}),
				getValidatingWebhookConfig(nil),
			},
			wantError: errInvalidObject,
		},
		{
			name: "WHs do not contain valid CABundle",
			givenObjects: []client.Object{
				getSecretWithTLSSecret(dummyCABundle),
				getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{},
					},
				}),
				getValidatingWebhookConfig([]admissionv1.ValidatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{},
					},
				}),
			},
			wantMutatingWH: getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
				{
					ClientConfig: admissionv1.WebhookClientConfig{
						CABundle: dummyCABundle,
					},
				},
			}),
			wantValidatingWH: getValidatingWebhookConfig([]admissionv1.ValidatingWebhook{
				{
					ClientConfig: admissionv1.WebhookClientConfig{
						CABundle: dummyCABundle,
					},
				},
			}),
			wantError: nil,
		},
		{
			name: "WHs contains valid CABundle",
			givenObjects: []client.Object{
				getSecretWithTLSSecret(dummyCABundle),
				getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
				getValidatingWebhookConfig([]admissionv1.ValidatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
			},
			wantMutatingWH: getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
				{
					ClientConfig: admissionv1.WebhookClientConfig{
						CABundle: dummyCABundle,
					},
				},
			}),
			wantValidatingWH: getValidatingWebhookConfig([]admissionv1.ValidatingWebhook{
				{
					ClientConfig: admissionv1.WebhookClientConfig{
						CABundle: dummyCABundle,
					},
				},
			}),
			wantError: nil,
		},
		{
			name: "WHs contains outdated valid CABundle",
			givenObjects: []client.Object{
				getSecretWithTLSSecret(newCABundle),
				getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
				getValidatingWebhookConfig([]admissionv1.ValidatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
			},
			wantMutatingWH: getMutatingWebhookConfig([]admissionv1.MutatingWebhook{
				{
					ClientConfig: admissionv1.WebhookClientConfig{
						CABundle: newCABundle,
					},
				},
			}),
			wantValidatingWH: getValidatingWebhookConfig([]admissionv1.ValidatingWebhook{
				{
					ClientConfig: admissionv1.WebhookClientConfig{
						CABundle: newCABundle,
					},
				},
			}),
			wantError: nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			testEnv := NewMockedUnitTestEnvironment(t, tc.givenObjects...)
			testEnv.Reconciler.backendConfig = getTestBackendConfig()

			// when
			err := testEnv.Reconciler.reconcileWebhooksWithCABundle(ctx)

			// then
			require.ErrorIs(t, err, tc.wantError)
			if tc.wantError == nil {
				mutatingWH, newErr := testEnv.Reconciler.kubeClient.GetMutatingWebHookConfiguration(ctx,
					testEnv.Reconciler.backendConfig.MutatingWebhookName)
				require.NoError(t, newErr)
				validatingWH, newErr := testEnv.Reconciler.kubeClient.GetValidatingWebHookConfiguration(ctx,
					testEnv.Reconciler.backendConfig.ValidatingWebhookName)
				require.NoError(t, newErr)
				require.Equal(t, mutatingWH.Webhooks[0], tc.wantMutatingWH.Webhooks[0])
				require.Equal(t, validatingWH.Webhooks[0], tc.wantValidatingWH.Webhooks[0])
			}
		})
	}
}

func getSecretWithTLSSecret(dummyCABundle []byte) *kcorev1.Secret {
	return &kcorev1.Secret{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      getTestBackendConfig().WebhookSecretName,
			Namespace: getTestBackendConfig().Namespace,
		},
		Data: map[string][]byte{
			TLSCertField: dummyCABundle,
		},
	}
}

func getMutatingWebhookConfig(webhook []admissionv1.MutatingWebhook) *admissionv1.MutatingWebhookConfiguration {
	return &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: getTestBackendConfig().MutatingWebhookName,
		},
		Webhooks: webhook,
	}
}

func getValidatingWebhookConfig(webhook []admissionv1.ValidatingWebhook) *admissionv1.ValidatingWebhookConfiguration {
	return &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: getTestBackendConfig().ValidatingWebhookName,
		},
		Webhooks: webhook,
	}
}

func getTestBackendConfig() env.BackendConfig {
	return env.BackendConfig{
		WebhookSecretName:     "webhookSecret",
		MutatingWebhookName:   "mutatingWH",
		ValidatingWebhookName: "validatingWH",
		Namespace:             "kyma-system",
	}
}
