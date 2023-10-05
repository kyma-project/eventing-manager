package k8s

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	testutils "github.com/kyma-project/eventing-manager/test/utils"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	apiclientsetfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testFieldManager = "eventing-manager"

func Test_PatchApply(t *testing.T) {
	t.Parallel()

	// NOTE: In real k8s client, the kubeClient.PatchApply creates the resource
	// if it does not exist on the cluster. But in the fake client the behaviour
	// is not properly replicated. As mentioned: "ObjectMeta's `Generation` and
	// `ResourceVersion` don't behave properly, Patch or Update operations that
	// rely on these fields will fail, or give false positives." in docs
	// https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client/fake
	// This scenario will be tested in integration tests with envTest pkg.

	twoReplicas := int32(2)
	threeReplicas := int32(3)

	// define test cases
	testCases := []struct {
		name                  string
		givenDeployment       *appsv1.Deployment
		givenUpdateDeployment *appsv1.Deployment
	}{
		{
			name: "should update resource when exists in k8s",
			givenDeployment: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &twoReplicas,
				},
			},
			givenUpdateDeployment: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &threeReplicas,
				},
			},
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []client.Object
			if tc.givenDeployment != nil {
				objs = append(objs, tc.givenDeployment)
			}
			fakeClientBuilder := fake.NewClientBuilder()
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()
			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			err := kubeClient.PatchApply(context.Background(), tc.givenUpdateDeployment)

			// then
			require.NoError(t, err)
			// check that it should exist on k8s.
			gotSTS, err := kubeClient.GetDeployment(context.Background(),
				tc.givenUpdateDeployment.GetName(), tc.givenUpdateDeployment.GetNamespace())
			require.NoError(t, err)
			require.Equal(t, tc.givenUpdateDeployment.GetName(), gotSTS.Name)
			require.Equal(t, tc.givenUpdateDeployment.GetNamespace(), gotSTS.Namespace)
			require.Equal(t, *tc.givenUpdateDeployment.Spec.Replicas, *gotSTS.Spec.Replicas)
		})
	}
}

func Test_UpdateDeployment(t *testing.T) {
	t.Parallel()

	// Define test cases
	testCases := []struct {
		name                   string
		namespace              string
		givenNewDeploymentSpec appsv1.DeploymentSpec
		givenDeploymentExists  bool
	}{
		{
			name:                  "should update the deployment",
			namespace:             "test-namespace-1",
			givenDeploymentExists: true,
		},
		{
			name:                  "should give error that deployment does not exist",
			namespace:             "test-namespace-2",
			givenDeploymentExists: false,
		},
	}

	// Run tests
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			givenDeployment := testutils.NewDeployment("test-deployment", tc.namespace, map[string]string{})
			// Create the deployment if it should exist
			if tc.givenDeploymentExists {
				require.NoError(t, fakeClient.Create(ctx, givenDeployment))
			}

			givenUpdatedDeployment := givenDeployment.DeepCopy()
			givenUpdatedDeployment.Spec = tc.givenNewDeploymentSpec

			// when
			err := kubeClient.UpdateDeployment(ctx, givenUpdatedDeployment)

			// then
			if !tc.givenDeploymentExists {
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))
			} else {
				gotDeploy, err := kubeClient.GetDeployment(ctx, givenDeployment.Name, givenDeployment.Namespace)
				require.NoError(t, err)
				require.Equal(t, tc.givenNewDeploymentSpec, gotDeploy.Spec)
			}
		})
	}
}

func Test_DeleteDeployment(t *testing.T) {
	t.Parallel()
	// Define test cases
	testCases := []struct {
		name         string
		namespace    string
		noDeployment bool
	}{
		{
			name:      "deployment exists",
			namespace: "test-namespace",
		},
		{
			name:         "deployment does not exist",
			namespace:    "test-namespace",
			noDeployment: true,
		},
	}

	// Run tests
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
			}
			// Create the deployment if it should exist
			if !tc.noDeployment {
				if err := fakeClient.Create(ctx, deployment); err != nil {
					t.Fatalf("failed to create deployment: %v", err)
				}
			}

			// when
			err := kubeClient.DeleteDeployment(ctx, deployment.Name, deployment.Namespace)

			// then
			require.Nil(t, err)
			// Check that the deployment was deleted
			err = fakeClient.Get(ctx,
				types.NamespacedName{Name: "test-deployment", Namespace: tc.namespace}, &appsv1.Deployment{})
			require.True(t, apierrors.IsNotFound(err), "DeleteDeployment did not delete deployment")
		})
	}
}

