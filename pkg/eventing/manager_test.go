package eventing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ecdeployment "github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	autoscalingv2 "k8s.io/api/autoscaling/v2"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_CreateOrUpdatePublisherProxy(t *testing.T) {
	testCases := []struct {
		name           string
		givenEventing  *v1alpha1.Eventing
		givenNats      []natsv1alpha1.NATS
		expectedResult *appsv1.Deployment
		expectedError  error
	}{
		{
			name: "CreateOrUpdatePublisherProxy success",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingPublisherData(2, 4, "100m", "256Mi", "200m", "512Mi"),
			),
			givenNats: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSCRName("test-eventing"),
					natstestutils.WithNATSCRNamespace(ecdeployment.PublisherNamespace),
					natstestutils.WithNATSStateReady(),
				),
			},
			expectedResult: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing-publisher-proxy",
					Namespace: "test-namespace",
				},
			},
			expectedError: nil,
		},
		{
			name: "CreateOrUpdatePublisherProxy error",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				testutils.WithEventingInvalidBackend(),
				testutils.WithEventingPublisherData(2, 4, "100m", "256Mi", "200m", "512Mi"),
			),
			givenNats: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSCRName("test-eventing"),
					natstestutils.WithNATSCRNamespace(ecdeployment.PublisherNamespace),
					natstestutils.WithNATSStateReady(),
				),
			},
			expectedResult: nil,
			expectedError:  fmt.Errorf("NATs backend is not specified in the eventing CR"),
		},
	}

	// Iterate over the test cases and run sub-tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			mockECReconcileClient := new(mocks.ECReconcilerClient)
			mockClient := new(mocks.Client)
			kubeClient := new(k8smocks.Client)

			kubeClient.On("GetNATSResources", ctx, tc.givenEventing.Namespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNats,
			}, nil)
			mockECReconcileClient.On("CreateOrUpdatePublisherProxy",
				mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.expectedResult, nil)

			logger, _ := logger.New("json", "info")
			em := EventingManager{
				Client:             mockClient,
				kubeClient:         kubeClient,
				ecReconcilerClient: mockECReconcileClient,
				logger:             logger,
			}

			// when
			result, err := em.CreateOrUpdatePublisherProxy(ctx, tc.givenEventing)

			// then
			require.Equal(t, tc.expectedResult, result)
			require.Equal(t, tc.expectedError, err)
		})
	}
}

