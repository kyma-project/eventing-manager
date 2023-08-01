package controller_test

import (
	"os"
	"testing"

	"github.com/kyma-project/eventing-manager/pkg/eventing"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils"
	testutils "github.com/kyma-project/eventing-manager/test/utils/integration"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"

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
	testEnvironment, err = testutils.NewTestEnvironment(projectRootDir, false)
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
	t.Parallel()

	testCases := []struct {
		name                 string
		givenEventing        *eventingv1alpha1.Eventing
		givenNATS            *natsv1alpha1.NATS
		givenDeploymentReady bool
		givenNATSReady       bool
		wantMatches          gomegatypes.GomegaMatcher
		wantEnsureK8sObjects bool
	}{
		{
			name: "Eventing CR should error state due to NATS is unavailable",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNATS: natstestutils.NewNATSCR(
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
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			),
			givenDeploymentReady: true,
			givenNATSReady:       true,
			wantMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableConditionAvailable(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
			),
			wantEnsureK8sObjects: true,
		},
		{
			name: "Eventing CR should have processing state deployment is not ready yet",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNATS: natstestutils.NewNATSCR(
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
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return tc.givenDeploymentReady
			}
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.Namespace
			tc.givenNATS.SetNamespace(givenNamespace)

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)
			// when
			// create NATS.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)
			if tc.givenNATSReady {
				testEnvironment.EnsureNATSResourceStateReady(t, tc.givenNATS)
			}
			// create Eventing CR.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)
				if tc.givenNATSReady && !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, tc.givenEventing.Name, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, tc.givenNATS)
			}()

			// then
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantMatches)
			if tc.givenDeploymentReady {
				testEnvironment.EnsureDeploymentExists(t, tc.givenEventing.Name, givenNamespace)
				testEnvironment.EnsureHPAExists(t, tc.givenEventing.Name, givenNamespace)
				testEnvironment.EnsureEventingSpecPublisherReflected(t, tc.givenEventing)
				testEnvironment.EnsureEventingReplicasReflected(t, tc.givenEventing)
				testEnvironment.EnsureDeploymentOwnerReferenceSet(t, tc.givenEventing)
				// TODO: ensure NATS Backend config is reflected. Done as subscription controller is implemented.
			}

			if tc.wantEnsureK8sObjects {
				// check if EPP resources exists.
				testEnvironment.EnsureEPPK8sResourcesExists(t, *tc.givenEventing)

				// check if the owner reference is set.
				testEnvironment.EnsureEPPK8sResourcesHaveOwnerReference(t, *tc.givenEventing)

				// check if EPP resources are correctly created.
				deployment, err := testEnvironment.GetDeploymentFromK8s(tc.givenEventing.Name, givenNamespace)
				require.NoError(t, err)
				// K8s Services
				testEnvironment.EnsureEPPPublishServiceCorrect(t, deployment, *tc.givenEventing)
				testEnvironment.EnsureEPPMetricsServiceCorrect(t, deployment, *tc.givenEventing)
				testEnvironment.EnsureEPPHealthServiceCorrect(t, deployment, *tc.givenEventing)
				// ClusterRole
				testEnvironment.EnsureEPPClusterRoleCorrect(t, *tc.givenEventing)
				// ClusterRoleBinding
				testEnvironment.EnsureEPPClusterRoleBindingCorrect(t, *tc.givenEventing)
			}
		})
	}
}

func Test_UpdateEventingCR(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                      string
		givenExistingEventing     *eventingv1alpha1.Eventing
		givenNewEventingForUpdate *eventingv1alpha1.Eventing
	}{
		{
			name: "Updating Eventing CR should be reflected",
			givenExistingEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNewEventingForUpdate: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "2M", 2, 2),
				utils.WithEventingPublisherData(2, 2, "299m", "199Mi", "499m", "299Mi"),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return true
			}
			// create unique namespace for this test run.
			givenNamespace := tc.givenExistingEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			nats := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(tc.givenExistingEventing.Namespace),
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			)
			testEnvironment.EnsureK8sResourceCreated(t, nats)
			testEnvironment.EnsureNATSResourceStateReady(t, nats)
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenExistingEventing)

			testEnvironment.EnsureDeploymentExists(t, tc.givenExistingEventing.Name, givenNamespace)
			testEnvironment.EnsureHPAExists(t, tc.givenExistingEventing.Name, givenNamespace)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenExistingEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, tc.givenExistingEventing.Name, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
			}()

			// get Eventing CR.
			eventing, err := testEnvironment.GetEventingFromK8s(tc.givenExistingEventing.Name, givenNamespace)
			require.NoError(t, err)

			// when
			// update NATS CR.
			newEventing := eventing.DeepCopy()
			newEventing.Spec = tc.givenNewEventingForUpdate.Spec
			testEnvironment.EnsureK8sResourceUpdated(t, newEventing)

			// then
			testEnvironment.EnsureEventingSpecPublisherReflected(t, newEventing)
			testEnvironment.EnsureEventingReplicasReflected(t, newEventing)
			testEnvironment.EnsureDeploymentOwnerReferenceSet(t, tc.givenExistingEventing)
		})
	}
}

