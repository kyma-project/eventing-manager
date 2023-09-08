//go:build e2e
// +build e2e

package eventing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/eventing"
	ecv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	k8stypes "k8s.io/apimachinery/pkg/types"

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

var testConfigs env.E2EConfig

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
	logger.Info(fmt.Sprintf("##### NOTE: Tests will run w.r.t. backend: %s", testConfigs.BackendType))

	clientSet, k8sClient, err = GetK8sClients()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	ctx := context.TODO()
	// Create the Namespace used for testing.
	err = Retry(attempts, interval, func() error {
		// It's fine if the Namespace already exists.
		return client.IgnoreAlreadyExists(k8sClient.Create(ctx, Namespace(testConfigs.TestNamespace)))
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// verify if the sink is ready!
	err = waitForSubscriptionSink()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// setup subscriptions.
	err = setupSubscriptions()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func setupSubscriptions() error {
	ctx := context.TODO()
	// create v1alpha1 subscriptions if not exists.
	err := createV1Alpha1Subscriptions(ctx, V1Alpha1SubscriptionsToTest())
	if err != nil {
		return err
	}

	// create v1alpha2 subscriptions if not exists.
	err = createV1Alpha2Subscriptions(ctx, V1Alpha2SubscriptionsToTest())
	if err != nil {
		return err
	}

	// wait for v1alpha1 subscriptions to get ready.
	err = waitForSubscriptions(ctx, V1Alpha1SubscriptionsToTest())
	if err != nil {
		return err
	}

	// wait for v1alpha2 subscriptions to get ready
	err = waitForSubscriptions(ctx, V1Alpha1SubscriptionsToTest())
	if err != nil {
		return err
	}
	return nil
}

func createV1Alpha1Subscriptions(ctx context.Context, subList []eventing.TestSubscriptionInfo) error {
	for _, subInfo := range subList {
		err := Retry(fewAttempts, smallInterval, func() error {
			newSub := subInfo.ToSubscriptionV1Alpha1(testConfigs.SubscriptionSinkURL, testConfigs.TestNamespace)
			return client.IgnoreAlreadyExists(k8sClient.Create(ctx, newSub))
		})
		// return error if all retries are exhausted.
		if err != nil {
			return err
		}
	}
	return nil
}

func createV1Alpha2Subscriptions(ctx context.Context, subList []eventing.TestSubscriptionInfo) error {
	for _, subInfo := range subList {
		err := Retry(fewAttempts, smallInterval, func() error {
			newSub := subInfo.ToSubscriptionV1Alpha2(testConfigs.SubscriptionSinkURL, testConfigs.TestNamespace)
			return client.IgnoreAlreadyExists(k8sClient.Create(ctx, newSub))
		})
		// return error if all retries are exhausted.
		if err != nil {
			return err
		}
	}
	return nil
}

func waitForSubscriptions(ctx context.Context, subsToTest []eventing.TestSubscriptionInfo) error {
	for _, subToTest := range subsToTest {
		return waitForSubscription(ctx, subToTest)
	}
	return nil
}

func waitForSubscription(ctx context.Context, subsToTest eventing.TestSubscriptionInfo) error {
	return Retry(attempts, interval, func() error {
		// get subscription from cluster.
		gotSub := ecv1alpha2.Subscription{}
		err := k8sClient.Get(ctx, k8stypes.NamespacedName{
			Name:      subsToTest.Name,
			Namespace: subsToTest.Namespace,
		}, &gotSub)
		if err != nil {
			logger.Debug(fmt.Sprintf("failed to check readiness; failed to fetch subscription: %s "+
				"in namespace: %s", subsToTest.Name, subsToTest.Namespace))
			return err
		}

		// check if subscription is reconciled by correct backend.
		if !IsSubscriptionReconcileByBackend(gotSub, testConfigs.BackendType) {
			errMsg := fmt.Sprintf("waiting subscription: %s "+
				"in namespace: %s to get recocniled by backend: %s", subsToTest.Name, subsToTest.Namespace,
				testConfigs.BackendType)
			logger.Debug(errMsg)
			return errors.New(errMsg)
		}

		// check if subscription is ready.
		if !gotSub.Status.Ready {
			errMsg := fmt.Sprintf("waiting subscription: %s "+
				"in namespace: %s to get ready", subsToTest.Name, subsToTest.Namespace)
			logger.Debug(errMsg)
			return errors.New(errMsg)
		}
		return nil
	})
}

func waitForSubscriptionSink() error {
	// TODO: implement me
	return nil
}

func IsSubscriptionReconcileByBackend(sub ecv1alpha2.Subscription, activeBackend string) bool {
	condition := sub.Status.FindCondition(ecv1alpha2.ConditionSubscriptionActive)
	if condition == nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(condition.Reason)), strings.ToLower(activeBackend))
}
