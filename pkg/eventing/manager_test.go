package eventing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	ecdeployment "github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	autoscalingv2 "k8s.io/api/autoscaling/v2"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_CreateOrUpdatePublisherProxy(t *testing.T) {
	// Define a list of test cases
	var replicas *int32 = new(int32)
	*replicas = 2

	testCases := []struct {
		name           string
		eventing       *v1alpha1.Eventing
		givenNats      []natsv1alpha1.NATS
		expectedResult *appsv1.Deployment
		expectedError  error
	}{
		{
			name: "CreateOrUpdatePublisherProxy success",
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: ecdeployment.PublisherNamespace,
				},
				Spec: v1alpha1.EventingSpec{
					Backends: []v1alpha1.Backend{
						{
							Type: v1alpha1.NatsBackendType,
						},
					},
					Publisher: v1alpha1.Publisher{
						Replicas: v1alpha1.Replicas{
							Min: 2,
							Max: 4,
						},
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("100m"),
								v1.ResourceMemory: resource.MustParse("256Mi"),
							},
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("200m"),
								v1.ResourceMemory: resource.MustParse("512Mi"),
							},
						},
					},
				},
			},
			givenNats: []natsv1alpha1.NATS{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-eventing",
						Namespace: ecdeployment.PublisherNamespace,
					},
					Status: natsv1alpha1.NATSStatus{
						State: natsv1alpha1.StateReady,
					},
				},
			},
			expectedResult: &appsv1.Deployment{},
			expectedError:  nil,
		},
		{
			name: "CreateOrUpdatePublisherProxy error",
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.EventingSpec{
					Backends: []v1alpha1.Backend{
						{
							Type: "invalid",
						},
					},
					Publisher: v1alpha1.Publisher{
						Replicas: v1alpha1.Replicas{
							Min: 1,
							Max: 5,
						},
					},
				},
			},
			givenNats: []natsv1alpha1.NATS{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-eventing",
						Namespace: ecdeployment.PublisherNamespace,
					},
					Status: natsv1alpha1.NATSStatus{
						State: natsv1alpha1.StateReady,
					},
				},
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

			kubeClient.On("GetNATSResources", ctx, tc.eventing.Namespace).Return(&natsv1alpha1.NATSList{
				Items: tc.givenNats,
			}, nil)
			mockECReconcileClient.On("CreateOrUpdatePublisherProxy",
				mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{}, nil)

			logger, err := logger.New("json", "info")
			em := EventingManager{
				Client:             mockClient,
				kubeClient:         kubeClient,
				ecReconcilerClient: mockECReconcileClient,
				logger:             logger,
			}

			// when
			result, err := em.CreateOrUpdatePublisherProxy(ctx, tc.eventing)

			// then
			require.Equal(t, tc.expectedResult, result)
			require.Equal(t, tc.expectedError, err)
		})
	}
}

