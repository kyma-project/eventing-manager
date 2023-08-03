package eventing

import (
	"testing"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_EPPResourcesNames(t *testing.T) {
	eventingCR := v1alpha1.Eventing{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
		},
	}
	require.Equal(t, "test1-publisher-proxy", GetEPPDeploymentName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetEPPPublishServiceName(eventingCR))
	require.Equal(t, "test1-publisher-proxy-metrics", GetEPPMetricsServiceName(eventingCR))
	require.Equal(t, "test1-publisher-proxy-health", GetEPPHealthServiceName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetEPPServiceAccountName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetEPPClusterRoleName(eventingCR))
	require.Equal(t, "test1-publisher-proxy", GetEPPClusterRoleBindingName(eventingCR))
}
