package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			kubeClient := NewKubeClient(fakeClient, testFieldManager)

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
