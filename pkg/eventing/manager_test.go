package eventing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/kyma-project/eventing-manager/test"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"

	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
)

func Test_ApplyPublisherProxyDeployment(t *testing.T) {
	// given
	newScheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(newScheme))

	// Define test cases
	testCases := []struct {
		name             string
		givenEventing    *v1alpha1.Eventing
		givenBackendType v1alpha1.BackendType
		givenDeployment  *appsv1.Deployment
		patchApplyErr    error
		wantedDeployment *appsv1.Deployment
		wantErr          error
	}{
		{
			name: "NATS backend, no current publisher",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
			),
			givenBackendType: v1alpha1.NatsBackendType,
			wantedDeployment: testutils.NewDeployment(
				"test-eventing-nats-publisher",
				"test-namespace", nil),
		},
		{
			name: "NATS backend, preserve only allowed annotations",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
			),
			givenBackendType: v1alpha1.NatsBackendType,
			givenDeployment: testutils.NewDeployment(
				"test-eventing-nats-publisher",
				"test-namespace",
				map[string]string{
					"kubectl.kubernetes.io/restartedAt": "value1",
					"annotation2":                       "value2",
				}),
			wantedDeployment: testutils.NewDeployment(
				"test-eventing-nats-publisher",
				"test-namespace",
				map[string]string{
					"kubectl.kubernetes.io/restartedAt": "value1",
				}),
		},
		{
			name: "Unknown backend",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
			),
			givenBackendType: "unknown-backend",
			wantErr:          fmt.Errorf("unknown EventingBackend type %q", "unknown-backend"),
		},
		{
			name: "PatchApply failure",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingEventTypePrefix("test-prefix"),
			),
			givenBackendType: v1alpha1.NatsBackendType,
			patchApplyErr:    errors.New("patch apply error"),
			wantErr: fmt.Errorf("failed to apply Publisher Proxy deployment: %v",
				errors.New("patch apply error")),
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			if tc.givenDeployment != nil {
				tc.givenDeployment.Namespace = tc.givenEventing.Namespace
			}
			// define mocks.
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetDeployment", ctx, mock.Anything, mock.Anything).Return(tc.givenDeployment, nil)
			kubeClient.On("UpdateDeployment", ctx, mock.Anything).Return(nil)
			kubeClient.On("Create", ctx, mock.Anything).Return(nil)
			kubeClient.On("PatchApply", ctx, mock.Anything).Return(tc.patchApplyErr)

			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()

			logger, err := test.NewEventingLogger()
			require.NoError(t, err)

			em := &EventingManager{
				Client:        mockClient,
				kubeClient:    kubeClient,
				backendConfig: env.BackendConfig{},
				logger:        logger,
			}

			// when
			deployment, err := em.applyPublisherProxyDeployment(ctx, tc.givenEventing, &env.NATSConfig{}, tc.givenBackendType)

			// then
			require.Equal(t, tc.wantErr, err)
			if tc.wantedDeployment != nil {
				require.NotNil(t, deployment)
				require.Equal(t, tc.wantedDeployment.Spec.Template.ObjectMeta.Annotations,
					deployment.Spec.Template.ObjectMeta.Annotations)
			}
		})
	}
}

func Test_migratePublisherDeploymentFromEC(t *testing.T) {
	// given
	newScheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(newScheme))

	// Define test cases
	testCases := []struct {
		name                       string
		givenEventing              *v1alpha1.Eventing
		givenCurrentDeploymentFunc func() *appsv1.Deployment
		givenDesiredDeploymentFunc func() *appsv1.Deployment
		givenKubeClientFunc        func() *k8smocks.Client
	}{
		{
			name: "should update deployment when the owner reference is not correct",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
			),
			givenCurrentDeploymentFunc: func() *appsv1.Deployment {
				oldPublisher := testutils.NewDeployment(
					"test-eventing-nats-publisher",
					"test-namespace",
					map[string]string{})
				oldPublisher.OwnerReferences = []metav1.OwnerReference{{
					APIVersion:         "apps/v1",
					Kind:               "Deployment",
					Name:               "eventing-controller",
					UID:                "a3cdcc7b-6853-4772-99cc-fc1511399d63",
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(true),
				}}
				return oldPublisher
			},
			givenKubeClientFunc: func() *k8smocks.Client {
				kubeClient := new(k8smocks.Client)
				// UpdateDeployment method must have been called once.
				kubeClient.On("UpdateDeployment", mock.Anything,
					mock.Anything).Return(nil).Once()
				return kubeClient
			},
		},
		{
			name: "should not update deployment when the owner reference is correct",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
			),
			givenCurrentDeploymentFunc: func() *appsv1.Deployment {
				oldPublisher := testutils.NewDeployment(
					"test-eventing-nats-publisher",
					"test-namespace",
					map[string]string{})
				oldPublisher.OwnerReferences = []metav1.OwnerReference{{
					APIVersion:         "apps/v1",
					Kind:               "Deployment",
					Name:               "eventing-manager",
					UID:                "a3cdcc7b-6853-4772-99cc-fc1511399d63",
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(true),
				}}
				return oldPublisher
			},
			givenKubeClientFunc: func() *k8smocks.Client {
				kubeClient := new(k8smocks.Client)
				// mock is empty, because we do not want the UpdateDeployment method to have been called.
				return kubeClient
			},
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			tc.givenEventing.Name = "eventing-manager"
			existingPublisher := tc.givenCurrentDeploymentFunc()
			existingPublisher.Namespace = tc.givenEventing.Namespace
			desiredPublisher := testutils.NewDeployment(existingPublisher.Name,
				tc.givenEventing.Namespace, map[string]string{})

			// define mocks.
			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()
			kubeClientMock := tc.givenKubeClientFunc()

			// define logger.
			logger, err := test.NewEventingLogger()
			require.NoError(t, err)

			// create eventing manager instance.
			em := &EventingManager{
				Client:     mockClient,
				kubeClient: kubeClientMock,
				logger:     logger,
			}

			// when
			err = em.migratePublisherDeploymentFromEC(ctx, tc.givenEventing, *existingPublisher, *desiredPublisher)

			// then
			require.NoError(t, err)
			kubeClientMock.AssertExpectations(t)
		})
	}
}

