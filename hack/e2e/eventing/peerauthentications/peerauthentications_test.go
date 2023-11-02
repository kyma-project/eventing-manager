//go:build e2e
// +build e2e

package peerauthentications

import (
	"fmt"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
	"github.com/kyma-project/eventing-manager/pkg/istio/peerauthentication"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
)

var testEnvironment *testenvironment.TestEnvironment

// TestMain runs before all the other test functions.
func TestMain(m *testing.M) {
	testEnvironment = testenvironment.NewTestEnvironment()

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_EventPublisherProxyMetricsPeerAuthentication checks if the Istio PeerAuthentication for the metrics endpoint of
// Event-Publisher-Proxy was created.
func Test_EventPublisherProxyMetricsPeerAuthentication(t *testing.T) {
	t.Parallel()

	deploy, err := testEnvironment.GetDeploymentFromK8s(ManagerDeploymentName, NamespaceName)
	require.NoError(t, err)

	wantPeerAuth := peerauthentication.EventPublisherProxyMetrics(NamespaceName, deploy.OwnerReferences)
	var givenPeerAuth *v1beta1.PeerAuthentication
	eppErr := Retry(test.Attempts, test.Interval, func() error {
		givenPeerAuth, err = istioClient.SecurityV1beta1().PeerAuthentications(NamespaceName).Get(
			context.TODO(), wantPeerAuth.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if givenPeerAuth == nil {
			return fmt.Errorf("could not fetch PeerAuthentication %s", wantPeerAuth.Name)
		}
		return err
	})
	require.NoError(t, eppErr)

	require.Equal(t, wantPeerAuth.OwnerReferences, givenPeerAuth.OwnerReferences)
}

// Test_EventingManagerMetricsPeerAuthentication checks if the Istio PeerAuthentication for the metrics endpoint of
// Eventing-Manager was created.
func Test_EventingManagerMetricsPeerAuthentication(t *testing.T) {
	t.Parallel()

	deploy, err := testEnvironment.GetDeploymentFromK8s(ManagerDeploymentName, NamespaceName)
	require.NoError(t, err)

	wantPeerAuth := peerauthentication.EventingManagerMetrics(NamespaceName, deploy.OwnerReferences)
	var givenPeerAuth *v1beta1.PeerAuthentication
	emErr := Retry(test.Attempts, test.Interval, func() error {
		givenPeerAuth, err = istioClient.SecurityV1beta1().PeerAuthentications(NamespaceName).Get(
			context.TODO(), wantPeerAuth.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if givenPeerAuth == nil {
			return fmt.Errorf("could not fetch PeerAuthentication %s", wantPeerAuth.Name)
		}
		return err
	})
	require.NoError(t, emErr)

	require.Equal(t, wantPeerAuth.OwnerReferences, givenPeerAuth.OwnerReferences)
}