func Test_CreateOrUpdateHPA(t *testing.T) {
	// Define a list of test cases
	testCases := []struct {
		name              string
		givenDeployment   *appsv1.Deployment
		givenEventing     *v1alpha1.Eventing
		cpuUtilization    int32
		memoryUtilization int32
		expectedGetHPAErr error
		expectedError     error
	}{
		{
			name: "Create new HPA",
			givenDeployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(2),
				},
			},
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test-namespace"),
				testutils.WithEventingInvalidBackend(),
				testutils.WithEventingPublisherData(1, 5, "100m", "256Mi", "200m", "512Mi"),
			),
			cpuUtilization:    50,
			memoryUtilization: 50,
			expectedGetHPAErr: apierrors.NewNotFound(autoscalingv2.Resource("HorizontalPodAutoscaler"), "eventing-publisher-proxy"),
			expectedError:     nil,
		},
		{
			name: "Update existing HPA",
			givenDeployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(2),
				},
			},
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test-namespace"),
				testutils.WithEventingInvalidBackend(),
				testutils.WithEventingPublisherData(1, 5, "100m", "256Mi", "200m", "512Mi"),
			),
			cpuUtilization:    50,
			memoryUtilization: 50,
			expectedError:     nil,
		},
		{
			name: "Get HPA error",
			givenDeployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(2),
				},
			},
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test-namespace"),
				testutils.WithEventingInvalidBackend(),
				testutils.WithEventingPublisherData(1, 5, "100m", "256Mi", "200m", "512Mi"),
			),
			cpuUtilization:    50,
			memoryUtilization: 50,
			expectedGetHPAErr: errors.New("get HPA error"),
			expectedError: fmt.Errorf("failed to get horizontal pod autoscaler: %v",
				errors.New("get HPA error")),
		},
	}

	// Iterate over the test cases and run sub-tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock client
			mockClient := new(mocks.Client)
			kubeClient := new(k8smocks.Client)

			// Create a fake EventingManager with the mock client
			em := &EventingManager{
				Client:     mockClient,
				kubeClient: kubeClient,
			}

			// Set up the mock client to return an error or a HorizontalPodAutoscaler object
			var hpa *autoscalingv2.HorizontalPodAutoscaler
			if tc.expectedError == nil {
				hpa = newHorizontalPodAutoscaler(
					tc.givenDeployment,
					int32(tc.givenEventing.Spec.Publisher.Min), int32(tc.givenEventing.Spec.Publisher.Max),
					tc.cpuUtilization, tc.memoryUtilization,
				)
			}

			mockClient.On("Scheme").Return(func() *runtime.Scheme {
				scheme := runtime.NewScheme()
				_ = v1alpha1.AddToScheme(scheme)
				_ = v1.AddToScheme(scheme)
				_ = autoscalingv2.AddToScheme(scheme)
				return scheme
			}())
			mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(tc.expectedGetHPAErr).Run(
				func(args mock.Arguments) {
					hpaArg := args.Get(2).(*autoscalingv2.HorizontalPodAutoscaler)
					if hpa != nil {
						*hpaArg = *hpa
					}
				})
			mockClient.On("Create", mock.Anything, mock.Anything).Return(nil)
			mockClient.On("Update", mock.Anything, mock.Anything).Return(nil)

			// when
			err := em.CreateOrUpdateHPA(context.Background(), tc.givenDeployment, tc.givenEventing, tc.cpuUtilization, tc.memoryUtilization)

			// then

			require.Equal(t, tc.expectedError, err)
			// create case
			if tc.expectedGetHPAErr != nil && apierrors.IsNotFound(tc.expectedGetHPAErr) {
				mockClient.AssertCalled(t, "Create", mock.Anything, mock.Anything)
			}
			// update case
			if tc.expectedError == nil && tc.expectedGetHPAErr == nil {
				mockClient.AssertCalled(t, "Update", mock.Anything, mock.Anything)
				hpaArg := mockClient.Calls[0].Arguments.Get(2).(*autoscalingv2.HorizontalPodAutoscaler)
				require.Equal(t, int32(tc.givenEventing.Spec.Publisher.Min), *hpaArg.Spec.MinReplicas)
				require.Equal(t, int32(tc.givenEventing.Spec.Publisher.Max), hpaArg.Spec.MaxReplicas)
				require.Equal(t, int32(tc.cpuUtilization), *hpaArg.Spec.Metrics[0].Resource.Target.AverageUtilization)
				require.Equal(t, int32(tc.memoryUtilization), *hpaArg.Spec.Metrics[1].Resource.Target.AverageUtilization)
			}
		})
	}
}

func int32Ptr(i int32) *int32 { return &i }

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
			wantErr:             fmt.Errorf("NATS CR is not found to build NATS server URL"),
		},
		{
			name:                "NATS resource does not exist",
			givenNatsResources:  nil,
			givenNamespace:      "test-namespace",
			want:                "",
			getNATSResourcesErr: fmt.Errorf("NATS CR is not found to build NATS server URL"),
			wantErr:             fmt.Errorf("NATS CR is not found to build NATS server URL"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, tc.givenNamespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNatsResources,
			}, tc.getNATSResourcesErr)

			em := EventingManager{
				kubeClient: kubeClient,
			}

			// when
			url, err := em.getNATSUrl(ctx, tc.givenNamespace)

			// then
			require.Equal(t, tc.wantErr, err)
			require.Equal(t, tc.want, url)
		})
	}
}