func TestCreateOrUpdateHPA(t *testing.T) {
	// Define a list of test cases
	testCases := []struct {
		name              string
		deployment        *appsv1.Deployment
		eventing          *v1alpha1.Eventing
		cpuUtilization    int32
		memoryUtilization int32
		givenGetErr       error
		expectedError     error
	}{
		{
			name: "Create new HPA",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(2),
				},
			},
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.EventingSpec{
					Publisher: v1alpha1.Publisher{
						Replicas: v1alpha1.Replicas{
							Min: 1,
							Max: 5,
						},
					},
				},
			},
			cpuUtilization:    50,
			memoryUtilization: 50,
			givenGetErr:       apierrors.NewNotFound(autoscalingv2.Resource("HorizontalPodAutoscaler"), "eventing-publisher-proxy"),
			expectedError:     nil,
		},
		{
			name: "Update existing HPA",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(2),
				},
			},
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.EventingSpec{
					Publisher: v1alpha1.Publisher{
						Replicas: v1alpha1.Replicas{
							Min: 1,
							Max: 5,
						},
					},
				},
			},
			cpuUtilization:    50,
			memoryUtilization: 50,
			expectedError:     nil,
		},
		{
			name: "Get HPA error",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(2),
				},
			},
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.EventingSpec{
					Publisher: v1alpha1.Publisher{
						Replicas: v1alpha1.Replicas{
							Min: 1,
							Max: 5,
						},
					},
				},
			},
			cpuUtilization:    50,
			memoryUtilization: 50,
			givenGetErr:       errors.New("get HPA error"),
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
			var err error
			if tc.expectedError != nil {
				err = tc.expectedError
			} else {
				hpa = createNewHorizontalPodAutoscaler(
					tc.deployment,
					int32(tc.eventing.Spec.Publisher.Min), int32(tc.eventing.Spec.Publisher.Max),
					tc.cpuUtilization, tc.memoryUtilization,
				)
				err = nil
			}

			mockClient.On("Scheme").Return(func() *runtime.Scheme {
				scheme := runtime.NewScheme()
				_ = v1alpha1.AddToScheme(scheme)
				_ = v1.AddToScheme(scheme)
				_ = autoscalingv2.AddToScheme(scheme)
				return scheme
			}())
			mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(tc.givenGetErr).Run(
				func(args mock.Arguments) {
					hpaArg := args.Get(2).(*autoscalingv2.HorizontalPodAutoscaler)
					if hpa != nil {
						*hpaArg = *hpa
					}
				})
			mockClient.On("Create", mock.Anything, mock.Anything).Return(nil)
			mockClient.On("Update", mock.Anything, mock.Anything).Return(nil)

			// when
			err = em.CreateOrUpdateHPA(context.Background(), tc.deployment, tc.eventing, tc.cpuUtilization, tc.memoryUtilization)

			// then

			require.Equal(t, tc.expectedError, err)
			// create case
			if tc.givenGetErr != nil && apierrors.IsNotFound(tc.givenGetErr) {
				mockClient.AssertCalled(t, "Create", mock.Anything, mock.Anything)
			}
			// update case
			if tc.expectedError == nil && tc.givenGetErr == nil {
				mockClient.AssertCalled(t, "Update", mock.Anything, mock.Anything)
				hpaArg := mockClient.Calls[0].Arguments.Get(2).(*autoscalingv2.HorizontalPodAutoscaler)
				require.Equal(t, int32(tc.eventing.Spec.Publisher.Min), *hpaArg.Spec.MinReplicas)
				require.Equal(t, int32(tc.eventing.Spec.Publisher.Max), hpaArg.Spec.MaxReplicas)
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
				{
					Status: natsv1alpha1.NATSStatus{
						State: natsv1alpha1.StateReady,
					},
				},
			},
			givenNamespace: "test-namespace",
			wantAvailable:  true,
			wantErr:        nil,
		},
		{
			name: "NATS is not available",
			givenNATSResources: []natsv1alpha1.NATS{
				{
					Status: natsv1alpha1.NATSStatus{
						State: natsv1alpha1.StateError,
					},
				},
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-nats",
						Namespace: "test-namespace",
					},
				},
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
			require.Equal(t, tc.want, url)
			require.Equal(t, tc.wantErr, err)
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
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.EventingSpec{
					Backends: []v1alpha1.Backend{
						{
							Type: v1alpha1.NatsBackendType,
							Config: v1alpha1.BackendConfig{
								NATSStorageType:    "file",
								NATSStreamReplicas: 2,
								MaxStreamSize:      resource.MustParse("1Gi"),
								MaxMsgsPerTopic:    1000,
							},
						},
					},
				},
			},
			givenNatsResources: []natsv1alpha1.NATS{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-nats",
						Namespace: "test-namespace",
					},
				},
			},
			expectedConfig: env.NATSConfig{
				URL:                     "nats://test-nats.test-namespace.svc.cluster.local:4222",
				JSStreamStorageType:     "file",
				JSStreamReplicas:        2,
				JSStreamMaxBytes:        "1Gi",
				JSStreamMaxMsgsPerTopic: 1000,
			},
			expectedError: nil,
		},
		{
			name: "Error getting NATS URL",
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.EventingSpec{
					Backends: []v1alpha1.Backend{
						{
							Type: v1alpha1.NatsBackendType,
							Config: v1alpha1.BackendConfig{
								NATSStorageType:    "file",
								NATSStreamReplicas: 2,
								MaxStreamSize:      resource.MustParse("1Gi"),
								MaxMsgsPerTopic:    1000,
							},
						},
					},
				},
			},
			expectedError: fmt.Errorf("failed to get NATS URL"),
		},
	}

	// Iterate over the test cases and run sub-tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
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
			eventing: &v1alpha1.Eventing{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-eventing",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.EventingSpec{
					Publisher: v1alpha1.Publisher{
						Replicas: v1alpha1.Replicas{
							Min: 2,
							Max: 4,
						},
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("100m"),
								v1.ResourceMemory: resource.MustParse("256Mi"),
							},
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("200m"),
								v1.ResourceMemory: resource.MustParse("512Mi"),
							},
						},
					},
				},
			},
			expectedConfig: env.BackendConfig{
				PublisherConfig: env.PublisherConfig{
					RequestsCPU:    "100m",
					RequestsMemory: "256Mi",
					LimitsCPU:      "200m",
					LimitsMemory:   "512Mi",
					Replicas:       2,
				},
			},
		},
	}

	// Iterate over the test cases and run sub-tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake BackendConfig object
			em := &EventingManager{}

			em.updatePublisherConfig(tc.eventing)

			// Check that the BackendConfig object was updated correctly
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
