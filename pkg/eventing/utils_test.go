package eventing

import (
	"testing"

	"github.com/kyma-project/eventing-manager/api/operator.kyma-project.io/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_EPPResourcesNames(t *testing.T) {
	eventingCR := v1alpha1.Eventing{
		ObjectMeta: metav1.ObjectMeta{
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
