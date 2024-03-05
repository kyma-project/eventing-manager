//go:build e2e
// +build e2e

package delivery

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/eventing"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/hack/e2e/common"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
)

var testEnvironment *testenvironment.TestEnvironment

type EventTestCase string

const (
	LegacyEventCase          EventTestCase = "legacy event"
	StructuredCloudEventCase EventTestCase = "structured cloud event"
	BinaryCloudEventCase     EventTestCase = "binary cloud event"
)

// TestMain runs before all the other test functions.
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
	err := testEnvironment.InitSinkClient()
	if err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// ++ Tests

func Test_LegacyEvents_SubscriptionV1Alpha1(t *testing.T) {
	t.Parallel()
	// binding.EncodingUnknown means legacy event.
	testEventDelivery(t, LegacyEventCase, fixtures.V1Alpha1SubscriptionsToTest(), binding.EncodingUnknown, fixtures.V1Alpha1SubscriptionCRVersion)
}

func Test_LegacyEvents(t *testing.T) {
	t.Parallel()
	// binding.EncodingUnknown means legacy event.
	testEventDelivery(t, LegacyEventCase, fixtures.V1Alpha2SubscriptionsToTest(), binding.EncodingUnknown, fixtures.V1Alpha2SubscriptionCRVersion)
}

func Test_StructuredCloudEvents_SubscriptionV1Alpha1(t *testing.T) {
	t.Parallel()
	testEventDelivery(t, StructuredCloudEventCase, fixtures.V1Alpha1SubscriptionsToTest(), binding.EncodingStructured, fixtures.V1Alpha1SubscriptionCRVersion)
}

func Test_BinaryCloudEvents_SubscriptionV1Alpha1(t *testing.T) {
	t.Parallel()
	testEventDelivery(t, BinaryCloudEventCase, fixtures.V1Alpha1SubscriptionsToTest(), binding.EncodingBinary, fixtures.V1Alpha1SubscriptionCRVersion)
}

func Test_StructuredCloudEvents(t *testing.T) {
	t.Parallel()
	testEventDelivery(t, StructuredCloudEventCase, fixtures.V1Alpha2SubscriptionsToTest(), binding.EncodingStructured, fixtures.V1Alpha2SubscriptionCRVersion)
}

func Test_BinaryCloudEvents(t *testing.T) {
	t.Parallel()
	testEventDelivery(t, BinaryCloudEventCase, fixtures.V1Alpha2SubscriptionsToTest(), binding.EncodingBinary, fixtures.V1Alpha2SubscriptionCRVersion)
}

// ++ Helper functions

func testEventDelivery(t *testing.T,
	testCase EventTestCase,
	subsToTest []eventing.TestSubscriptionInfo,
	encoding binding.Encoding,
	subCRVersion fixtures.SubscriptionCRVersion) {
	// In each subscription, we need to run the tests for each event type.
	// loop over each subscription.
	for _, subToTest := range subsToTest {
		subToTest := subToTest
		// loop over each event type in the subscription.
		for id, eventTypeToTest := range subToTest.Types {
			eventTypeToTest := eventTypeToTest
			// define the test name.
			testName := getTestName(testCase, subToTest, id)
			// run test for the eventType.
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				// given
				eventSourceToUse := subToTest.Source
				if subCRVersion == fixtures.V1Alpha1SubscriptionCRVersion {
					// For EventMesh with Subscription v1alpha1, the eventSource should be EventMesh NameSpace.
					eventSourceToUse = testEnvironment.TestConfigs.EventMeshNamespace
				}

				// when
				err := common.Retry(testenvironment.ThreeAttempts, testenvironment.Interval, func() error {
					if encoding == binding.EncodingUnknown {
						// binding.EncodingUnknown means legacy event.
						return testEnvironment.TestDeliveryOfLegacyEvent(eventSourceToUse, eventTypeToTest, subCRVersion)
					}
					return testEnvironment.TestDeliveryOfCloudEvent(eventSourceToUse, eventTypeToTest, encoding)
				})

				// then
				require.NoError(t, err)
			})
		}
	}
}

func getTestName(testCase EventTestCase, subToTest eventing.TestSubscriptionInfo, typeIndex int) string {
	return fmt.Sprintf("%s should work for subscription(%s) (typeIndex[%v]) %s", testCase, subToTest.Name, typeIndex, subToTest.Description)
}
