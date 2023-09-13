package setup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
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

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func Test_SubscriptionsReady(t *testing.T) {
	t.Parallel()
	require.NoError(t, testEnvironment.WaitForAllSubscriptions())
}
