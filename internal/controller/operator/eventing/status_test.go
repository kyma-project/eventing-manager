package eventing

import (
	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
	"testing"
)

func Test_IsDeploymentReady(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name            string
		givenDeployment *kappsv1.Deployment
		wantResult      bool
	}{
		{
			name: "should return false when all replicas are not updated",
			givenDeployment: &kappsv1.Deployment{
				Spec: kappsv1.DeploymentSpec{Replicas: ptr.To[int32](2)},
				Status: kappsv1.DeploymentStatus{
					Replicas:          2,
					UpdatedReplicas:   1,
					AvailableReplicas: 2,
					ReadyReplicas:     2,
				},
			},
			wantResult: false,
		},
		{
			name: "should return false when all replicas are not updated",
			givenDeployment: &kappsv1.Deployment{
				Spec: kappsv1.DeploymentSpec{Replicas: ptr.To[int32](2)},
				Status: kappsv1.DeploymentStatus{
					Replicas:          2,
					UpdatedReplicas:   1,
					AvailableReplicas: 2,
					ReadyReplicas:     2,
				},
			},
			wantResult: false,
		},
		{
			name: "should return false when all replicas are not available",
			givenDeployment: &kappsv1.Deployment{
				Spec: kappsv1.DeploymentSpec{Replicas: ptr.To[int32](2)},
				Status: kappsv1.DeploymentStatus{
					Replicas:          2,
					UpdatedReplicas:   2,
					AvailableReplicas: 1,
					ReadyReplicas:     2,
				},
			},
			wantResult: false,
		},
		{
			name: "should return false when all replicas are not ready",
			givenDeployment: &kappsv1.Deployment{
				Spec: kappsv1.DeploymentSpec{Replicas: ptr.To[int32](2)},
				Status: kappsv1.DeploymentStatus{
					Replicas:          2,
					UpdatedReplicas:   2,
					AvailableReplicas: 2,
					ReadyReplicas:     1,
				},
			},
			wantResult: false,
		},
		{
			name: "should return true when all replicas are ready",
			givenDeployment: &kappsv1.Deployment{
				Spec: kappsv1.DeploymentSpec{Replicas: ptr.To[int32](2)},
				Status: kappsv1.DeploymentStatus{
					Replicas:          2,
					UpdatedReplicas:   2,
					AvailableReplicas: 2,
					ReadyReplicas:     2,
				},
			},
			wantResult: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// when
			result := IsDeploymentReady(testcase.givenDeployment)

			// then
			require.Equal(t, testcase.wantResult, result)
		})
	}
}