func Test_IsNATSAvailable(t *testing.T) {
	testCases := []struct {
		name               string
		givenNATSResources []natsv1alpha1.NATS
		givenNamespace     string
		wantAvailable      bool
		wantErr            error
	}{
		{
			name: "NATS is available",
			givenNATSResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSStateReady(),
				),
			},
			givenNamespace: "test-namespace",
			wantAvailable:  true,
			wantErr:        nil,
		},
		{
			name: "NATS is not available",
			givenNATSResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSStateProcessing(),
				),
			},
			givenNamespace: "test-namespace",
			wantAvailable:  false,
			wantErr:        nil,
		},
		{
			name:               "NATS is not available if there are no NATS resources",
			givenNATSResources: []natsv1alpha1.NATS{},
			givenNamespace:     "test-namespace",
			wantAvailable:      false,
			wantErr:            nil,
		},
		{
			name:               "Error getting NATS resources",
			givenNATSResources: nil,
			givenNamespace:     "test-namespace",
			wantAvailable:      false,
			wantErr:            errors.New("failed to get NATS resources"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, tc.givenNamespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNATSResources,
			}, tc.wantErr)

			// when
			em := EventingManager{
				kubeClient: kubeClient,
			}

			// then
			available, err := em.IsNATSAvailable(ctx, tc.givenNamespace)
			require.Equal(t, tc.wantAvailable, available)
			require.Equal(t, tc.wantErr, err)
		})
	}

}

func Test_ConvertECBackendType(t *testing.T) {
	// Define a list of test cases
	testCases := []struct {
		name           string
		backendType    v1alpha1.BackendType
		expectedResult ecv1alpha1.BackendType
		expectedError  error
	}{
		{
			name:           "Convert EventMeshBackendType",
			backendType:    v1alpha1.EventMeshBackendType,
			expectedResult: ecv1alpha1.BEBBackendType,
			expectedError:  nil,
		},
		{
			name:           "Convert NatsBackendType",
			backendType:    v1alpha1.NatsBackendType,
			expectedResult: ecv1alpha1.NatsBackendType,
			expectedError:  nil,
		},
		{
			name:           "Unknown backend type",
			backendType:    "unknown",
			expectedResult: "",
			expectedError:  fmt.Errorf("unknown backend type: unknown"),
		},
	}

	// Iterate over the test cases and run sub-tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			result, err := convertECBackendType(tc.backendType)
			// then
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func Test_DeployPublisherProxyResources(t *testing.T) {
	t.Parallel()

	// given
	newScheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(newScheme))

	// test cases
	testCases := []struct {
		name                      string
		givenEventing             *v1alpha1.Eventing
		givenEPPDeployment        *appsv1.Deployment
		wantError                 bool
		wantCreatedResourcesCount int
	}{
		{
			name: "should create all required EPP resources",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test"),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingPublisherData(2, 4, "100m", "256Mi", "200m", "512Mi"),
			),
			givenEPPDeployment:        testutils.NewDeployment("test", "test", map[string]string{}),
			wantCreatedResourcesCount: 7,
		},
		{
			name: "should return error when patch apply fails",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test"),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingPublisherData(2, 4, "100m", "256Mi", "200m", "512Mi"),
			),
			givenEPPDeployment: testutils.NewDeployment("test", "test", map[string]string{}),
			wantError:          true,
		},
	}

	// Iterate over the test cases.
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()
			kubeClient := new(k8smocks.Client)

			var createdObjects []client.Object
			// define mocks behaviours.
			if tc.wantError {
				kubeClient.On("PatchApply", ctx, mock.Anything).Return(errors.New("failed"))
			} else {
				kubeClient.On("PatchApply", ctx, mock.Anything).Run(func(args mock.Arguments) {
					obj := args.Get(1).(client.Object)
					createdObjects = append(createdObjects, obj)
				}).Return(nil)
			}

			em := EventingManager{
				Client:     mockClient,
				kubeClient: kubeClient,
			}

			// when
			err := em.DeployPublisherProxyResources(ctx, tc.givenEventing, tc.givenEPPDeployment)

			// then
			if tc.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantCreatedResourcesCount, len(createdObjects))

			// check ServiceAccount.
			sa, err := testutils.FindObjectByKind("ServiceAccount", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(sa, *tc.givenEventing))

			// check ClusterRole.
			cr, err := testutils.FindObjectByKind("ClusterRole", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(cr, *tc.givenEventing))

			// check ClusterRoleBinding.
			crb, err := testutils.FindObjectByKind("ClusterRoleBinding", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(crb, *tc.givenEventing))

			// check Publish Service.
			pSvc, err := testutils.FindServiceFromK8sObjects(GetPublisherPublishServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(pSvc, *tc.givenEventing))

			// check Metrics Service.
			mSvc, err := testutils.FindServiceFromK8sObjects(GetPublisherMetricsServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(mSvc, *tc.givenEventing))

			// check Health Service.
			hSvc, err := testutils.FindServiceFromK8sObjects(GetPublisherHealthServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(hSvc, *tc.givenEventing))

			// check HPA.
			hpa, err := testutils.FindObjectByKind("HorizontalPodAutoscaler", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(hpa, *tc.givenEventing))
		})
	}
}

