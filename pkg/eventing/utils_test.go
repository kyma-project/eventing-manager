package eventing

import (
	"testing"

	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

func Test_EPPResourcesNames(t *testing.T) {
	eventingCR := v1alpha1.Eventing{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: "test1",
		},
	}

	require.Equal(t, "test1-publisher-proxy", GetPublisherDeploymentName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetPublisherPublishServiceName(eventingCR))
	require.Equal(t, "test1-publisher-proxy-metrics", GetPublisherMetricsServiceName(eventingCR))
	require.Equal(t, "test1-publisher-proxy-health", GetPublisherHealthServiceName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetPublisherServiceAccountName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetPublisherClusterRoleName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetPublisherClusterRoleBindingName(eventingCR))
}

func Test_getECBackendType(t *testing.T) {
	// given
	type args struct {
		backendType v1alpha1.BackendType
	}
	tests := []struct {
		name string
		args args
		want v1alpha1.BackendType
	}{
		{
			name: "should return the correct backend type for NATS",
			args: args{
				backendType: "NATS",
			},
			want: "NATS",
		},
		{
			name: "should return the correct backend type for EventMesh",
			args: args{
				backendType: "EventMesh",
			},
			want: "EventMesh",
		},
		{
			name: "should return the default backend type for unsupported input",
			args: args{
				backendType: "Unsupported",
			},
			want: "NATS",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := getECBackendType(tt.args.backendType)

			// then
			require.Equal(t, tt.want, got)
		})
	}
}