func Test_UpdateNatsConfig(t *testing.T) {
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
			eventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test-namespace"),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingStreamData("File", "1Gi", "700Mi", 2, 1000),
			),
			givenNatsResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSCRName("test-nats"),
					natstestutils.WithNATSCRNamespace("test-namespace"),
				),
			},
			expectedConfig: env.NATSConfig{
				URL:                     "nats://test-nats.test-namespace.svc.cluster.local:4222",
				JSStreamStorageType:     "File",
				JSStreamReplicas:        2,
				JSStreamMaxBytes:        "700Mi",
				JSStreamMaxMsgsPerTopic: 1000,
			},
			expectedError: nil,
		},
		{
			name: "Error getting NATS URL",
			eventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test-namespace"),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingStreamData("Memory", "1Gi", "700Mi", 2, 1000),
			),
			givenNatsResources: nil,
			expectedError:      fmt.Errorf("failed to get NATS URL"),
		},
	}

	// Iterate over the test cases and run sub-tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			ctx := context.Background()
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetNATSResources", ctx, tc.eventing.Namespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNatsResources,
			}, tc.expectedError)

			em := &EventingManager{
				kubeClient: kubeClient,
			}

			// when
			err := em.updateNatsConfig(ctx, tc.eventing)

			// then
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expectedConfig, em.natsConfig)
		})
	}
}

func Test_UpdatePublisherConfig(t *testing.T) {
	// Define a list of test cases
	testCases := []struct {
		name           string
		eventing       *v1alpha1.Eventing
		expectedConfig env.BackendConfig
	}{
		{
			name: "Update BackendConfig",
			eventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingPublisherData(2, 2, "100m", "99Mi", "399m", "199Mi"),
			),
			expectedConfig: env.BackendConfig{
				PublisherConfig: env.PublisherConfig{
					RequestsCPU:    "100m",
					RequestsMemory: "99Mi",
					LimitsCPU:      "399m",
					LimitsMemory:   "199Mi",
					Replicas:       2,
				},
			},
		},
	}

	// Iterate over the test cases and run sub-tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			em := &EventingManager{}

			// when
			em.updatePublisherConfig(tc.eventing)

			// then
			require.Equal(t, tc.expectedConfig, em.backendConfig)
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

	// EPP deployment
	sampleEPPDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels: map[string]string{
				"test": "test",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(2),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Labels: map[string]string{
						"test2": "test2",
					},
				},
			},
		},
	}

	findObjectByKind := func(kind string, objects []client.Object) (client.Object, error) {
		for _, obj := range objects {
			if obj.GetObjectKind().GroupVersionKind().Kind == kind {
				return obj, nil
			}
		}

		return nil, errors.New("not found")
	}

	findService := func(name string, objects []client.Object) (client.Object, error) {
		for _, obj := range objects {
			if obj.GetObjectKind().GroupVersionKind().Kind == "Service" &&
				obj.GetName() == name {
				return obj, nil
			}
		}

		return nil, errors.New("not found")
	}

	// test cases
	testCases := []struct {
		name                      string
		givenEventing             *v1alpha1.Eventing
		wantCreatedResourcesCount int
	}{
		{
			name: "CreateOrUpdatePublisherProxy success",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingPublisherData(2, 4, "100m", "256Mi", "200m", "512Mi"),
			),
			wantCreatedResourcesCount: 6,
		},
	}

	// Iterate over the test cases.
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			mockClient := new(mocks.Client)
			kubeClient := new(k8smocks.Client)

			var createdObjects []client.Object
			mockClient.On("Scheme").Return(newScheme)
			kubeClient.On("PatchApply", ctx, mock.Anything).Run(func(args mock.Arguments) {
				obj := args.Get(1).(client.Object)
				createdObjects = append(createdObjects, obj)
			}).Return(nil)

			logger, _ := logger.New("json", "info")
			em := EventingManager{
				Client:     mockClient,
				kubeClient: kubeClient,
				logger:     logger,
			}

			// when
			err := em.DeployPublisherProxyResources(ctx, tc.givenEventing, sampleEPPDeployment)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantCreatedResourcesCount, len(createdObjects))

			// check ServiceAccount.
			sa, err := findObjectByKind("ServiceAccount", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(sa, *tc.givenEventing))

			// check ClusterRole.
			cr, err := findObjectByKind("ClusterRole", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(cr, *tc.givenEventing))

			// check ClusterRoleBinding.
			crb, err := findObjectByKind("ClusterRoleBinding", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(crb, *tc.givenEventing))

			// check Publish Service.
			pSvc, err := findService(GetEPPPublishServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(pSvc, *tc.givenEventing))

			// check Metrics Service.
			mSvc, err := findService(GetEPPMetricsServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(mSvc, *tc.givenEventing))

			// check Health Service.
			hSvc, err := findService(GetEPPHealthServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(hSvc, *tc.givenEventing))
		})
	}
}
