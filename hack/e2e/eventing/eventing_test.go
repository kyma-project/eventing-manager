//go:build e2e
// +build e2e

package eventing

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/eventing-manager/hack/e2e/common"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
	"github.com/stretchr/testify/require"
)

var testEnvironment *testenvironment.TestEnvironment

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	testEnvironment = testenvironment.NewTestEnvironment()

	// create test namespace,
	if err := testEnvironment.CreateTestNamespace(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// setup sink for subscriptions.
	if err := testEnvironment.SetupSink(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// create subscriptions.
	if err := testEnvironment.CreateAllSubscriptions(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// wait for subscriptions.
	if err := testEnvironment.WaitForAllSubscriptions(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// initialize event publisher client.
	testEnvironment.InitEventPublisherClient()

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func Test_LegacyEvents_SubscriptionV1Alpha1(t *testing.T) {
	t.Parallel()
	for _, subToTest := range fixtures.V1Alpha1SubscriptionsToTest() {
		subToTest := subToTest
		for _, eventTypeToTest := range subToTest.Types {
			eventTypeToTest := eventTypeToTest
			testName := fmt.Sprintf("legacy event should work for subscription: %s with type: %s", subToTest.Name, eventTypeToTest)
			// run test for the eventType.
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				// when
				err := common.Retry(testenvironment.ThreeAttempts, testenvironment.Interval, func() error {
					return testEnvironment.TestDeliveryOfLegacyEventForSubV1Alpha1(eventTypeToTest)
				})

				// then
				require.NoError(t, err)
			})
		}
	}
}

func Test_LegacyEvents(t *testing.T) {
	t.Parallel()
	for _, subToTest := range fixtures.V1Alpha2SubscriptionsToTest() {
		subToTest := subToTest
		for _, eventTypeToTest := range subToTest.Types {
			eventTypeToTest := eventTypeToTest
			testName := fmt.Sprintf("legacy event should work for subscription: %s with type: %s", subToTest.Name, eventTypeToTest)
			// run test for the eventType.
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				err := common.Retry(testenvironment.ThreeAttempts, testenvironment.Interval, func() error {
					// It's fine if the Namespace already exists.
					return testEnvironment.TestDeliveryOfLegacyEvent(subToTest.Source, eventTypeToTest)
				})
				require.NoError(t, err)
			})
		}
	}
}
