package controller_test

import (
	"os"
	"testing"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"

	ecdeployment "github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
)

const projectRootDir = "../../../../../"

var testEnvironment *testutils.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = testutils.NewTestEnvironment(projectRootDir, false, nil)
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

func Test_CreateEventingCR(t *testing.T) {
	// t.Parallel()

	testCases := []struct {
		name                 string
		givenEventing        *eventingv1alpha1.Eventing
		givenNATS            *natsv1alpha1.NATS
		givenDeploymentReady bool
		givenNATSReady       bool
		wantMatches          gomegatypes.GomegaMatcher
	}{
		{
			name: "Eventing CR should have processing due to NATS is not available",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				utils.WithEventingCRDefaults(),
				utils.WithEventingStreamData("Memory", 1, "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(ecdeployment.PublisherNamespace),
				natstestutils.WithNATSCRDefaults(),
			),
			givenNATSReady: false,
			wantMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HaveNATSAvailableConditionNotAvailable(),
			),
		},
		{
			name: "Eventing CR should have ready state when all deployment replicas are ready",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				utils.WithEventingCRDefaults(),
				utils.WithEventingStreamData("Memory", 1, "1M", 1, 1),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(ecdeployment.PublisherNamespace),
				natstestutils.WithNATSCRDefaults(),
			),
			givenDeploymentReady: true,
			givenNATSReady:       true,
			wantMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableConditionAvailable(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
			),
		},
		{
			name: "Eventing CR should have processing state deployment is not ready yet",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				utils.WithEventingCRDefaults(),
				utils.WithEventingStreamData("Memory", 1, "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(ecdeployment.PublisherNamespace),
				natstestutils.WithNATSCRDefaults(),
			),
			givenDeploymentReady: false,
			givenNATSReady:       true,
			wantMatches: gomega.And(
				matchers.HaveStatusProcessing(),
				matchers.HaveNATSAvailableConditionAvailable(),
				matchers.HavePublisherProxyReadyConditionProcessing(),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel()
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return tc.givenDeploymentReady
			}
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)
			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)
			if tc.givenNATSReady {
				testEnvironment.EnsureNATSResourceStateReady(t, tc.givenNATS)
			}

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)
				if tc.givenNATSReady && !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, ecdeployment.PublisherName, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, tc.givenNATS)
			}()

			// then
			if tc.givenDeploymentReady {
				testEnvironment.EnsureDeploymentExists(t, ecdeployment.PublisherName, givenNamespace)
				testEnvironment.EnsureHPAExists(t, ecdeployment.PublisherName, givenNamespace)
				testEnvironment.EnsureEventingSpecPublisherReflected(t, tc.givenEventing)
				testEnvironment.EnsureEventingReplicasReflected(t, tc.givenEventing)
				// TODO: ensure NATS Backend config is reflected. Done as subscription controller is implemented.
			}
			// check NATS CR status.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantMatches)
		})
	}
}

func Test_UpdateEventingCR(t *testing.T) {
	testCases := []struct {
		name                string
		givenEventing       *eventingv1alpha1.Eventing
		givenUpdateEventing *eventingv1alpha1.Eventing
	}{
		{
			name: "Updating Eventing CR should be reflected",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				utils.WithEventingCRDefaults(),
				utils.WithEventingStreamData("Memory", 1, "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenUpdateEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				utils.WithEventingCRDefaults(),
				utils.WithEventingStreamData("Memory", 2, "2M", 2, 2),
				utils.WithEventingPublisherData(2, 2, "299m", "199Mi", "499m", "299Mi"),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return true
			}
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)
			nats := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(ecdeployment.PublisherNamespace),
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			)
			testEnvironment.EnsureK8sResourceCreated(t, nats)
			testEnvironment.EnsureNATSResourceStateReady(t, nats)

			testEnvironment.EnsureDeploymentExists(t, ecdeployment.PublisherName, givenNamespace)
			testEnvironment.EnsureHPAExists(t, ecdeployment.PublisherName, givenNamespace)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, ecdeployment.PublisherName, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
			}()

			// get Eventing CR.
			eventing, err := testEnvironment.GetEventingFromK8s(tc.givenEventing.Name, givenNamespace)
			require.NoError(t, err)

			// when
			// update NATS CR.
			newEventing := eventing.DeepCopy()
			newEventing.Spec = tc.givenUpdateEventing.Spec
			testEnvironment.EnsureK8sResourceUpdated(t, newEventing)

			// then
			testEnvironment.EnsureEventingSpecPublisherReflected(t, newEventing)
			testEnvironment.EnsureEventingReplicasReflected(t, newEventing)
		})
	}
}

func Test_DeleteEventingCR(t *testing.T) {

	testCases := []struct {
		name          string
		givenEventing *eventingv1alpha1.Eventing
	}{
		{
			name: "Delete Eventing CR should delete the owned resources",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRNamespace(ecdeployment.PublisherNamespace),
				utils.WithEventingCRDefaults(),
				utils.WithEventingStreamData("Memory", 1, "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return true
			}
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)
			nats := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(ecdeployment.PublisherNamespace),
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			)
			testEnvironment.EnsureK8sResourceCreated(t, nats)
			testEnvironment.EnsureNATSResourceStateReady(t, nats)

			defer func() {
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, ecdeployment.PublisherName, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
			}()

			testEnvironment.EnsureDeploymentExists(t, ecdeployment.PublisherName, givenNamespace)
			testEnvironment.EnsureHPAExists(t, ecdeployment.PublisherName, givenNamespace)
			testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)

			// then
			if *testEnvironment.EnvTestInstance.UseExistingCluster {
				testEnvironment.EnsureDeploymentNotFound(t, ecdeployment.PublisherName, givenNamespace)
				testEnvironment.EnsureHPANotFound(t, ecdeployment.PublisherName, givenNamespace)
			}
		})
	}
}
