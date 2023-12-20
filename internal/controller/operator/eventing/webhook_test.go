package eventing

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	kadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
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
		wantMutatingWH   *kadmissionregistrationv1.MutatingWebhookConfiguration
		wantValidatingWH *kadmissionregistrationv1.ValidatingWebhookConfiguration
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
				getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{},
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
				getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{},
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
				getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{},
					},
				}),
				getValidatingWebhookConfig([]kadmissionregistrationv1.ValidatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{},
					},
				}),
			},
			wantMutatingWH: getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
				{
					ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
						CABundle: dummyCABundle,
					},
				},
			}),
			wantValidatingWH: getValidatingWebhookConfig([]kadmissionregistrationv1.ValidatingWebhook{
				{
					ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
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
				getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
				getValidatingWebhookConfig([]kadmissionregistrationv1.ValidatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
			},
			wantMutatingWH: getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
				{
					ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
						CABundle: dummyCABundle,
					},
				},
			}),
			wantValidatingWH: getValidatingWebhookConfig([]kadmissionregistrationv1.ValidatingWebhook{
				{
					ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
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
				getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
				getValidatingWebhookConfig([]kadmissionregistrationv1.ValidatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
							CABundle: dummyCABundle,
						},
					},
				}),
			},
			wantMutatingWH: getMutatingWebhookConfig([]kadmissionregistrationv1.MutatingWebhook{
				{
					ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
						CABundle: newCABundle,
					},
				},
			}),
			wantValidatingWH: getValidatingWebhookConfig([]kadmissionregistrationv1.ValidatingWebhook{
				{
					ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
						CABundle: newCABundle,
					},
				},
			}),
			wantError: nil,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			testEnv := NewMockedUnitTestEnvironment(t, testcase.givenObjects...)
			testEnv.Reconciler.backendConfig = getTestBackendConfig()

			// when
			err := testEnv.Reconciler.reconcileWebhooksWithCABundle(ctx)

			// then
			require.ErrorIs(t, err, testcase.wantError)
			if testcase.wantError == nil {
				mutatingWH, newErr := testEnv.Reconciler.kubeClient.GetMutatingWebHookConfiguration(ctx,
					testEnv.Reconciler.backendConfig.MutatingWebhookName)
				require.NoError(t, newErr)
				validatingWH, newErr := testEnv.Reconciler.kubeClient.GetValidatingWebHookConfiguration(ctx,
					testEnv.Reconciler.backendConfig.ValidatingWebhookName)
				require.NoError(t, newErr)
				require.Equal(t, mutatingWH.Webhooks[0], testcase.wantMutatingWH.Webhooks[0])
				require.Equal(t, validatingWH.Webhooks[0], testcase.wantValidatingWH.Webhooks[0])
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

func getMutatingWebhookConfig(webhook []kadmissionregistrationv1.MutatingWebhook) *kadmissionregistrationv1.MutatingWebhookConfiguration {
	return &kadmissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: getTestBackendConfig().MutatingWebhookName,
		},
		Webhooks: webhook,
	}
}

func getValidatingWebhookConfig(webhook []kadmissionregistrationv1.ValidatingWebhook) *kadmissionregistrationv1.ValidatingWebhookConfiguration {
	return &kadmissionregistrationv1.ValidatingWebhookConfiguration{
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
