package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/kyma-project/eventing-manager/pkg/k8s"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils"
	testutils "github.com/kyma-project/eventing-manager/test/utils/integration"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"

	eventinv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	v1 "k8s.io/api/apps/v1"
)

const (
	projectRootDir  = "../../../../../"
	eventTypePrefix = "test-prefix"
)

var testEnvironment *testutils.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = testutils.NewTestEnvironment(testutils.TestEnvironmentConfig{
		ProjectRootDir:            projectRootDir,
		CELValidationEnabled:      false,
		APIRuleCRDEnabled:         true,
		ApplicationRuleCRDEnabled: true,
		NATSCRDEnabled:            true,
		AllowedEventingCR:         nil,
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

func Test_CreateEventingCR_NATS(t *testing.T) {
	testCases := []struct {
		name                 string
		givenEventing        *eventingv1alpha1.Eventing
		givenNATS            *natsv1alpha1.NATS
		givenDeploymentReady bool
		givenNATSReady       bool
		givenNATSCRDMissing  bool
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
				matchers.HaveNATSNotAvailableCondition(),
				matchers.HaveFinalizer(),
			),
		},
		{
			name: "Eventing CR should have ready state when all deployment replicas are ready",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			),
			givenDeploymentReady: true,
			givenNATSReady:       true,
			wantMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableCondition(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
				matchers.HaveFinalizer(),
			),
			wantEnsureK8sObjects: true,
		},
		{
			name: "Eventing CR should have processing state when deployment is not ready yet",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingDomain(utils.Domain),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			),
			givenDeploymentReady: false,
			givenNATSReady:       true,
			wantMatches: gomega.And(
				matchers.HaveStatusProcessing(),
				matchers.HaveNATSAvailableCondition(),
				matchers.HavePublisherProxyReadyConditionProcessing(),
				matchers.HaveFinalizer(),
			),
		},
		{
			name: "Eventing CR should error state due to NATS CRD is missing",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			),
			givenNATSCRDMissing: true,
			wantMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HaveNATSNotAvailableConditionWith("NATS module has to be installed: "+
					"customresourcedefinitions.apiextensions.k8s.io \"nats.operator.kyma-project.io\" not found"),
				matchers.HaveFinalizer(),
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
			originalKubeClient := testEnvironment.KubeClient
			if !tc.givenNATSCRDMissing {
				testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)
				if tc.givenNATSReady {
					testEnvironment.EnsureNATSResourceStateReady(t, tc.givenNATS)
				}
			} else {
				mockedKubeClient := &MockKubeClient{
					Client: originalKubeClient,
				}
				testEnvironment.KubeClient = mockedKubeClient
				testEnvironment.Reconciler.SetKubeClient(mockedKubeClient)
			}
			// create Eventing CR.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			defer func() {
				testEnvironment.KubeClient = originalKubeClient
				testEnvironment.Reconciler.SetKubeClient(originalKubeClient)
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)
				if tc.givenNATSReady && !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)
				}
				if !tc.givenNATSCRDMissing {
					testEnvironment.EnsureK8sResourceDeleted(t, tc.givenNATS)
				}
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			// then
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantMatches)
			if tc.givenDeploymentReady {
				// check if EPP deployment, HPA resources created and values are reflected including owner reference.
				ensureEPPDeploymentAndHPAResources(t, tc.givenEventing, testEnvironment)
				// TODO: ensure NATS Backend config is reflected. Done as subscription controller is implemented.
			}

			if tc.wantEnsureK8sObjects {
				// check if EPP resources exists.
				ensureK8sResources(t, tc.givenEventing, testEnvironment)
				// check if webhook configurations are updated with correct CABundle.
				testEnvironment.EnsureCABundleInjectedIntoWebhooks(t)
			}
		})
	}
}

