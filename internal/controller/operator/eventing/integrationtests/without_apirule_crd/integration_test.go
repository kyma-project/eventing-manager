package controllerswitching

import (
	"os"
	"testing"

	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils"
	testutilsintegration "github.com/kyma-project/eventing-manager/test/utils/integration"
)

const (
	projectRootDir = "../../../../../../"
)

var testEnvironment *testutilsintegration.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = testutilsintegration.NewTestEnvironment(testutilsintegration.TestEnvironmentConfig{
		ProjectRootDir:            projectRootDir,
		CELValidationEnabled:      true,
		APIRuleCRDEnabled:         false,
		ApplicationRuleCRDEnabled: true,
		NATSCRDEnabled:            true,
		AllowedEventingCR:         nil,
	}, nil)
	if err != nil {
		panic(err)
	}

	// run tests
	code := m.Run()

	// tear down test env
	if err = testEnvironment.TearDown(); err != nil {
		panic(err)
	}

	os.Exit(code)
}

// Test_EventMesh_APIRule_Dependency_Check tests that when the APIRule CRD is missing in case of EventMesh,
// the Eventing CR should have Error status with a message. The success case where APIRule exists is
// covered in integrationtests/controller/integration_test.go.
func Test_EventMesh_APIRule_Dependency_Check(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		givenEventing *operatorv1alpha1.Eventing
		wantMatches   gomegatypes.GomegaMatcher
	}{
		{
			name: "Eventing CR should error state due to APIRule CRD not available",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name1"),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HaveEventMeshSubManagerNotReadyCondition(
					"API-Gateway module is needed for EventMesh backend. APIRules CRD is not installed"),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			// given
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.Namespace
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create eventing-webhook-auth secret.
			testEnvironment.EnsureOAuthSecretCreated(t, tc.givenEventing)

			// create EventMesh secret.
			testEnvironment.EnsureEventMeshSecretCreated(t, tc.givenEventing)

			// when
			// create Eventing CR.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			// then
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantMatches)
		})
	}
}
