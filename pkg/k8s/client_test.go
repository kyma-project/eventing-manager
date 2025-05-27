package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	kapixclientsetfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
)

const testFieldManager = "eventing-manager"

func Test_UpdateDeployment(t *testing.T) {
	t.Parallel()

	// Define test cases
	testCases := []struct {
		name                   string
		namespace              string
		givenNewDeploymentSpec kappsv1.DeploymentSpec
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := t.Context()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			givenDeployment := testutils.NewDeployment("test-deployment", testcase.namespace, map[string]string{})
			// Create the deployment if it should exist
			if testcase.givenDeploymentExists {
				require.NoError(t, fakeClient.Create(ctx, givenDeployment))
			}

			givenUpdatedDeployment := givenDeployment.DeepCopy()
			givenUpdatedDeployment.Spec = testcase.givenNewDeploymentSpec

			// when
			err := kubeClient.UpdateDeployment(ctx, givenUpdatedDeployment)

			// then
			if !testcase.givenDeploymentExists {
				require.Error(t, err)
				require.True(t, kerrors.IsNotFound(err))
			} else {
				gotDeploy, err := kubeClient.GetDeployment(ctx, givenDeployment.Name, givenDeployment.Namespace)
				require.NoError(t, err)
				require.Equal(t, testcase.givenNewDeploymentSpec, gotDeploy.Spec)
			}
		})
	}
}

func Test_DeleteResource(t *testing.T) {
	t.Parallel()
	// Define test cases
	testCases := []struct {
		name                  string
		givenDeployment       *kappsv1.Deployment
		givenDeploymentExists bool
	}{
		{
			name: "should delete the deployment",
			givenDeployment: &kappsv1.Deployment{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
			},
			givenDeploymentExists: true,
		},
		{
			name: "should not return error when the deployment does not exist",
			givenDeployment: &kappsv1.Deployment{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace2",
				},
			},
			givenDeploymentExists: false,
		},
	}

	// Run tests
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := t.Context()
			var givenObjs []client.Object
			if testcase.givenDeploymentExists {
				givenObjs = append(givenObjs, testcase.givenDeployment)
			}
			fakeClient := fake.NewClientBuilder().WithObjects(givenObjs...).Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// when
			err := kubeClient.DeleteDeployment(ctx, testcase.givenDeployment.Name, testcase.givenDeployment.Namespace)

			// then
			require.NoError(t, err)
			// Check that the deployment must not exist.
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      testcase.givenDeployment.Name,
				Namespace: testcase.givenDeployment.Namespace,
			}, &kappsv1.Deployment{})
			require.True(t, kerrors.IsNotFound(err), "DeleteResource did not delete deployment")
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := t.Context()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			deployment := &kappsv1.Deployment{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
			}
			// Create the deployment if it should exist
			if !testcase.noDeployment {
				if err := fakeClient.Create(ctx, deployment); err != nil {
					t.Fatalf("failed to create deployment: %v", err)
				}
			}

			// when
			err := kubeClient.DeleteDeployment(ctx, deployment.Name, deployment.Namespace)

			// then
			require.NoError(t, err)
			// Check that the deployment was deleted
			err = fakeClient.Get(ctx,
				types.NamespacedName{Name: "test-deployment", Namespace: testcase.namespace}, &kappsv1.Deployment{})
			require.True(t, kerrors.IsNotFound(err), "DeleteDeployment did not delete deployment")
		})
	}
}

//nolint:dupl //not the same as ClusterRoleBinding
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := t.Context()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			clusterRole := &krbacv1.ClusterRole{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test-clusterrole",
					Namespace: "test-namespace",
				},
			}
			// Create the deployment if it should exist
			if !testcase.noDeployment {
				if err := fakeClient.Create(ctx, clusterRole); err != nil {
					t.Fatalf("failed to create ClusterRole: %v", err)
				}
			}

			// when
			err := kubeClient.DeleteClusterRole(ctx, clusterRole.Name, clusterRole.Namespace)

			// then
			require.NoError(t, err)
			// Check that the deployment was deleted
			err = fakeClient.Get(ctx,
				types.NamespacedName{Name: clusterRole.Name, Namespace: clusterRole.Namespace}, &krbacv1.ClusterRole{})
			require.True(t, kerrors.IsNotFound(err), "DeleteClusterRole did not delete ClusterRole")
		})
	}
}

//nolint:dupl // not the same as ClusterRole
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := t.Context()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}
			clusterRoleBinding := &krbacv1.ClusterRoleBinding{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test-clusterrolebinding",
					Namespace: "test-namespace",
				},
			}
			// Create the deployment if it should exist
			if !testcase.noDeployment {
				if err := fakeClient.Create(ctx, clusterRoleBinding); err != nil {
					t.Fatalf("failed to create ClusterRoleBinding: %v", err)
				}
			}

			// when
			err := kubeClient.DeleteClusterRoleBinding(ctx, clusterRoleBinding.Name, clusterRoleBinding.Namespace)

			// then
			require.NoError(t, err)
			// Check that the deployment was deleted
			err = fakeClient.Get(ctx,
				types.NamespacedName{Name: clusterRoleBinding.Name, Namespace: clusterRoleBinding.Namespace}, &krbacv1.ClusterRoleBinding{})
			require.True(t, kerrors.IsNotFound(err), "DeleteClusterRoleBinding did not delete ClusterRoleBinding")
		})
	}
}