func Test_UpdateEventingCR(t *testing.T) {
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
			givenEPPDeploymentName := eventing.GetPublisherDeploymentName(*tc.givenExistingEventing)
			testEnvironment.EnsureDeploymentExists(t, givenEPPDeploymentName, givenNamespace)
			testEnvironment.EnsureHPAExists(t, givenEPPDeploymentName, givenNamespace)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenExistingEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, givenEPPDeploymentName, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
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

// Test_WatcherEventingCRK8sObjects tests that deleting the k8s objects deployed by Eventing CR
// should trigger reconciliation.
func Test_WatcherEventingCRK8sObjects(t *testing.T) {
	type deletionFunc func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error

	deletePublishServiceFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetPublisherPublishServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteMetricsServiceFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetPublisherMetricsServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteHealthServiceFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetPublisherHealthServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteServiceAccountFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteServiceAccountFromK8s(eventing.GetPublisherServiceAccountName(eventingCR), eventingCR.Namespace)
	}

	deleteClusterRoleFromK8s := func(env *testutils.TestEnvironment, eventingCR eventingv1alpha1.Eventing) error {
		return env.DeleteClusterRoleFromK8s(eventing.GetPublisherClusterRoleName(eventingCR), eventingCR.Namespace)
	}

	deleteClusterRoleBindingFromK8s := func(env *testutils.TestEnvironment,
		eventingCR eventingv1alpha1.Eventing,
	) error {
		return env.DeleteClusterRoleBindingFromK8s(eventing.GetPublisherClusterRoleBindingName(eventingCR),
			eventingCR.Namespace)
	}

	deleteHPAFromK8s := func(env *testutils.TestEnvironment,
		eventingCR eventingv1alpha1.Eventing,
	) error {
		return env.DeleteHPAFromK8s(eventing.GetPublisherDeploymentName(eventingCR),
			eventingCR.Namespace)
	}

	testCases := []struct {
		name                 string
		givenEventing        *eventingv1alpha1.Eventing
		wantResourceDeletion []deletionFunc
		runForRealCluster    bool
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
		{
			name: "should recreate HPA",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deleteHPAFromK8s,
			},
		},
		{
			name: "should recreate ClusterRole",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deleteClusterRoleFromK8s,
			},
			runForRealCluster: true,
		},
		{
			name: "should recreate ClusterRoleBinding",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			wantResourceDeletion: []deletionFunc{
				deleteClusterRoleBindingFromK8s,
			},
			runForRealCluster: true,
		},
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

			if !*testEnvironment.EnvTestInstance.UseExistingCluster && tc.runForRealCluster {
				t.Skip("Skipping test case as it can only be run on real cluster")
			}

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

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

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

func Test_CreateEventingCR_EventMesh(t *testing.T) {
	testCases := []struct {
		name                 string
		givenEventing        *eventingv1alpha1.Eventing
		givenDeploymentReady bool
		shouldFailSubManager bool
		wantMatches          gomegatypes.GomegaMatcher
		wantEnsureK8sObjects bool
	}{
		{
			name: "Eventing CR should have error state when subscription manager is not ready",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name1"),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HaveEventMeshSubManagerNotReadyCondition(
					"failed to get EventMesh secret: Secret \"test-secret-name1\" not found"),
				matchers.HaveFinalizer(),
			),
			shouldFailSubManager: true,
		},
		{
			name: "Eventing CR should have ready state when all deployment replicas are ready",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name2"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			givenDeploymentReady: true,
			wantMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveEventMeshSubManagerReadyCondition(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
				matchers.HaveFinalizer(),
			),
			wantEnsureK8sObjects: true,
		},
		{
			name: "Eventing CR should have processing state when deployment is not ready yet",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name3"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusProcessing(),
				matchers.HaveEventMeshSubManagerReadyCondition(),
				matchers.HavePublisherProxyReadyConditionProcessing(),
				matchers.HaveFinalizer(),
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

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create eventing-webhook-auth secret.
			testEnvironment.EnsureOAuthSecretCreated(t, tc.givenEventing)

			if !tc.shouldFailSubManager {
				// create EventMesh secret.
				testEnvironment.EnsureEventMeshSecretCreated(t, tc.givenEventing)
			}

			// when
			// create Eventing CR.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster && !tc.shouldFailSubManager {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)
				}
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			// then
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantMatches)
			if tc.givenDeploymentReady {
				// check if EPP deployment, HPA resources created and values are reflected including owner reference.
				ensureEPPDeploymentAndHPAResources(t, tc.givenEventing, testEnvironment)
				// TODO: ensure NATS Backend config is reflected. Done as subscription controller is implemented.
			}

			if tc.wantEnsureK8sObjects {
				// check if other EPP resources exists and values are reflected.
				ensureK8sResources(t, tc.givenEventing, testEnvironment)
				// check if webhook configurations are updated with correct CABundle.
				testEnvironment.EnsureCABundleInjectedIntoWebhooks(t)
			}
		})
	}
}

