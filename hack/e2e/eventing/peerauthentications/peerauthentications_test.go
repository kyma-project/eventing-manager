//go:build e2e
// +build e2e

package peerauthentications

import (
	"fmt"
	"os"
	"testing"

	istiosecv1beta1 "istio.io/api/security/v1beta1"
	istio "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	. "github.com/kyma-project/eventing-manager/hack/e2e/common"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
	"github.com/stretchr/testify/require"
)

var testEnvironment *testenvironment.TestEnvironment

// TestMain runs before all the other test functions.
func TestMain(m *testing.M) {
	testEnvironment = testenvironment.NewTestEnvironment()

	// Wait for eventing-manager deployment to get ready.
	if err := testEnvironment.WaitForDeploymentReady(ManagerDeploymentName, NamespaceName, ""); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_EventPublisherProxyMetricsPeerAuthentication checks if the Istio PeerAuthentication for the metrics endpoint of
// Event-Publisher-Proxy was created.
func Test_EventPublisherProxyMetricsPeerAuthentication(t *testing.T) {
	t.Parallel()
	peerAuthName := "eventing-publisher-proxy-metrics"

	deploy, err := testEnvironment.GetDeploymentFromK8s(ManagerDeploymentName, NamespaceName)
	require.NoError(t, err)

	var gotPeerAuth *istio.PeerAuthentication
	retryErr := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		var err error
		gotPeerAuth, err = testEnvironment.GetPeerAuthenticationFromK8s(peerAuthName, NamespaceName)
		if err != nil {
			return err
		}
		if gotPeerAuth == nil {
			return fmt.Errorf("PeerAuthentication %s not found", peerAuthName)
		}
		return nil
	})
	require.NoError(t, retryErr)

	// check the PeerAuthentication OwnerReference
	wantOwnerReferences := []metav1.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               deploy.Name,
			UID:                deploy.UID,
			Controller:         ptr.To(true),
			BlockOwnerDeletion: ptr.To(false),
		},
	}
	require.Equal(t, wantOwnerReferences, gotPeerAuth.OwnerReferences)

	// check spec
	// check MatchLabels
	wantMatchLabels := map[string]string{
		"app.kubernetes.io/name": "eventing-publisher-proxy",
	}
	require.Equal(t, wantMatchLabels, gotPeerAuth.Spec.Selector.MatchLabels)

	// check PortLevelMtls
	require.NotNil(t, gotPeerAuth.Spec.PortLevelMtls[9090])
	require.Equal(t, istiosecv1beta1.PeerAuthentication_MutualTLS_PERMISSIVE, gotPeerAuth.Spec.PortLevelMtls[9090].Mode)
}

// Test_EventingManagerMetricsPeerAuthentication checks if the Istio PeerAuthentication for the metrics endpoint of
// Eventing-Manager was created.
func Test_EventingManagerMetricsPeerAuthentication(t *testing.T) {
	t.Parallel()
	peerAuthName := "eventing-manager-metrics"

	deploy, err := testEnvironment.GetDeploymentFromK8s(ManagerDeploymentName, NamespaceName)
	require.NoError(t, err)

	var gotPeerAuth *istio.PeerAuthentication
	retryErr := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		var err error
		gotPeerAuth, err = testEnvironment.GetPeerAuthenticationFromK8s(peerAuthName, NamespaceName)
		if err != nil {
			return err
		}
		if gotPeerAuth == nil {
			return fmt.Errorf("PeerAuthentication %s not found", peerAuthName)
		}
		return nil
	})
	require.NoError(t, retryErr)

	// check the PeerAuthentication OwnerReference
	wantOwnerReferences := []metav1.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               deploy.Name,
			UID:                deploy.UID,
			Controller:         ptr.To(true),
			BlockOwnerDeletion: ptr.To(false),
		},
	}
	require.Equal(t, wantOwnerReferences, gotPeerAuth.OwnerReferences)

	// check spec
	// check MatchLabels
	wantMatchLabels := map[string]string{
		"app.kubernetes.io/name":     "eventing-manager",
		"app.kubernetes.io/instance": "eventing-manager",
	}
	require.Equal(t, wantMatchLabels, gotPeerAuth.Spec.Selector.MatchLabels)

	// check PortLevelMtls
	require.NotNil(t, gotPeerAuth.Spec.PortLevelMtls[8080])
	require.Equal(t, istiosecv1beta1.PeerAuthentication_MutualTLS_PERMISSIVE, gotPeerAuth.Spec.PortLevelMtls[8080].Mode)
}
