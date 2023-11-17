package controller_test

import (
	"fmt"
	"os"
	"testing"

	natstestutils "github.com/kyma-project/nats-manager/testutils"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/batch/v1alpha1"
	"github.com/kyma-project/eventing-manager/test/matchers"
	testutils "github.com/kyma-project/eventing-manager/test/utils"

	integrationutils "github.com/kyma-project/eventing-manager/test/utils/integration"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
)

const (
	projectRootDir = "../../../../../../"
)

var testEnvironment *integrationutils.TestEnvironment //nolint:gochecknoglobals // used in tests

// define allowed Eventing CR.
//
//nolint:gochecknoglobals // used in tests
var givenAllowedEventingCR = testutils.NewEventingCR(
	testutils.WithEventingCRName("eventing"),
	testutils.WithEventingCRNamespace("kyma-system"),
)

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integrationutils.NewTestEnvironment(integrationutils.TestEnvironmentConfig{
		ProjectRootDir:            projectRootDir,
		CELValidationEnabled:      false,
		APIRuleCRDEnabled:         true,
		ApplicationRuleCRDEnabled: true,
		NATSCRDEnabled:            true,
		AllowedEventingCR:         givenAllowedEventingCR,
	})
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

// Test_PreventMultipleEventingCRs tests that only single Eventing CR is allowed to be reconciled in a kyma cluster.
func Test_PreventMultipleEventingCRs(t *testing.T) {
	t.Parallel()

	errMsg := fmt.Sprintf("Only a single Eventing CR with name: %s and namespace: %s "+
		"is allowed to be created in a Kyma cluster.", "eventing",
		"kyma-system")

	testCases := []struct {
		name          string
		givenEventing *eventingv1alpha1.Eventing
		wantMatches   gomegatypes.GomegaMatcher
	}{
		{
			name: "should allow Eventing CR if name and namespace is correct",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingCRName(givenAllowedEventingCR.Name),
				testutils.WithEventingCRNamespace(givenAllowedEventingCR.Namespace),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusProcessing(),
				matchers.HavePublisherProxyReadyConditionProcessing(),
			),
		},
		{
			name: "should not allow Eventing CR if name is incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingCRName("not-allowed-name"),
				testutils.WithEventingCRNamespace("kyma-system"),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HavePublisherProxyConditionForbiddenWithMsg(errMsg),
			),
		},
		{
			name: "should not allow Eventing CR if namespace is incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingCRName("eventing"),
				testutils.WithEventingCRNamespace("not-allowed-namespace"),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HavePublisherProxyConditionForbiddenWithMsg(errMsg),
			),
		},
		{
			name: "should not allow Eventing CR if name and namespace, both are incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRMinimal(),
				testutils.WithEventingCRName("not-allowed-name"),
				testutils.WithEventingCRNamespace("not-allowed-namespace"),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HavePublisherProxyConditionForbiddenWithMsg(errMsg),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewGomegaWithT(t)

			// given
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)
			// create NATS CR with ready status.
			givenNATS := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			)
			givenNATS.SetNamespace(givenNamespace)
			testEnvironment.EnsureK8sResourceCreated(t, givenNATS)
			testEnvironment.EnsureNATSResourceStateReady(t, givenNATS)

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			// then
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantMatches)
		})
	}
}