func Test_DeleteClusterRole(t *testing.T) {
	t.Parallel()
	// Define test cases
	testCases := []struct {
		name         string
		noDeployment bool
	}{
		{
			name: "ClusterRole exists",
		},
		{
			name:         "ClusterRole does not exist",
			noDeployment: true,
		},
	}

	// Run tests
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			clusterRole := &rbac.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-clusterrole",
					Namespace: "test-namespace",
				},
			}
			// Create the deployment if it should exist
			if !tc.noDeployment {
				if err := fakeClient.Create(ctx, clusterRole); err != nil {
					t.Fatalf("failed to create ClusterRole: %v", err)
				}
			}

			// when
			err := kubeClient.DeleteClusterRole(ctx, clusterRole.Name, clusterRole.Namespace)

			// then
			require.Nil(t, err)
			// Check that the deployment was deleted
			err = fakeClient.Get(ctx,
				types.NamespacedName{Name: clusterRole.Name, Namespace: clusterRole.Namespace}, &rbac.ClusterRole{})
			require.True(t, apierrors.IsNotFound(err), "DeleteClusterRole did not delete ClusterRole")
		})
	}
}

func Test_DeleteClusterRoleBinding(t *testing.T) {
	t.Parallel()
	// Define test cases
	testCases := []struct {
		name         string
		noDeployment bool
	}{
		{
			name: "ClusterRoleBinding exists",
		},
		{
			name:         "ClusterRoleBinding does not exist",
			noDeployment: true,
		},
	}

	// Run tests
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			clusterRoleBinding := &rbac.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-clusterrolebinding",
					Namespace: "test-namespace",
				},
			}
			// Create the deployment if it should exist
			if !tc.noDeployment {
				if err := fakeClient.Create(ctx, clusterRoleBinding); err != nil {
					t.Fatalf("failed to create ClusterRoleBinding: %v", err)
				}
			}

			// when
			err := kubeClient.DeleteClusterRoleBinding(ctx, clusterRoleBinding.Name, clusterRoleBinding.Namespace)

			// then
			require.Nil(t, err)
			// Check that the deployment was deleted
			err = fakeClient.Get(ctx,
				types.NamespacedName{Name: clusterRoleBinding.Name, Namespace: clusterRoleBinding.Namespace}, &rbac.ClusterRoleBinding{})
			require.True(t, apierrors.IsNotFound(err), "DeleteClusterRoleBinding did not delete ClusterRoleBinding")
		})
	}
}

func Test_GetSecret(t *testing.T) {
	t.Parallel()
	// Define test cases as a table.
	testCases := []struct {
		name                string
		givenNamespacedName string
		wantSecret          *corev1.Secret
		wantError           error
		wantNotFoundError   bool
	}{
		{
			name:                "success",
			givenNamespacedName: "test-namespace/test-secret",
			wantSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
		},
		{
			name:                "not found",
			givenNamespacedName: "test-namespace/test-secret",
			wantSecret:          nil,
			wantNotFoundError:   true,
		},
		{
			name:                "namespaced name format error",
			givenNamespacedName: "my-secret",
			wantSecret:          nil,
			wantError:           errors.New("invalid namespaced name. It must be in the format of 'namespace/name'"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// Create the secret if it should exist
			if tc.wantSecret != nil {
				require.NoError(t, fakeClient.Create(ctx, tc.wantSecret))
			}

			// Call the GetSecret function with the test case's givenNamespacedName.
			secret, err := kubeClient.GetSecret(context.Background(), tc.givenNamespacedName)

			// Assert that the function returned the expected secret and error.
			if tc.wantNotFoundError {
				require.True(t, apierrors.IsNotFound(err))
			} else {
				require.Equal(t, tc.wantError, err)
			}
			require.Equal(t, tc.wantSecret, secret)
		})
	}
}