func Test_DeleteEventingCR(t *testing.T) {
	testCases := []struct {
		name              string
		givenEventing     *eventingv1alpha1.Eventing
		givenSubscription *eventinv1alpha2.Subscription
		wantMatches       gomegatypes.GomegaMatcher
	}{
		{
			name: "Delete Eventing CR should delete the owned resources",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
		},
		{
			name: "Delete EventMesh Eventing CR should delete the owned resources",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name"),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
		},
		{
			name: "Delete should be blocked as subscription exists for NATS",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
			),
			givenSubscription: utils.NewSubscription("test-nats-subscription", "test-nats-namespace"),
			wantMatches: gomega.And(
				matchers.HaveStatusWarning(),
				matchers.HaveDeletionErrorCondition(eventingcontroller.SubscriptionExistsErrMessage),
				matchers.HaveFinalizer(),
			),
		},
		{
			name: "Delete should be blocked as subscription exist for EventMesh",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name"),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			givenSubscription: utils.NewSubscription("test-eventmesh-subscription", "test-eventmesh-namespace"),
			wantMatches: gomega.And(
				matchers.HaveStatusWarning(),
				matchers.HaveDeletionErrorCondition(eventingcontroller.SubscriptionExistsErrMessage),
				matchers.HaveFinalizer(),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return true
			}

			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			var nats *natsv1alpha1.NATS
			if tc.givenEventing.Spec.Backend.Type == eventingv1alpha1.NatsBackendType {
				nats = natstestutils.NewNATSCR(
					natstestutils.WithNATSCRNamespace(tc.givenEventing.Namespace),
					natstestutils.WithNATSCRDefaults(),
					natstestutils.WithNATSStateReady(),
				)

				testEnvironment.EnsureK8sResourceCreated(t, nats)
				testEnvironment.EnsureNATSResourceStateReady(t, nats)
			} else {
				// create eventing-webhook-auth secret.
				testEnvironment.EnsureOAuthSecretCreated(t, tc.givenEventing)
				testEnvironment.EnsureEventMeshSecretCreated(t, tc.givenEventing)
			}
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			defer func() {
				if tc.givenSubscription != nil {
					testEnvironment.EnsureSubscriptionResourceDeletion(t, tc.givenSubscription.Name, tc.givenSubscription.Namespace)
				}

				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)
				}
				if tc.givenEventing.Spec.Backend.Type == eventingv1alpha1.NatsBackendType {
					testEnvironment.EnsureK8sResourceDeleted(t, nats)
				}
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			testEnvironment.EnsureDeploymentExists(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)
			testEnvironment.EnsureHPAExists(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)

			if tc.givenSubscription != nil {
				// create subscriptions if given.
				testEnvironment.EnsureNamespaceCreation(t, tc.givenSubscription.Namespace)
				testEnvironment.EnsureK8sResourceCreated(t, tc.givenSubscription)
				testEnvironment.EnsureSubscriptionExists(t, tc.givenSubscription.Name, tc.givenSubscription.Namespace)

				// then
				// givenSubscription existence means deletion of Eventing CR should be blocked.
				testEnvironment.EnsureK8sResourceDeleted(t, tc.givenEventing)
				testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantMatches)
			} else {
				// then
				// givenSubscription is nil means deletion of Eventing CR should be successful.
				testEnvironment.EnsureEventingResourceDeletion(t, tc.givenEventing.Name, givenNamespace)

				// then
				if *testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentNotFound(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)
					testEnvironment.EnsureHPANotFound(t, eventing.GetPublisherDeploymentName(*tc.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceNotFound(t,
						eventing.GetPublisherPublishServiceName(*tc.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceNotFound(t,
						eventing.GetPublisherMetricsServiceName(*tc.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceNotFound(t,
						eventing.GetPublisherHealthServiceName(*tc.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceAccountNotFound(t,
						eventing.GetPublisherServiceAccountName(*tc.givenEventing), givenNamespace)
				} else {
					// check if the owner reference is set.
					// if owner reference is set then these resources would be garbage collected in real k8s cluster.
					testEnvironment.EnsureEPPK8sResourcesHaveOwnerReference(t, *tc.givenEventing)
				}
				testEnvironment.EnsureK8sClusterRoleNotFound(t,
					eventing.GetPublisherClusterRoleName(*tc.givenEventing), givenNamespace)
				testEnvironment.EnsureK8sClusterRoleBindingNotFound(t,
					eventing.GetPublisherClusterRoleBindingName(*tc.givenEventing), givenNamespace)
			}
		})
	}
}

func Test_WatcherNATSResource(t *testing.T) {
	testCases := []struct {
		name                        string
		givenOriginalNATS           *natsv1alpha1.NATS
		givenTargetNATS             *natsv1alpha1.NATS
		isEventMesh                 bool
		wantedOriginalEventingState string
		wantedTargetEventingState   string
		wantOriginalEventingMatches gomegatypes.GomegaMatcher
		wantTargetEventingMatches   gomegatypes.GomegaMatcher
	}{
		{
			name: "should update Eventing CR state if NATS CR state changes from ready to error",
			givenOriginalNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			),
			givenTargetNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateError(),
			),
			wantOriginalEventingMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableCondition(),
			),
			wantTargetEventingMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HaveNATSNotAvailableCondition(),
			),
		},
		{
			name: "should update Eventing CR state if NATS CR state changes from error to ready",
			givenOriginalNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateError(),
			),
			givenTargetNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			),
			wantOriginalEventingMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HaveNATSNotAvailableCondition(),
			),
			wantTargetEventingMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableCondition(),
			),
		},
		{
			name: "should update Eventing CR state to error when NATS CR is deleted",
			givenOriginalNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			),
			givenTargetNATS: nil, // means, NATS CR is deleted.
			wantOriginalEventingMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableCondition(),
			),
			wantTargetEventingMatches: gomega.And(
				matchers.HaveStatusError(),
				matchers.HaveNATSNotAvailableCondition(),
			),
		},
		{
			name: "should not reconcile Eventing CR in EventMesh mode",
			givenOriginalNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			),
			givenTargetNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateError(),
			),
			isEventMesh: true,
			wantOriginalEventingMatches: gomega.And(
				matchers.HaveStatusReady(),
			),
			wantTargetEventingMatches: gomega.And(
				matchers.HaveStatusReady(),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *v1.Deployment) bool {
				return true
			}

			givenNamespace := tc.givenOriginalNATS.Namespace
			if tc.givenTargetNATS != nil {
				tc.givenTargetNATS.Namespace = givenNamespace
				tc.givenTargetNATS.Name = tc.givenOriginalNATS.Name
			}

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create NATS CR.
			originalNats := tc.givenOriginalNATS.DeepCopy()
			testEnvironment.EnsureK8sResourceCreated(t, originalNats)
			// create original NATS CR.

			if tc.givenOriginalNATS.Status.State == natsv1alpha1.StateReady {
				testEnvironment.EnsureNATSResourceStateReady(t, originalNats)
			} else if tc.givenOriginalNATS.Status.State == natsv1alpha1.StateError {
				testEnvironment.EnsureNATSResourceStateError(t, originalNats)
			}

			// create Eventing CR.
			var eventingResource *eventingv1alpha1.Eventing
			if tc.isEventMesh {
				eventingResource = utils.NewEventingCR(
					utils.WithEventingCRNamespace(givenNamespace),
					utils.WithEventMeshBackend("test-secret-name"),
					utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
					utils.WithStatusState(tc.wantedOriginalEventingState),
					utils.WithEventingDomain(utils.Domain),
				)
				// create necessary EventMesh secrets
				testEnvironment.EnsureOAuthSecretCreated(t, eventingResource)
				testEnvironment.EnsureEventMeshSecretCreated(t, eventingResource)
			} else {
				eventingResource = utils.NewEventingCR(
					utils.WithEventingCRNamespace(givenNamespace),
					utils.WithEventingCRMinimal(),
					utils.WithEventingStreamData("Memory", "1M", 1, 1),
					utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
					utils.WithStatusState(tc.wantedOriginalEventingState),
					utils.WithEventingDomain(utils.Domain),
				)
			}
			testEnvironment.EnsureK8sResourceCreated(t, eventingResource)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, eventingResource.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*eventingResource), givenNamespace)
				}

				if tc.givenOriginalNATS != nil && tc.givenTargetNATS != nil {
					testEnvironment.EnsureK8sResourceDeleted(t, tc.givenOriginalNATS)
				}

				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)

				if tc.isEventMesh {
					testEnvironment.EnsureEventMeshSecretDeleted(t, eventingResource)
				}
			}()

			// update target NATS CR to target NATS CR state
			if tc.givenOriginalNATS != nil {
				if tc.givenOriginalNATS.Status.State == natsv1alpha1.StateReady {
					testEnvironment.EnsureNATSResourceStateReady(t, tc.givenOriginalNATS)
				} else if tc.givenOriginalNATS.Status.State == natsv1alpha1.StateError {
					testEnvironment.EnsureNATSResourceStateError(t, tc.givenOriginalNATS)
				}
			}

			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, eventingResource).Should(tc.wantOriginalEventingMatches)

			// update target NATS CR to target NATS CR state
			if tc.givenTargetNATS != nil {
				if tc.givenTargetNATS.Status.State == natsv1alpha1.StateReady {
					testEnvironment.EnsureNATSResourceStateReady(t, tc.givenTargetNATS)
				} else if tc.givenTargetNATS.Status.State == natsv1alpha1.StateError {
					testEnvironment.EnsureNATSResourceStateError(t, tc.givenTargetNATS)
				}
			} else {
				// delete NATS CR
				testEnvironment.EnsureK8sResourceDeleted(t, tc.givenOriginalNATS)
			}

			// check target Eventing CR status.
			testEnvironment.GetEventingAssert(g, eventingResource).Should(tc.wantTargetEventingMatches)
		})
	}
}