func Test_GetSecret(t *testing.T) {
	t.Parallel()
	// Define test cases as a table.
	testCases := []struct {
		name                string
		givenNamespacedName string
		wantSecret          *kcorev1.Secret
		wantError           error
		wantNotFoundError   bool
	}{
		{
			name:                "success",
			givenNamespacedName: "test-namespace/test-secret",
			wantSecret: &kcorev1.Secret{
				TypeMeta: kmetav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: kmetav1.ObjectMeta{
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
			wantError:           ErrSecretRefInvalid,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := t.Context()
			fakeClient := fake.NewClientBuilder().Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// Create the secret if it should exist
			if testcase.wantSecret != nil {
				require.NoError(t, fakeClient.Create(ctx, testcase.wantSecret))
			}

			// Call the GetSecret function with the test case's givenNamespacedName.
			secret, err := kubeClient.GetSecret(t.Context(), testcase.givenNamespacedName)

			// Assert that the function returned the expected secret and error.
			if testcase.wantNotFoundError {
				require.True(t, kerrors.IsNotFound(err))
			} else {
				require.ErrorIs(t, err, testcase.wantError)
			}
			require.Equal(t, testcase.wantSecret, secret)
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD := testutils.NewApplicationCRD()
			var objs []runtime.Object
			if !testcase.wantNotFoundError {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotCRD, err := kubeClient.GetCRD(t.Context(), testcase.givenCRDName)

			// then
			if testcase.wantNotFoundError {
				require.Error(t, err)
				require.True(t, kerrors.IsNotFound(err))
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD := testutils.NewApplicationCRD()
			var objs []runtime.Object
			if testcase.wantResult {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotResult, err := kubeClient.ApplicationCRDExists(t.Context())

			// then
			require.NoError(t, err)
			require.Equal(t, testcase.wantResult, gotResult)
		})
	}
}

func Test_PeerAuthenticationCRDExists(t *testing.T) {
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD, err := testutils.NewPeerAuthenticationCRD()
			require.NoError(t, err)
			var objs []runtime.Object
			if testcase.wantResult {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotResult, err := kubeClient.PeerAuthenticationCRDExists(t.Context())

			// then
			require.NoError(t, err)
			require.Equal(t, testcase.wantResult, gotResult)
		})
	}
}

func TestGetSubscriptions(t *testing.T) {
	t.Parallel()
	// Define test cases
	testCases := []struct {
		name                 string
		wantSubscriptionList *eventingv1alpha2.SubscriptionList
	}{
		{
			name: "exists subscription",
			wantSubscriptionList: &eventingv1alpha2.SubscriptionList{
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
		},
		{
			name:                 "no subscription",
			wantSubscriptionList: &eventingv1alpha2.SubscriptionList{},
		},
	}

	// Iterate over test cases
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := t.Context()
			scheme := runtime.NewScheme()
			err := eventingv1alpha2.AddToScheme(scheme)
			require.NoError(t, err)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// Create the secret if it should exist
			if testcase.wantSubscriptionList != nil && len(testcase.wantSubscriptionList.Items) > 0 {
				require.NoError(t, fakeClient.Create(ctx, &testcase.wantSubscriptionList.Items[0]))
			}

			// Call the GetSubscriptions method
			result, _ := kubeClient.GetSubscriptions(t.Context())

			// Assert the result of the method
			if testcase.wantSubscriptionList != nil && len(testcase.wantSubscriptionList.Items) > 0 {
				require.NotEmpty(t, result.Items)
			} else {
				require.Empty(t, result.Items)
			}
		})
	}
}

func Test_GetConfigMap(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		givenName         string
		givenNamespace    string
		wantNotFoundError bool
	}{
		{
			name:              "should return configmap",
			givenName:         "test-name",
			givenNamespace:    "test-namespace",
			wantNotFoundError: false,
		},
		{
			name:              "should not return configmap",
			givenName:         "non-existing",
			givenNamespace:    "non-existing",
			wantNotFoundError: true,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			ctx := t.Context()
			kubeClient := &KubeClient{client: fake.NewClientBuilder().Build()}
			givenCM := testutils.NewConfigMap(testcase.givenName, testcase.givenNamespace)
			if !testcase.wantNotFoundError {
				require.NoError(t, kubeClient.client.Create(ctx, givenCM))
			}

			// when
			gotCM, err := kubeClient.GetConfigMap(t.Context(), testcase.givenName, testcase.givenNamespace)

			// then
			if testcase.wantNotFoundError {
				require.Error(t, err)
				require.True(t, kerrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, givenCM.GetName(), gotCM.Name)
			}
		})
	}
}

func Test_APIRuleCRDExists(t *testing.T) {
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []runtime.Object
			if testcase.wantResult {
				sampleCRD := testutils.NewAPIRuleCRD()
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotResult, err := kubeClient.APIRuleCRDExists(t.Context())

			// then
			require.NoError(t, err)
			require.Equal(t, testcase.wantResult, gotResult)
		})
	}
}