func Test_DeleteEventingCR(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		givenEventing *eventingv1alpha1.Eventing
	}{
		{
			name: "Delete Eventing CR should delete the owned resources",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return true
			}
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			nats := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(tc.givenEventing.Namespace),
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			)
			testEnvironment.EnsureK8sResourceCreated(t, nats)
			testEnvironment.EnsureNATSResourceStateReady(t, nats)
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			defer func() {
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, tc.givenEventing.Name, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
			}()

			testEnvironment.EnsureDeploymentExists(t, tc.givenEventing.Name, givenNamespace)
			testEnvironment.EnsureHPAExists(t, tc.givenEventing.Name, givenNamespace)
			testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)

			// then
			if *testEnvironment.EnvTestInstance.UseExistingCluster {
				testEnvironment.EnsureDeploymentNotFound(t, tc.givenEventing.Name, givenNamespace)
				testEnvironment.EnsureHPANotFound(t, tc.givenEventing.Name, givenNamespace)
				testEnvironment.EnsureK8sServiceNotFound(t,
					eventing.GetEPPPublishServiceName(*tc.givenEventing), givenNamespace)
				testEnvironment.EnsureK8sServiceNotFound(t,
					eventing.GetEPPMetricsServiceName(*tc.givenEventing), givenNamespace)
				testEnvironment.EnsureK8sServiceNotFound(t,
					eventing.GetEPPHealthServiceName(*tc.givenEventing), givenNamespace)
				testEnvironment.EnsureK8sServiceAccountNotFound(t,
					eventing.GetEPPServiceAccountName(*tc.givenEventing), givenNamespace)
				testEnvironment.EnsureK8sClusterRoleNotFound(t,
					eventing.GetEPPClusterRoleName(*tc.givenEventing), givenNamespace)
				testEnvironment.EnsureK8sClusterRoleBindingNotFound(t,
					eventing.GetEPPClusterRoleBindingName(*tc.givenEventing), givenNamespace)
			} else {
				// check if the owner reference is set.
				// if owner reference is set then these resources would be garbage collected in real k8s cluster.
				testEnvironment.EnsureEPPK8sResourcesHaveOwnerReference(t, *tc.givenEventing)
			}
		})
	}
}

// Test_WatcherEventingCRK8sObjects tests that deleting the k8s objects deployed by Eventing CR
// should trigger reconciliation.
func Test_WatcherEventingCRK8sObjects(t *testing.T) {
	t.Parallel()

	type deletionFunc func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error

	deletePublishServiceFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetEPPPublishServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteMetricsServiceFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetEPPMetricsServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteHealthServiceFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetEPPHealthServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteServiceAccountFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceAccountFromK8s(eventing.GetEPPServiceAccountName(eventingCR), eventingCR.Namespace)
	}

	deleteClusterRoleFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteClusterRoleFromK8s(eventing.GetEPPClusterRoleName(eventingCR), eventingCR.Namespace)
	}

	deleteClusterRoleBindingFromK8s := func(env *testutils.TestEnvironment,
		eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteClusterRoleBindingFromK8s(eventing.GetEPPClusterRoleBindingName(eventingCR),
			eventingCR.Namespace)
	}

	testCases := []struct {
		name                 string
		givenEventing        *eventingv1alpha1.Eventing
		wantResourceDeletion []deletionFunc
	}{
		{
			name: "should recreate Publish Service",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deletePublishServiceFromK8s,
			},
		},
		{
			name: "should recreate Metrics Service",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deleteMetricsServiceFromK8s,
			},
		},
		{
			name: "should recreate Health Service",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deleteHealthServiceFromK8s,
			},
		},
		{
			name: "should recreate ServiceAccount",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deleteServiceAccountFromK8s,
			},
		},
		// @TODO: Fix the watching of ClusterRoles and ClusterRoleBindings
		//{
		//	name: "should recreate ClusterRole",
		//	givenEventing: utils.NewEventingCR(
		//		utils.WithEventingCRMinimal(),
		//		utils.WithEventingStreamData("Memory", "1M", "1M", 1, 1),
		//		utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
		//	),
		//	wantResourceDeletion: []deletionFunc{
		//		deleteClusterRoleFromK8s,
		//	},
		//},
		//{
		//	name: "should recreate ClusterRoleBinding",
		//	givenEventing: utils.NewEventingCR(
		//		utils.WithEventingCRMinimal(),
		//		utils.WithEventingStreamData("Memory", "1M", "1M", 1, 1),
		//		utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
		//	),
		//	wantResourceDeletion: []deletionFunc{
		//		deleteClusterRoleBindingFromK8s,
		//	},
		//},
		{
			name: "should recreate all objects",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deletePublishServiceFromK8s,
				deleteMetricsServiceFromK8s,
				deleteHealthServiceFromK8s,
				deleteServiceAccountFromK8s,
				deleteClusterRoleFromK8s,
				deleteClusterRoleBindingFromK8s,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			g := gomega.NewWithT(t)
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return true
			}

			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			nats := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(tc.givenEventing.Namespace),
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			)
			testEnvironment.EnsureK8sResourceCreated(t, nats)
			testEnvironment.EnsureNATSResourceStateReady(t, nats)

			// create Eventing CR
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(matchers.HaveStatusReady())

			// check if EPP resources exists.
			testEnvironment.EnsureEPPK8sResourcesExists(t, *tc.givenEventing)

			// when
			for _, f := range tc.wantResourceDeletion {
				require.NoError(t, f(testEnvironment, *tc.givenEventing))
			}

			// then
			// ensure all k8s objects exists again.
			testEnvironment.EnsureEPPK8sResourcesExists(t, *tc.givenEventing)
		})
	}
}
