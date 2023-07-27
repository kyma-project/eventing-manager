package eventing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
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

func Test_ApplyPublisherProxyDeployment(t *testing.T) {
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
			givenEventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
			},
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
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetDeployment", ctx, mock.Anything, mock.Anything).Return(tc.givenDeployment, nil)
			kubeClient.On("Create", ctx, mock.Anything).Return(nil)
			kubeClient.On("PatchApply", ctx, mock.Anything).Return(tc.patchApplyErr)
			setOwnerReference = func(eventing *v1alpha1.Eventing, desiredPublisher *appsv1.Deployment,
				scheme *runtime.Scheme) error {
				return nil
			}

			mockClient := new(mocks.Client)
			mockClient.On("Scheme").Return(&runtime.Scheme{})
			em := &EventingManager{
				Client:        mockClient,
				kubeClient:    kubeClient,
				natsConfig:    env.NATSConfig{},
				backendConfig: env.BackendConfig{},
			}
			em.updatePublisherConfig(tc.givenEventing)

			// when
			deployment, err := em.applyPublisherProxyDeployment(ctx, tc.givenEventing, tc.givenBackendType)

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
