package k8s

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	istiopkgsecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	kadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	kapixclientsetfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
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

var errPatchNotAllowed = errors.New("apply patches are not supported in the fake client")

func Test_PatchApply(t *testing.T) {
	t.Parallel()

	twoReplicas := int32(2)
	threeReplicas := int32(3)

	// define test cases
	testCases := []struct {
		name                  string
		givenDeployment       *kappsv1.Deployment
		givenUpdateDeployment *kappsv1.Deployment
	}{
		{
			name: "should update resource when exists in k8s",
			givenDeployment: &kappsv1.Deployment{
				TypeMeta: kmetav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: kappsv1.DeploymentSpec{
					Replicas: &twoReplicas,
				},
			},
			givenUpdateDeployment: &kappsv1.Deployment{
				TypeMeta: kmetav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: kappsv1.DeploymentSpec{
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
			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager, nil)

			// when
			err := kubeClient.PatchApply(context.Background(), tc.givenUpdateDeployment)

			// then
			// NOTE: The kubeClient.PatchApply is not supported in the fake client.
			// (https://github.com/kubernetes/kubernetes/issues/115598)
			// So in unit test we only check that the client.Patch with client.Apply
			// is called or not.
			// The real behaviour will be tested in integration tests with envTest pkg.
			require.ErrorContains(t, err, errPatchNotAllowed.Error())
		})
	}
}