func Test_DeletePublisherProxyResources(t *testing.T) {
	t.Parallel()

	// given
	newScheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(newScheme))

	// test cases
	testCases := []struct {
		name                      string
		givenEventing             *v1alpha1.Eventing
		givenEPPDeployment        *appsv1.Deployment
		wantError                 bool
		wantDeletedResourcesCount int
	}{
		{
			name: "should have delete EPP resources",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test"),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingPublisherData(2, 4, "100m", "256Mi", "200m", "512Mi"),
			),
			givenEPPDeployment:        testutils.NewDeployment("test", "test", map[string]string{}),
			wantDeletedResourcesCount: 6,
		},
		{
			name: "should return error when delete fails",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test"),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingPublisherData(2, 4, "100m", "256Mi", "200m", "512Mi"),
			),
			givenEPPDeployment: testutils.NewDeployment("test", "test", map[string]string{}),
			wantError:          true,
		},
	}

	// Iterate over the test cases.
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()
			kubeClient := new(k8smocks.Client)

			// define mocks behaviours.
			if tc.wantError {
				kubeClient.On("DeleteResource", ctx, mock.Anything).Return(errors.New("failed"))
			} else {
				kubeClient.On("DeleteResource", ctx, mock.Anything).Return(nil).Times(tc.wantDeletedResourcesCount)
			}

			// initialize EventingManager object.
			em := EventingManager{
				Client:     mockClient,
				kubeClient: kubeClient,
			}

			// when
			err := em.DeletePublisherProxyResources(ctx, tc.givenEventing)

			// then
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				kubeClient.AssertExpectations(t)
			}
		})
	}
}

func Test_SubscriptionExists(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name               string
		givenSubscriptions *eventingv1alpha2.SubscriptionList
		wantResult         bool
		wantError          error
	}{
		{
			name:               "no subscription should exist",
			givenSubscriptions: &eventingv1alpha2.SubscriptionList{},
			wantResult:         false,
			wantError:          nil,
		},
		{
			name: "subscriptions should exist",
			givenSubscriptions: &eventingv1alpha2.SubscriptionList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "SubscriptionList",
					APIVersion: "eventing.kyma-project.io/v1alpha2",
				},
				Items: []eventingv1alpha2.Subscription{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Subscription",
							APIVersion: "eventing.kyma-project.io/v1alpha2",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-subscription",
							Namespace: "test-namespace",
						},
					},
				},
			},
			wantResult: true,
			wantError:  nil,
		},
		{
			name:       "error should have occurred",
			wantResult: false,
			wantError:  errors.New("client error"),
		},
	}

	// Iterate over test cases
	for _, tc := range testCases {
		// Create a new instance of the mock client
		kubeClient := new(k8smocks.Client)

		// Set up the behavior of the mock client
		kubeClient.On("GetSubscriptions", mock.Anything).Return(tc.givenSubscriptions, tc.wantError)

		// Create a new instance of the EventingManager with the mock client
		em := &EventingManager{
			kubeClient: kubeClient,
		}

		// Call the SubscriptionExists method
		result, err := em.SubscriptionExists(context.Background())

		// Assert the result of the method
		require.Equal(t, tc.wantResult, result, tc.name)
		require.Equal(t, tc.wantError, err, tc.name)
	}
}