func ensureEPPDeploymentAndHPAResources(t *testing.T, givenEventing *eventingv1alpha1.Eventing, testEnvironment *testutils.TestEnvironment) {
	testEnvironment.EnsureDeploymentExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureHPAExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureEventingSpecPublisherReflected(t, givenEventing)
	testEnvironment.EnsureEventingReplicasReflected(t, givenEventing)
	testEnvironment.EnsureDeploymentOwnerReferenceSet(t, givenEventing)
	testEnvironment.EnsurePublisherDeploymentENVSet(t, givenEventing)
}

func ensureK8sResources(t *testing.T, givenEventing *eventingv1alpha1.Eventing, testEnvironment *testutils.TestEnvironment) {
	testEnvironment.EnsureEPPK8sResourcesExists(t, *givenEventing)

	// check if the owner reference is set.
	testEnvironment.EnsureEPPK8sResourcesHaveOwnerReference(t, *givenEventing)

	// check if EPP resources are correctly created.
	deployment, err := testEnvironment.GetDeploymentFromK8s(eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	require.NoError(t, err)
	// K8s Services
	testEnvironment.EnsureEPPPublishServiceCorrect(t, deployment, *givenEventing)
	testEnvironment.EnsureEPPMetricsServiceCorrect(t, deployment, *givenEventing)
	testEnvironment.EnsureEPPHealthServiceCorrect(t, deployment, *givenEventing)
	// ClusterRole
	testEnvironment.EnsureEPPClusterRoleCorrect(t, *givenEventing)
	// ClusterRoleBinding
	testEnvironment.EnsureEPPClusterRoleBindingCorrect(t, *givenEventing)
}

type MockKubeClient struct {
	k8s.Client
}

// mock only GetCRD and leave the rest as is.
func (mkc *MockKubeClient) GetCRD(ctx context.Context, name string) (*apiextensionsv1.CustomResourceDefinition, error) {
	notFoundError := &errors.StatusError{
		ErrStatus: metav1.Status{
			Code:    http.StatusNotFound,
			Reason:  metav1.StatusReasonNotFound,
			Message: fmt.Sprintf("customresourcedefinitions.apiextensions.k8s.io \"%s\" not found", name),
		},
	}
	return nil, notFoundError
}
