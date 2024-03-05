package eventing

import (
	"context"
	"errors"
	"testing"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	"github.com/kyma-project/eventing-manager/test"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
)

var ErrUseMeWithMocks = errors.New("use me with mocks")

func Test_ApplyPublisherProxyDeployment(t *testing.T) {
	// given
	newScheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(newScheme))

	// Define test cases
	testCases := []struct {
		name             string
		givenEventing    *v1alpha1.Eventing
		givenBackendType v1alpha1.BackendType
		givenDeployment  *kappsv1.Deployment
		patchApplyErr    error
		wantedDeployment *kappsv1.Deployment
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
			wantErr:          ErrUnknownBackendType,
		},
		{
			name: "PatchApply failure",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingEventTypePrefix("test-prefix"),
			),
			givenBackendType: v1alpha1.NatsBackendType,
			patchApplyErr:    ErrUseMeWithMocks,
			wantErr:          ErrUseMeWithMocks,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			if testcase.givenDeployment != nil {
				testcase.givenDeployment.Namespace = testcase.givenEventing.Namespace
			}
			// define mocks.
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetDeployment", ctx, mock.Anything, mock.Anything).Return(testcase.givenDeployment, nil)
			kubeClient.On("UpdateDeployment", ctx, mock.Anything).Return(nil)
			kubeClient.On("Create", ctx, mock.Anything).Return(nil)
			kubeClient.On("PatchApply", ctx, mock.Anything).Return(testcase.patchApplyErr)

			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()

			logger, err := test.NewEventingLogger()
			require.NoError(t, err)

			mgr := &EventingManager{
				Client:        mockClient,
				kubeClient:    kubeClient,
				backendConfig: env.BackendConfig{},
				logger:        logger,
			}

			// when
			deployment, err := mgr.applyPublisherProxyDeployment(ctx, testcase.givenEventing, &env.NATSConfig{}, testcase.givenBackendType)

			// then
			require.ErrorIs(t, err, testcase.wantErr)
			if testcase.wantedDeployment != nil {
				require.NotNil(t, deployment)
				require.Equal(t, testcase.wantedDeployment.Spec.Template.ObjectMeta.Annotations,
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
		givenCurrentDeploymentFunc func() *kappsv1.Deployment
		givenDesiredDeploymentFunc func() *kappsv1.Deployment
		givenKubeClientFunc        func() *k8smocks.Client
	}{
		{
			name: "should update deployment when the owner reference is not correct",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
			),
			givenCurrentDeploymentFunc: func() *kappsv1.Deployment {
				oldPublisher := testutils.NewDeployment(
					"test-eventing-nats-publisher",
					"test-namespace",
					map[string]string{})
				oldPublisher.OwnerReferences = []kmetav1.OwnerReference{{
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
			givenCurrentDeploymentFunc: func() *kappsv1.Deployment {
				oldPublisher := testutils.NewDeployment(
					"test-eventing-nats-publisher",
					"test-namespace",
					map[string]string{})
				oldPublisher.OwnerReferences = []kmetav1.OwnerReference{{
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			testcase.givenEventing.Name = "eventing-manager"
			existingPublisher := testcase.givenCurrentDeploymentFunc()
			existingPublisher.Namespace = testcase.givenEventing.Namespace
			desiredPublisher := testutils.NewDeployment(existingPublisher.Name,
				testcase.givenEventing.Namespace, map[string]string{})

			// define mocks.
			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()
			kubeClientMock := testcase.givenKubeClientFunc()

			// define logger.
			logger, err := test.NewEventingLogger()
			require.NoError(t, err)

			// create eventing manager instance.
			mgr := &EventingManager{
				Client:     mockClient,
				kubeClient: kubeClientMock,
				logger:     logger,
			}

			// when
			err = mgr.migratePublisherDeploymentFromEC(ctx, testcase.givenEventing, *existingPublisher, *desiredPublisher)

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
			name: "NATS is available if in Warning state",
			givenNATSResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSStateWarning(),
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
			wantErr:            ErrUseMeWithMocks,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, testcase.givenNamespace).Return(&natsv1alpha1.NATSList{
				Items: testcase.givenNATSResources,
			}, testcase.wantErr)

			// when
			em := EventingManager{
				kubeClient: kubeClient,
			}

			// then
			available, err := em.IsNATSAvailable(ctx, testcase.givenNamespace)
			require.Equal(t, testcase.wantAvailable, available)
			require.Equal(t, testcase.wantErr, err)
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
		givenEPPDeployment        *kappsv1.Deployment
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()
			kubeClient := new(k8smocks.Client)

			var createdObjects []client.Object
			// define mocks behaviours.
			if testcase.wantError {
				kubeClient.On("PatchApply", ctx, mock.Anything).Return(ErrUseMeWithMocks)
			} else {
				kubeClient.On("PatchApply", ctx, mock.Anything).Run(func(args mock.Arguments) {
					obj, ok := args.Get(1).(client.Object)
					require.True(t, ok)
					createdObjects = append(createdObjects, obj)
				}).Return(nil)
			}

			mgr := EventingManager{
				Client:     mockClient,
				kubeClient: kubeClient,
			}

			// when
			err := mgr.DeployPublisherProxyResources(ctx, testcase.givenEventing, testcase.givenEPPDeployment)

			// then
			if testcase.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, createdObjects, testcase.wantCreatedResourcesCount)

			// check ServiceAccount.
			sa, err := testutils.FindObjectByKind("ServiceAccount", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(sa, *testcase.givenEventing))

			// check ClusterRole.
			cr, err := testutils.FindObjectByKind("ClusterRole", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(cr, *testcase.givenEventing))

			// check ClusterRoleBinding.
			crb, err := testutils.FindObjectByKind("ClusterRoleBinding", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(crb, *testcase.givenEventing))

			// check Publish Service.
			pSvc, err := testutils.FindServiceFromK8sObjects(GetPublisherPublishServiceName(*testcase.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(pSvc, *testcase.givenEventing))

			// check Metrics Service.
			mSvc, err := testutils.FindServiceFromK8sObjects(GetPublisherMetricsServiceName(*testcase.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(mSvc, *testcase.givenEventing))

			// check Health Service.
			hSvc, err := testutils.FindServiceFromK8sObjects(GetPublisherHealthServiceName(*testcase.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(hSvc, *testcase.givenEventing))

			// check HPA.
			hpa, err := testutils.FindObjectByKind("HorizontalPodAutoscaler", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(hpa, *testcase.givenEventing))
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
		givenEPPDeployment        *kappsv1.Deployment
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			mockClient := fake.NewClientBuilder().WithScheme(newScheme).WithObjects().Build()
			kubeClient := new(k8smocks.Client)

			// define mocks behaviours.
			if testcase.wantError {
				kubeClient.On("DeleteResource", ctx, mock.Anything).Return(ErrUseMeWithMocks)
			} else {
				kubeClient.On("DeleteResource", ctx, mock.Anything).Return(nil).Times(testcase.wantDeletedResourcesCount)
			}

			// initialize EventingManager object.
			mgr := EventingManager{
				Client:     mockClient,
				kubeClient: kubeClient,
			}

			// when
			err := mgr.DeletePublisherProxyResources(ctx, testcase.givenEventing)

			// then
			if testcase.wantError {
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
				TypeMeta: kmetav1.TypeMeta{
					Kind:       "SubscriptionList",
					APIVersion: "eventing.kyma-project.io/v1alpha2",
				},
				Items: []eventingv1alpha2.Subscription{
					{
						TypeMeta: kmetav1.TypeMeta{
							Kind:       "Subscription",
							APIVersion: "eventing.kyma-project.io/v1alpha2",
						},
						ObjectMeta: kmetav1.ObjectMeta{
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
			wantError:  ErrUseMeWithMocks,
		},
	}

	// Iterate over test cases
	for _, tc := range testCases {
		testcase := tc
		// Create a new instance of the mock client
		kubeClient := new(k8smocks.Client)

		// Set up the behavior of the mock client
		kubeClient.On("GetSubscriptions", mock.Anything).Return(testcase.givenSubscriptions, testcase.wantError)

		// Create a new instance of the EventingManager with the mock client
		em := &EventingManager{
			kubeClient: kubeClient,
		}

		// Call the SubscriptionExists method
		result, err := em.SubscriptionExists(context.Background())

		// Assert the result of the method
		require.Equal(t, testcase.wantResult, result, testcase.name)
		require.Equal(t, testcase.wantError, err, testcase.name)
	}
}
