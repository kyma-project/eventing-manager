package eventing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
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
			kubeClient := new(k8smocks.Client)
			kubeClient.On("GetDeployment", ctx, mock.Anything, mock.Anything).Return(tc.givenDeployment, nil)
			kubeClient.On("Create", ctx, mock.Anything).Return(nil)
			kubeClient.On("PatchApply", ctx, mock.Anything).Return(tc.patchApplyErr)

			mockClient := new(mocks.Client)
			mockClient.On("Scheme").Return(newScheme)
			em := &EventingManager{
				Client:        mockClient,
				kubeClient:    kubeClient,
				natsConfig:    env.NATSConfig{},
				backendConfig: env.BackendConfig{},
			}

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
			),
			givenNatsResources: []natsv1alpha1.NATS{
				*natstestutils.NewNATSCR(
					natstestutils.WithNATSCRName("test-nats"),
					natstestutils.WithNATSCRNamespace("test-namespace"),
				),
			},
			expectedConfig: env.NATSConfig{
				URL: "nats://test-nats.test-namespace.svc.cluster.local:4222",
			},
			expectedError: nil,
		},
		{
			name: "Error getting NATS URL",
			eventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("test-eventing"),
				testutils.WithEventingCRNamespace("test-namespace"),
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingStreamData("Memory", "700Mi", 2, 1000),
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
			err := em.setUrlToNatsConfig(ctx, tc.eventing)

			// then
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expectedConfig, em.natsConfig)
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
			mockClient := new(mocks.Client)
			kubeClient := new(k8smocks.Client)

			var createdObjects []client.Object
			// define mocks behaviours.
			mockClient.On("Scheme").Return(newScheme)
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
			pSvc, err := testutils.FindServiceFromK8sObjects(GetEPPPublishServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(pSvc, *tc.givenEventing))

			// check Metrics Service.
			mSvc, err := testutils.FindServiceFromK8sObjects(GetEPPMetricsServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(mSvc, *tc.givenEventing))

			// check Health Service.
			hSvc, err := testutils.FindServiceFromK8sObjects(GetEPPHealthServiceName(*tc.givenEventing), createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(hSvc, *tc.givenEventing))

			// check HPA.
			hpa, err := testutils.FindObjectByKind("HorizontalPodAutoscaler", createdObjects)
			require.NoError(t, err)
			require.True(t, testutils.HasOwnerReference(hpa, *tc.givenEventing))
		})
	}
}