func Test_PatchApplyPeerAuthentication(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                          string
		givenPeerAuthentication       *istiopkgsecurityv1beta1.PeerAuthentication
		givenUpdatePeerAuthentication *istiopkgsecurityv1beta1.PeerAuthentication
	}{
		{
			name: "should update resource when exists in k8s",
			givenPeerAuthentication: &istiopkgsecurityv1beta1.PeerAuthentication{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "eventing-publisher-proxy-metrics",
					Namespace: "test",
					Labels: map[string]string{
						"app.kubernetes.io/name":    "eventing-publisher-proxy-old",
						"app.kubernetes.io/version": "0.1.0",
					},
				},
				TypeMeta: kmetav1.TypeMeta{
					Kind:       "PeerAuthentication",
					APIVersion: "security.istio.io/v1beta1",
				},
			},
			givenUpdatePeerAuthentication: &istiopkgsecurityv1beta1.PeerAuthentication{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      "eventing-publisher-proxy-metrics",
					Namespace: "test",
					Labels: map[string]string{
						"app.kubernetes.io/name":    "eventing-publisher-proxy-new",
						"app.kubernetes.io/version": "0.1.0",
					},
				},
				TypeMeta: kmetav1.TypeMeta{
					Kind:       "PeerAuthentication",
					APIVersion: "security.istio.io/v1beta1",
				},
			},
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// get crd
			paCRD, err := testutils.NewPeerAuthenticationCRD()
			require.NoError(t, err)

			// given
			var objs []client.Object
			objs = append(objs, paCRD)
			if tc.givenPeerAuthentication != nil {
				objs = append(objs, tc.givenPeerAuthentication)
			}

			// define scheme
			fakeClientBuilder := fake.NewClientBuilder()
			newScheme := scheme.Scheme
			require.NoError(t, istiopkgsecurityv1beta1.AddToScheme(newScheme))

			fakeClient := fakeClientBuilder.WithScheme(newScheme).WithObjects(objs...).Build()
			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager, nil)

			// when
			err = kubeClient.PatchApplyPeerAuthentication(context.Background(), tc.givenUpdatePeerAuthentication)

			// then
			// NOTE: The kubeClient.PatchApply is not supported in the fake client.
			// (https://github.com/kubernetes/kubernetes/issues/115598)
			// So in unit test we only check that the client.Patch with client.Apply
			// is called or not.
			// The real behaviour will be tested in integration tests with envTest pkg.
			require.ErrorContains(t, err, errPatchNotAllowed.Error())
		})
	}
}

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
				require.True(t, kerrors.IsNotFound(err))
			} else {
				gotDeploy, err := kubeClient.GetDeployment(ctx, givenDeployment.Name, givenDeployment.Namespace)
				require.NoError(t, err)
				require.Equal(t, tc.givenNewDeploymentSpec, gotDeploy.Spec)
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			var givenObjs []client.Object
			if tc.givenDeploymentExists {
				givenObjs = append(givenObjs, tc.givenDeployment)
			}
			fakeClient := fake.NewClientBuilder().WithObjects(givenObjs...).Build()
			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// when
			err := kubeClient.DeleteDeployment(ctx, tc.givenDeployment.Name, tc.givenDeployment.Namespace)

			// then
			require.NoError(t, err)
			// Check that the deployment must not exist.
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      tc.givenDeployment.Name,
				Namespace: tc.givenDeployment.Namespace,
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
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
			if !tc.noDeployment {
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
				types.NamespacedName{Name: "test-deployment", Namespace: tc.namespace}, &kappsv1.Deployment{})
			require.True(t, kerrors.IsNotFound(err), "DeleteDeployment did not delete deployment")
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
			clusterRole := &krbacv1.ClusterRole{
				ObjectMeta: kmetav1.ObjectMeta{
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
			require.NoError(t, err)
			// Check that the deployment was deleted
			err = fakeClient.Get(ctx,
				types.NamespacedName{Name: clusterRole.Name, Namespace: clusterRole.Namespace}, &krbacv1.ClusterRole{})
			require.True(t, kerrors.IsNotFound(err), "DeleteClusterRole did not delete ClusterRole")
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
			clusterRoleBinding := &krbacv1.ClusterRoleBinding{
				ObjectMeta: kmetav1.ObjectMeta{
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
				require.True(t, kerrors.IsNotFound(err))
			} else {
				require.ErrorIs(t, err, tc.wantError)
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
		wantMutatingWebhook *kadmissionregistrationv1.MutatingWebhookConfiguration
		wantNotFoundError   bool
	}{
		{
			name:      "success",
			givenName: "test-wh",
			wantMutatingWebhook: &kadmissionregistrationv1.MutatingWebhookConfiguration{
				ObjectMeta: kmetav1.ObjectMeta{
					Name: "test-wh",
				},
				Webhooks: []kadmissionregistrationv1.MutatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
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
				require.True(t, kerrors.IsNotFound(err))
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
		wantValidatingWebhook *kadmissionregistrationv1.ValidatingWebhookConfiguration
		wantNotFoundError     bool
	}{
		{
			name:      "success",
			givenName: "test-wh",
			wantValidatingWebhook: &kadmissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: kmetav1.ObjectMeta{
					Name: "test-wh",
				},
				Webhooks: []kadmissionregistrationv1.ValidatingWebhook{
					{
						ClientConfig: kadmissionregistrationv1.WebhookClientConfig{
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
				require.True(t, kerrors.IsNotFound(err))
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

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotCRD, err := kubeClient.GetCRD(context.Background(), tc.givenCRDName)

			// then
			if tc.wantNotFoundError {
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD := testutils.NewApplicationCRD()
			var objs []runtime.Object
			if tc.wantResult {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotResult, err := kubeClient.ApplicationCRDExists(context.Background())

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantResult, gotResult)
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD, err := testutils.NewPeerAuthenticationCRD()
			require.NoError(t, err)
			var objs []runtime.Object
			if tc.wantResult {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotResult, err := kubeClient.PeerAuthenticationCRDExists(context.Background())

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantResult, gotResult)
		})
	}
}

func TestGetSubscriptions(t *testing.T) {
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			ctx := context.Background()
			scheme := runtime.NewScheme()
			err := eventingv1alpha2.AddToScheme(scheme)
			require.NoError(t, err)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			kubeClient := &KubeClient{
				client: fakeClient,
			}

			// Create the secret if it should exist
			if tc.wantSubscriptionList != nil && len(tc.wantSubscriptionList.Items) > 0 {
				require.NoError(t, fakeClient.Create(ctx, &tc.wantSubscriptionList.Items[0]))
			}

			// Call the GetSubscriptions method
			result, _ := kubeClient.GetSubscriptions(context.Background())

			// Assert the result of the method
			if tc.wantSubscriptionList != nil && len(tc.wantSubscriptionList.Items) > 0 {
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			ctx := context.Background()
			kubeClient := &KubeClient{client: fake.NewClientBuilder().Build()}
			givenCM := testutils.NewConfigMap(tc.givenName, tc.givenNamespace)
			if !tc.wantNotFoundError {
				require.NoError(t, kubeClient.client.Create(ctx, givenCM))
			}

			// when
			gotCM, err := kubeClient.GetConfigMap(context.Background(), tc.givenName, tc.givenNamespace)

			// then
			if tc.wantNotFoundError {
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []runtime.Object
			if tc.wantResult {
				sampleCRD := testutils.NewAPIRuleCRD()
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapixclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager, nil)

			// when
			gotResult, err := kubeClient.APIRuleCRDExists(context.Background())

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantResult, gotResult)
		})
	}
}
