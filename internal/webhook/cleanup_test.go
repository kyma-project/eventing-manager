package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	kadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	kbatchv1 "k8s.io/api/batch/v1"
	kcorev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCleanupResources(t *testing.T) {
	// given
	const (
		testNamespace                      = "test-namespace"
		testService                        = "test-service"
		testCronjob                        = "test-cron-job"
		testJob                            = "test-job"
		testMutatingWebhookConfiguration   = "test-mutating-webhook-configuration"
		testValidatingWebhookConfiguration = "test-validating-webhook-configuration"
	)

	var (
		// webhook resources
		serviceObject = kcorev1.Service{
			ObjectMeta: kmetav1.ObjectMeta{
				Namespace: namespace,
				Name:      service,
			},
		}
		cronjobObject = kbatchv1.CronJob{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      cronjob,
				Namespace: namespace,
			},
		}
		jobObject = kbatchv1.Job{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      job,
				Namespace: namespace,
			},
		}
		mutatingWebhookConfigurationObject = kadmissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      mutatingWebhookConfiguration,
				Namespace: namespace,
			},
		}
		validatingWebhookConfigurationObject = kadmissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      validatingWebhookConfiguration,
				Namespace: namespace,
			},
		}

		// non-webhook resources
		anotherServiceObject = kcorev1.Service{
			ObjectMeta: kmetav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testService,
			},
		}
		anotherCronjobObject = kbatchv1.CronJob{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      testCronjob,
				Namespace: testNamespace,
			},
		}
		anotherJobObject = kbatchv1.Job{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      testJob,
				Namespace: testNamespace,
			},
		}
		anotherMutatingWebhookConfigurationObject = kadmissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      testMutatingWebhookConfiguration,
				Namespace: testNamespace,
			},
		}
		anotherValidatingWebhookConfigurationObject = kadmissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:      testValidatingWebhookConfiguration,
				Namespace: testNamespace,
			},
		}
	)

	tests := []struct {
		name                 string
		client               client.Client
		givenResources       []client.Object // resources to create before testing cleanup
		wantResourcesExists  []client.Object // resources that should exist after cleanup
		wantResourcesDeleted []client.Object // resources that should be deleted after cleanup
		wantErrors           []error
	}{
		{
			name:   "should delete all webhook resources and keep non-webhook resources",
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
			givenResources: []client.Object{
				// webhook resources
				copyObject(&serviceObject),
				copyObject(&cronjobObject),
				copyObject(&jobObject),
				copyObject(&mutatingWebhookConfigurationObject),
				copyObject(&validatingWebhookConfigurationObject),
				// non-webhook resources
				copyObject(&anotherServiceObject),
				copyObject(&anotherCronjobObject),
				copyObject(&anotherJobObject),
				copyObject(&anotherMutatingWebhookConfigurationObject),
				copyObject(&anotherValidatingWebhookConfigurationObject),
			},
			wantResourcesExists: []client.Object{
				// non-webhook resources
				&anotherServiceObject,
				&anotherCronjobObject,
				&anotherJobObject,
				&anotherMutatingWebhookConfigurationObject,
				&anotherValidatingWebhookConfigurationObject,
			},
			wantResourcesDeleted: []client.Object{
				// webhook resources
				&serviceObject,
				&cronjobObject,
				&jobObject,
				&mutatingWebhookConfigurationObject,
				&validatingWebhookConfigurationObject,
			},
			wantErrors: []error{},
		},
		{
			name:   "should not find webhook resources to delete and keep non-webhook resources",
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
			givenResources: []client.Object{
				// non-webhook resources
				copyObject(&anotherServiceObject),
				copyObject(&anotherCronjobObject),
				copyObject(&anotherJobObject),
				copyObject(&anotherMutatingWebhookConfigurationObject),
				copyObject(&anotherValidatingWebhookConfigurationObject),
			},
			wantResourcesExists: []client.Object{
				// non-webhook resources
				&anotherServiceObject,
				&anotherCronjobObject,
				&anotherJobObject,
				&anotherMutatingWebhookConfigurationObject,
				&anotherValidatingWebhookConfigurationObject,
			},
			wantResourcesDeleted: []client.Object{},
			wantErrors:           []error{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			/////
			// create resources
			/////
			ctx := context.TODO()
			for _, resource := range test.givenResources {
				require.NoError(t, test.client.Create(ctx, resource))
			}

			// ensure resources exist before cleanup
			for _, resource := range test.givenResources {
				require.NoError(t, test.client.Get(ctx, client.ObjectKeyFromObject(resource), resource))
			}

			// ensure test resources are deleted after execution
			defer func() {
				for _, resource := range test.givenResources {
					if err := test.client.Delete(ctx, resource); err != nil {
						t.Logf("failed to delete resource: %v", err)
					}
				}
			}()

			/////
			// cleanup resources once
			/////

			// when
			gotErrors1 := CleanupResources(ctx, test.client)

			// then
			require.Equal(t, test.wantErrors, gotErrors1)

			// ensure resources exist after cleanup
			for _, resource := range test.wantResourcesExists {
				require.NoError(t, test.client.Get(ctx, client.ObjectKeyFromObject(resource), resource))
			}

			// ensure resources are deleted after cleanup
			for _, resource := range test.wantResourcesDeleted {
				err := test.client.Get(ctx, client.ObjectKeyFromObject(resource), resource)
				require.Error(t, err)
				require.True(t, kerrors.IsNotFound(err))
			}

			/////
			// cleanup resources again
			/////

			// when
			gotErrors2 := CleanupResources(ctx, test.client)

			// then
			require.Equal(t, test.wantErrors, gotErrors2)

			// ensure resources exist after cleanup
			for _, resource := range test.wantResourcesExists {
				require.NoError(t, test.client.Get(ctx, client.ObjectKeyFromObject(resource), resource))
			}

			// ensure resources are deleted after cleanup
			for _, resource := range test.wantResourcesDeleted {
				err := test.client.Get(ctx, client.ObjectKeyFromObject(resource), resource)
				require.Error(t, err)
				require.True(t, kerrors.IsNotFound(err))
			}
		})
	}
}

func Test_appendIfError(t *testing.T) {
	// given
	var errList []error

	// when
	errList = appendIfError(errList, nil)
	errList = appendIfError(errList, nil)

	// then
	require.Empty(t, errList)

	// when
	errList = appendIfError(errList, errors.New("error1")) //nolint:goerr113
	errList = appendIfError(errList, errors.New("error2")) //nolint:goerr113

	// then
	require.Len(t, errList, 2)
}

func copyObject(object client.Object) client.Object {
	return object.DeepCopyObject().(client.Object) //nolint:forcetypeassert // no need to check type assertion here
}
