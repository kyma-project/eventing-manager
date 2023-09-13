//go:build e2e
// +build e2e

package cleanup

import (
	"os"
	"testing"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
	"github.com/stretchr/testify/require"
)

var testEnvironment *testenvironment.TestEnvironment

// TestMain runs before all the other test functions.
func TestMain(m *testing.M) {
	testEnvironment = testenvironment.NewTestEnvironment()

	// Run the tests and exit.
	code := m.Run()

	// delete test namespace,
	if err := testEnvironment.DeleteTestNamespace(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	os.Exit(code)
}

// Test_CleanupAllSubscriptions deletes all the subscriptions created for testing.
func Test_CleanupAllSubscriptions(t *testing.T) {
	t.Parallel()
	require.NoError(t, testEnvironment.DeleteAllSubscriptions())
}

// Test_CleanupSubscriptionSink deletes the subscription sink created for testing.
func Test_CleanupSubscriptionSink(t *testing.T) {
	t.Parallel()
	require.NoError(t, testEnvironment.DeleteSinkResources())
}
