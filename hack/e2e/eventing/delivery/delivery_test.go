package delivery

import (
	"fmt"
	"os"
	"testing"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/kyma-project/eventing-manager/hack/e2e/common"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
)

var testEnvironment *testenvironment.TestEnvironment

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	testEnvironment = testenvironment.NewTestEnvironment()

	// wait for subscriptions.
	if err := testEnvironment.WaitForAllSubscriptions(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// initialize event publisher client.
	if err := testEnvironment.InitEventPublisherClient(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// initialize sink client for fetching events.
	testEnvironment.InitSinkClient()

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
					return testEnvironment.TestDeliveryOfLegacyEvent("", eventTypeToTest, fixtures.V1Alpha1SubscriptionCRVersion)
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
					return testEnvironment.TestDeliveryOfLegacyEvent(subToTest.Source, eventTypeToTest, fixtures.V1Alpha2SubscriptionCRVersion)
				})
				require.NoError(t, err)
			})
		}
	}
}

func Test_StructuredCloudEvents_SubscriptionV1Alpha1(t *testing.T) {
	t.Parallel()
	for _, subToTest := range fixtures.V1Alpha1SubscriptionsToTest() {
		subToTest := subToTest
		for _, eventTypeToTest := range subToTest.Types {
			eventTypeToTest := eventTypeToTest
			testName := fmt.Sprintf("structured cloud event should work for subscription: %s with type: %s", subToTest.Name, eventTypeToTest)
			// run test for the eventType.
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				// when
				err := common.Retry(testenvironment.ThreeAttempts, testenvironment.Interval, func() error {
					// For EventMesh with Subscription v1alpha1, the eventSource should be EventMesh NameSpace.
					return testEnvironment.TestDeliveryOfCloudEvent(testEnvironment.TestConfigs.EventMeshNamespace, eventTypeToTest, binding.EncodingStructured)
				})

				// then
				require.NoError(t, err)
			})
		}
	}
}

func Test_BinaryCloudEvents_SubscriptionV1Alpha1(t *testing.T) {
	t.Parallel()
	for _, subToTest := range fixtures.V1Alpha1SubscriptionsToTest() {
		subToTest := subToTest
		for _, eventTypeToTest := range subToTest.Types {
			eventTypeToTest := eventTypeToTest
			testName := fmt.Sprintf("structured cloud event should work for subscription: %s with type: %s", subToTest.Name, eventTypeToTest)
			// run test for the eventType.
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				// when
				err := common.Retry(testenvironment.ThreeAttempts, testenvironment.Interval, func() error {
					// For EventMesh with Subscription v1alpha1, the eventSource should be EventMesh NameSpace.
					return testEnvironment.TestDeliveryOfCloudEvent(testEnvironment.TestConfigs.EventMeshNamespace, eventTypeToTest, binding.EncodingBinary)
				})

				// then
				require.NoError(t, err)
			})
		}
	}
}

func Test_StructuredCloudEvents(t *testing.T) {
	t.Parallel()
	for _, subToTest := range fixtures.V1Alpha2SubscriptionsToTest() {
		subToTest := subToTest
		for _, eventTypeToTest := range subToTest.Types {
			eventTypeToTest := eventTypeToTest
			testName := fmt.Sprintf("structured cloud event should work for subscription: %s with type: %s", subToTest.Name, eventTypeToTest)
			// run test for the eventType.
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				// when
				err := common.Retry(testenvironment.ThreeAttempts, testenvironment.Interval, func() error {
					return testEnvironment.TestDeliveryOfCloudEvent(subToTest.Source, eventTypeToTest, binding.EncodingStructured)
				})

				// then
				require.NoError(t, err)
			})
		}
	}
}

func Test_BinaryCloudEvents(t *testing.T) {
	t.Parallel()
	for _, subToTest := range fixtures.V1Alpha2SubscriptionsToTest() {
		subToTest := subToTest
		for _, eventTypeToTest := range subToTest.Types {
			eventTypeToTest := eventTypeToTest
			testName := fmt.Sprintf("structured cloud event should work for subscription: %s with type: %s", subToTest.Name, eventTypeToTest)
			// run test for the eventType.
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				// when
				err := common.Retry(testenvironment.ThreeAttempts, testenvironment.Interval, func() error {
					return testEnvironment.TestDeliveryOfCloudEvent(subToTest.Source, eventTypeToTest, binding.EncodingBinary)
				})

				// then
				require.NoError(t, err)
			})
		}
	}
}
