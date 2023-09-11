//go:build e2e
// +build e2e

package cleanup

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/eventing"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/eventing-manager/hack/e2e/env"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/kyma-project/eventing-manager/hack/e2e/common"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
)

// Constants for retries.
const (
	interval      = 2 * time.Second
	attempts      = 60
	smallInterval = 200 * time.Millisecond
	fewAttempts   = 5
)

// clientSet is what is used to access K8s build-in resources like Pods, Namespaces and so on.
var clientSet *kubernetes.Clientset //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// k8sClient is what is used to access the Eventing CR.
var k8sClient client.Client //nolint:gochecknoglobals // This will only be accessible in e2e tests.

var logger *zap.Logger

var testConfigs *env.E2EConfig

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	var err error
	logger, err = SetupLogger()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	testConfigs, err = env.GetE2EConfig()
	if err != nil {
		logger.Error(err.Error())
		panic(err)

	}

	clientSet, k8sClient, err = GetK8sClients()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()

	// Delete the Namespace used for testing.
	//ctx := context.TODO()
	//err = Retry(fewAttempts, interval, func() error {
	//	// It's fine if the Namespace already exists.
	//	return client.IgnoreAlreadyExists(k8sClient.Delete(ctx, Namespace(testConfigs.TestNamespace)))
	//})
	//if err != nil {
	//	logger.Error(err.Error())
	//	panic(err)
	//}

	os.Exit(code)
}

// Test_CleanupV1Alpha1Subscriptions deletes all the subscriptions created for testing.
func Test_CleanupV1Alpha1Subscriptions(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	for _, subToTest := range V1Alpha1SubscriptionsToTest() {
		subToTest := subToTest
		testName := fmt.Sprintf("Delete subscription %s", subToTest.Name)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, deleteSubscription(ctx, subToTest))
		})
	}
}

// Test_CleanupV1Alpha2Subscriptions deletes all the subscriptions created for testing.
func Test_CleanupV1Alpha2Subscriptions(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	for _, subToTest := range V1Alpha2SubscriptionsToTest() {
		subToTest := subToTest
		testName := fmt.Sprintf("Delete subscription %s", subToTest.Name)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, deleteSubscription(ctx, subToTest))
		})
	}
}

// Test_CleanupSubscriptionSink deletes the subscription sink created for testing.
func Test_CleanupSubscriptionSink(t *testing.T) {
	t.Parallel()

	// TODO: implement me
}

func deleteSubscription(ctx context.Context, subToTest eventing.TestSubscriptionInfo) error {
	return Retry(fewAttempts, interval, func() error {
		// delete subscription from cluster.
		err := k8sClient.Delete(ctx, subToTest.ToSubscriptionV1Alpha2("", testConfigs.TestNamespace))
		if err != nil && !k8serrors.IsNotFound(err) {
			logger.Debug(fmt.Sprintf("failed to delete subscription: %s "+
				"in namespace: %s", subToTest.Name, testConfigs.TestNamespace))
			return err
		}
		return nil
	})
}