func Test_GetMutatingWebHookConfiguration(t *testing.T) {
	t.Parallel()

	// given
	newCABundle := make([]byte, 20)
	_, readErr := rand.Read(newCABundle)
	require.NoError(t, readErr)

	// Define test cases as a table.
	testCases := []struct {
		name                string
		givenName           string
		wantMutatingWebhook *admissionv1.MutatingWebhookConfiguration
		wantNotFoundError   bool
	}{
		{
			name:      "success",
			givenName: "test-wh",
			wantMutatingWebhook: &admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-wh",
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{
							CABundle: newCABundle,
						},
					},
				},
			},
		},
		{
			name:                "not found",
			givenName:           "test-wh",
			wantMutatingWebhook: nil,
			wantNotFoundError:   true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// Create the MutatingWebHookConfiguration if it should exist
			if tc.wantMutatingWebhook != nil {
				require.NoError(t, fakeClient.Create(ctx, tc.wantMutatingWebhook))
			}

			// when
			gotWebhook, err := kubeClient.GetMutatingWebHookConfiguration(context.Background(), tc.givenName)

			// then
			if !tc.wantNotFoundError {
				require.NoError(t, err)
				require.Equal(t, tc.wantMutatingWebhook.Webhooks, gotWebhook.Webhooks)
			} else {
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))
			}
		})
	}
}

func Test_GetValidatingWebHookConfiguration(t *testing.T) {
	t.Parallel()

	// given
	newCABundle := make([]byte, 20)
	_, readErr := rand.Read(newCABundle)
	require.NoError(t, readErr)

	// Define test cases as a table.
	testCases := []struct {
		name                  string
		givenName             string
		wantValidatingWebhook *admissionv1.ValidatingWebhookConfiguration
		wantNotFoundError     bool
	}{
		{
			name:      "success",
			givenName: "test-wh",
			wantValidatingWebhook: &admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-wh",
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						ClientConfig: admissionv1.WebhookClientConfig{
							CABundle: newCABundle,
						},
					},
				},
			},
		},
		{
			name:                  "not found",
			givenName:             "test-wh",
			wantValidatingWebhook: nil,
			wantNotFoundError:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// Create the ValidatingWebhookConfiguration if it should exist
			if tc.wantValidatingWebhook != nil {
				require.NoError(t, fakeClient.Create(ctx, tc.wantValidatingWebhook))
			}

			// when
			gotWebhook, err := kubeClient.GetValidatingWebHookConfiguration(context.Background(), tc.givenName)

			// then
			if !tc.wantNotFoundError {
				require.NoError(t, err)
				require.Equal(t, tc.wantValidatingWebhook.Webhooks, gotWebhook.Webhooks)
			} else {
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))
			}
		})
	}
}

func Test_GetCRD(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name              string
		givenCRDName      string
		wantNotFoundError bool
	}{
		{
			name:              "should return correct CRD from k8s",
			givenCRDName:      ApplicationCrdName,
			wantNotFoundError: false,
		},
		{
			name:              "should return not found error when CRD is missing in k8s",
			givenCRDName:      "non-existing",
			wantNotFoundError: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD := testutils.NewApplicationCRD()
			var objs []runtime.Object
			if !tc.wantNotFoundError {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := apiclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager)

			// when
			gotCRD, err := kubeClient.GetCRD(context.Background(), tc.givenCRDName)

			// then
			if tc.wantNotFoundError {
				require.Error(t, err)
				require.True(t, apierrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, sampleCRD.GetName(), gotCRD.Name)
			}
		})
	}
}

func Test_ApplicationCRDExists(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name       string
		wantResult bool
	}{
		{
			name:       "should return false when CRD is missing in k8s",
			wantResult: false,
		},
		{
			name:       "should return true when CRD exists in k8s",
			wantResult: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD := testutils.NewApplicationCRD()
			var objs []runtime.Object
			if tc.wantResult {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := apiclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager)

			// when
			gotResult, err := kubeClient.ApplicationCRDExists(context.Background())

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantResult, gotResult)
		})
	}
}
