package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	kapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/operator/eventing"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils"
	testutilsintegration "github.com/kyma-project/eventing-manager/test/utils/integration"
)

const (
	projectRootDir = "../../../../../../"
)

var testEnvironment *testutilsintegration.TestEnvironment

var ErrPatchApplyFailed = errors.New("patch apply failed")

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = testutilsintegration.NewTestEnvironment(testutilsintegration.TestEnvironmentConfig{
		ProjectRootDir:            projectRootDir,
		CELValidationEnabled:      false,
		APIRuleCRDEnabled:         true,
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

func Test_CreateEventingCR_NATS(t *testing.T) {
	testCases := []struct {
		name                 string
		givenEventing        *operatorv1alpha1.Eventing
		givenNATS            *natsv1alpha1.NATS
		givenDeploymentReady bool
		givenNATSReady       bool
		givenNATSCRDMissing  bool
		wantMatches          gomegatypes.GomegaMatcher
		wantEnsureK8sObjects bool
	}{
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
				matchers.HaveStatusWarning(),
				matchers.HaveNATSNotAvailableConditionWith("NATS module has to be installed: "+
					"customresourcedefinitions.apiextensions.k8s.io \"nats.operator.kyma-project.io\" not found"),
				matchers.HaveFinalizer(),
			),
		},
		{
			name: "Eventing CR should have warning state when backend config is empty",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingEmptyBackend(),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
			),
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusWarning(),
				matchers.HaveBackendNotAvailableConditionWith(operatorv1alpha1.ConditionBackendNotSpecifiedMessage,
					operatorv1alpha1.ConditionReasonBackendNotSpecified),
				matchers.HaveFinalizer(),
			),
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return testcase.givenDeploymentReady
			}
			// create unique namespace for this test run.
			givenNamespace := testcase.givenEventing.Namespace
			testcase.givenNATS.SetNamespace(givenNamespace)

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)
			// when
			// create NATS.
			originalKubeClient := testEnvironment.KubeClient
			if !testcase.givenNATSCRDMissing {
				testEnvironment.EnsureK8sResourceCreated(t, testcase.givenNATS)
				if testcase.givenNATSReady {
					testEnvironment.EnsureNATSResourceStateReady(t, testcase.givenNATS)
				}
			} else {
				mockedKubeClient := &MockKubeClient{
					Client: originalKubeClient,
				}
				testEnvironment.KubeClient = mockedKubeClient
				testEnvironment.Reconciler.SetKubeClient(mockedKubeClient)
			}
			// create Eventing CR.
			testEnvironment.EnsureK8sResourceCreated(t, testcase.givenEventing)

			defer func() {
				testEnvironment.KubeClient = originalKubeClient
				testEnvironment.Reconciler.SetKubeClient(originalKubeClient)
				testEnvironment.EnsureEventingResourceDeletion(t, testcase.givenEventing.Name, givenNamespace)
				if testcase.givenNATSReady && !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)
				}
				if !testcase.givenNATSCRDMissing {
					testEnvironment.EnsureK8sResourceDeleted(t, testcase.givenNATS)
				}
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			// then
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, testcase.givenEventing).Should(testcase.wantMatches)
			if testcase.givenDeploymentReady && testcase.givenEventing.Spec.Backend != nil {
				// check if EPP deployment, HPA resources created and values are reflected including owner reference.
				ensureEPPDeploymentAndHPAResources(t, testcase.givenEventing, testEnvironment)
			}

			if testcase.wantEnsureK8sObjects && testcase.givenEventing.Spec.Backend != nil {
				// check if EPP resources exists.
				ensureK8sResources(t, testcase.givenEventing, testEnvironment)
			}

			// check the publisher service in the Eventing CR status
			testEnvironment.EnsurePublishServiceInEventingStatus(t, testcase.givenEventing.Name, testcase.givenEventing.Namespace)
		})
	}
}

func Test_UpdateEventingCR(t *testing.T) {
	testCases := []struct {
		name                      string
		givenExistingEventing     *operatorv1alpha1.Eventing
		givenNewEventingForUpdate *operatorv1alpha1.Eventing
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return true
			}
			// create unique namespace for this test run.
			givenNamespace := testcase.givenExistingEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			nats := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(testcase.givenExistingEventing.Namespace),
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			)
			testEnvironment.EnsureK8sResourceCreated(t, nats)
			testEnvironment.EnsureNATSResourceStateReady(t, nats)
			testEnvironment.EnsureK8sResourceCreated(t, testcase.givenExistingEventing)
			givenEPPDeploymentName := eventing.GetPublisherDeploymentName(*testcase.givenExistingEventing)
			testEnvironment.EnsureDeploymentExists(t, givenEPPDeploymentName, givenNamespace)
			testEnvironment.EnsureHPAExists(t, givenEPPDeploymentName, givenNamespace)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, testcase.givenExistingEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, givenEPPDeploymentName, givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			// get Eventing CR.
			eventingCR, err := testEnvironment.GetEventingFromK8s(testcase.givenExistingEventing.Name, givenNamespace)
			require.NoError(t, err)

			// when
			// update NATS CR.
			newEventing := eventingCR.DeepCopy()
			newEventing.Spec = testcase.givenNewEventingForUpdate.Spec
			testEnvironment.EnsureK8sResourceUpdated(t, newEventing)

			// then
			testEnvironment.EnsureEventingSpecPublisherReflected(t, newEventing)
			testEnvironment.EnsureEventingReplicasReflected(t, newEventing)
			testEnvironment.EnsureDeploymentOwnerReferenceSet(t, testcase.givenExistingEventing)

			// check the publisher service in the Eventing CR status
			testEnvironment.EnsurePublishServiceInEventingStatus(t, eventingCR.Name, eventingCR.Namespace)
		})
	}
}

func Test_ReconcileSameEventingCR(t *testing.T) {
	t.Parallel()

	// given
	eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool { return true }

	eventingCR := utils.NewEventingCR(
		utils.WithEventingCRMinimal(),
		utils.WithEventingStreamData("Memory", "1M", 1, 1),
		utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
	)

	namespace := eventingCR.GetNamespace()

	natsCR := natstestutils.NewNATSCR(
		natstestutils.WithNATSCRNamespace(namespace),
		natstestutils.WithNATSCRDefaults(),
		natstestutils.WithNATSStateReady(),
	)

	testEnvironment.EnsureNamespaceCreation(t, namespace)
	testEnvironment.EnsureK8sResourceCreated(t, natsCR)
	testEnvironment.EnsureNATSResourceStateReady(t, natsCR)
	testEnvironment.EnsureK8sResourceCreated(t, eventingCR)

	eppDeploymentName := eventing.GetPublisherDeploymentName(*eventingCR)
	testEnvironment.EnsureDeploymentExists(t, eppDeploymentName, namespace)
	testEnvironment.EnsureHPAExists(t, eppDeploymentName, namespace)

	eppDeployment, err := testEnvironment.GetDeploymentFromK8s(eppDeploymentName, namespace)
	require.NoError(t, err)
	require.NotNil(t, eppDeployment)

	defer func() {
		testEnvironment.EnsureEventingResourceDeletion(t, eventingCR.Name, namespace)
		if !*testEnvironment.EnvTestInstance.UseExistingCluster {
			testEnvironment.EnsureDeploymentDeletion(t, eppDeploymentName, namespace)
		}
		testEnvironment.EnsureK8sResourceDeleted(t, natsCR)
		testEnvironment.EnsureNamespaceDeleted(t, namespace)
	}()

	// Ensure reconciling the same Eventing CR multiple times does not update the EPP deployment.
	const runs = 3
	resourceVersionBefore := eppDeployment.ResourceVersion
	for r := range runs {
		// when
		runID := fmt.Sprintf("run-%d", r)

		eventingCR, err = testEnvironment.GetEventingFromK8s(eventingCR.Name, namespace)
		require.NoError(t, err)
		require.NotNil(t, eventingCR)

		eventingCR = eventingCR.DeepCopy()
		eventingCR.Labels = map[string]string{"reconcile": runID} // force new reconciliation
		testEnvironment.EnsureK8sResourceUpdated(t, eventingCR)

		eventingCR, err = testEnvironment.GetEventingFromK8s(eventingCR.Name, namespace)
		require.NoError(t, err)
		require.NotNil(t, eventingCR)
		require.Equal(t, eventingCR.Labels["reconcile"], runID)

		// then
		testEnvironment.EnsureEventingSpecPublisherReflected(t, eventingCR)
		testEnvironment.EnsureEventingReplicasReflected(t, eventingCR)
		testEnvironment.EnsureDeploymentOwnerReferenceSet(t, eventingCR)

		eppDeployment, err = testEnvironment.GetDeploymentFromK8s(eppDeploymentName, namespace)
		require.NoError(t, err)
		require.NotNil(t, eppDeployment)

		resourceVersionAfter := eppDeployment.ResourceVersion
		require.Equal(t, resourceVersionBefore, resourceVersionAfter)

		// check the publisher service in the Eventing CR status
		testEnvironment.EnsurePublishServiceInEventingStatus(t, eventingCR.Name, eventingCR.Namespace)
	}
}

// Test_WatcherEventingCRK8sObjects tests that deleting the k8s objects deployed by Eventing CR
// should trigger reconciliation.
func Test_WatcherEventingCRK8sObjects(t *testing.T) {
	t.Parallel()
	type deletionFunc func(env *testutilsintegration.TestEnvironment, eventingCR operatorv1alpha1.Eventing) error

	deletePublishServiceFromK8s := func(env *testutilsintegration.TestEnvironment, eventingCR operatorv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetPublisherPublishServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteMetricsServiceFromK8s := func(env *testutilsintegration.TestEnvironment, eventingCR operatorv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetPublisherMetricsServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteHealthServiceFromK8s := func(env *testutilsintegration.TestEnvironment, eventingCR operatorv1alpha1.Eventing) error {
		return env.DeleteServiceFromK8s(eventing.GetPublisherHealthServiceName(eventingCR), eventingCR.Namespace)
	}

	deleteServiceAccountFromK8s := func(env *testutilsintegration.TestEnvironment, eventingCR operatorv1alpha1.Eventing) error {
		return env.DeleteServiceAccountFromK8s(eventing.GetPublisherServiceAccountName(eventingCR), eventingCR.Namespace)
	}

	deleteClusterRoleFromK8s := func(env *testutilsintegration.TestEnvironment, eventingCR operatorv1alpha1.Eventing) error {
		return env.DeleteClusterRoleFromK8s(eventing.GetPublisherClusterRoleName(eventingCR), eventingCR.Namespace)
	}

	deleteClusterRoleBindingFromK8s := func(env *testutilsintegration.TestEnvironment,
		eventingCR operatorv1alpha1.Eventing,
	) error {
		return env.DeleteClusterRoleBindingFromK8s(eventing.GetPublisherClusterRoleBindingName(eventingCR),
			eventingCR.Namespace)
	}

	deleteHPAFromK8s := func(env *testutilsintegration.TestEnvironment,
		eventingCR operatorv1alpha1.Eventing,
	) error {
		return env.DeleteHPAFromK8s(eventing.GetPublisherDeploymentName(eventingCR),
			eventingCR.Namespace)
	}

	testCases := []struct {
		name                 string
		givenEventing        *operatorv1alpha1.Eventing
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			if !*testEnvironment.EnvTestInstance.UseExistingCluster && testcase.runForRealCluster {
				t.Skip("Skipping test case as it can only be run on real cluster")
			}

			// given
			g := gomega.NewWithT(t)
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return true
			}

			// create unique namespace for this test run.
			givenNamespace := testcase.givenEventing.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			nats := natstestutils.NewNATSCR(
				natstestutils.WithNATSCRNamespace(testcase.givenEventing.Namespace),
				natstestutils.WithNATSCRDefaults(),
				natstestutils.WithNATSStateReady(),
			)
			testEnvironment.EnsureK8sResourceCreated(t, nats)
			testEnvironment.EnsureNATSResourceStateReady(t, nats)

			// create Eventing CR
			testEnvironment.EnsureK8sResourceCreated(t, testcase.givenEventing)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, testcase.givenEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)
				}
				testEnvironment.EnsureK8sResourceDeleted(t, nats)
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, testcase.givenEventing).Should(matchers.HaveStatusReady())

			// check if EPP resources exists.
			testEnvironment.EnsureEPPK8sResourcesExists(t, *testcase.givenEventing)

			// when
			for _, f := range testcase.wantResourceDeletion {
				require.NoError(t, f(testEnvironment, *testcase.givenEventing))
			}

			// then
			// ensure all k8s objects exists again.
			testEnvironment.EnsureEPPK8sResourcesExists(t, *testcase.givenEventing)
		})
	}
}

func Test_CreateEventingCR_EventMesh(t *testing.T) {
	testCases := []struct {
		name                          string
		givenEventing                 *operatorv1alpha1.Eventing
		givenDeploymentReady          bool
		shouldFailSubManager          bool
		shouldEventMeshSecretNotFound bool
		wantMatches                   gomegatypes.GomegaMatcher
		wantEnsureK8sObjects          bool
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
					"failed to sync Publisher Proxy secret: patch apply failed"),
				matchers.HaveFinalizer(),
			),
			shouldFailSubManager: true,
		},
		{
			name: "Eventing CR should have warning state when EventMesh secret is missing",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name2"),
				utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
			),
			wantMatches: gomega.And(
				matchers.HaveStatusWarning(),
				matchers.HaveEventMeshSubManagerNotReadyCondition(
					eventingcontroller.EventMeshSecretMissingMessage),
				matchers.HaveFinalizer(),
			),
			shouldEventMeshSecretNotFound: true,
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return testcase.givenDeploymentReady
			}

			// create unique namespace for this test run.
			givenNamespace := testcase.givenEventing.Namespace

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create eventing-webhook-auth secret.
			testEnvironment.EnsureOAuthSecretCreated(t, testcase.givenEventing)

			if !testcase.shouldEventMeshSecretNotFound {
				// create EventMesh secret.
				testEnvironment.EnsureEventMeshSecretCreated(t, testcase.givenEventing)
			}

			originalKubeClient := testEnvironment.KubeClient
			if testcase.shouldFailSubManager {
				mockedKubeClient := &MockKubeClient{
					Client: originalKubeClient,
				}
				testEnvironment.KubeClient = mockedKubeClient
				testEnvironment.Reconciler.SetKubeClient(mockedKubeClient)
			}

			// when
			// create Eventing CR.
			testEnvironment.EnsureK8sResourceCreated(t, testcase.givenEventing)

			defer func() {
				testEnvironment.KubeClient = originalKubeClient
				testEnvironment.Reconciler.SetKubeClient(originalKubeClient)

				testEnvironment.EnsureEventingResourceDeletion(t, testcase.givenEventing.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster && !testcase.shouldFailSubManager && !testcase.shouldEventMeshSecretNotFound {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)
				}
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			// then
			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, testcase.givenEventing).Should(testcase.wantMatches)
			if testcase.givenDeploymentReady {
				// check if EPP deployment, HPA resources created and values are reflected including owner reference.
				ensureEPPDeploymentAndHPAResources(t, testcase.givenEventing, testEnvironment)
			}

			if testcase.wantEnsureK8sObjects {
				// check if other EPP resources exists and values are reflected.
				ensureK8sResources(t, testcase.givenEventing, testEnvironment)
			}

			// check the publisher service in the Eventing CR status
			testEnvironment.EnsurePublishServiceInEventingStatus(t, testcase.givenEventing.Name, givenNamespace)
		})
	}
}

func TestUpdateEventingCRFromEmptyToNonEmptyBackend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		givenBackendTypeToUse   operatorv1alpha1.BackendType
		wantMatchesBeforeUpdate gomegatypes.GomegaMatcher
		wantMatchesAfterUpdate  gomegatypes.GomegaMatcher
	}{
		{
			name:                  "update Eventing CR from empty backend to NATS",
			givenBackendTypeToUse: operatorv1alpha1.NatsBackendType,
			wantMatchesBeforeUpdate: gomega.And(
				matchers.HaveBackendNotAvailableConditionWith(
					operatorv1alpha1.ConditionBackendNotSpecifiedMessage,
					operatorv1alpha1.ConditionReasonBackendNotSpecified,
				),
			),
			wantMatchesAfterUpdate: gomega.And(
				matchers.HaveFinalizer(),
				matchers.HaveBackendAvailableConditionWith(
					operatorv1alpha1.ConditionNATSAvailableMessage,
					operatorv1alpha1.ConditionReasonNATSAvailable,
				),
			),
		},
		{
			name:                  "update Eventing CR from empty backend to EventMesh",
			givenBackendTypeToUse: operatorv1alpha1.EventMeshBackendType,
			wantMatchesBeforeUpdate: gomega.And(
				matchers.HaveBackendNotAvailableConditionWith(
					operatorv1alpha1.ConditionBackendNotSpecifiedMessage,
					operatorv1alpha1.ConditionReasonBackendNotSpecified,
				),
			),
			wantMatchesAfterUpdate: gomega.And(
				matchers.HaveFinalizer(),
				matchers.HaveBackendAvailableConditionWith(
					operatorv1alpha1.ConditionEventMeshConfigAvailableMessage,
					operatorv1alpha1.ConditionReasonEventMeshConfigAvailable,
				),
			),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			// Create the Eventing CR with an empty backend

			eventingCR := utils.NewEventingCR(utils.WithEmptyBackend())
			namespace := eventingCR.Namespace
			testEnvironment.EnsureNamespaceCreation(t, namespace)
			testEnvironment.EnsureK8sResourceCreated(t, eventingCR)
			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, eventingCR.Name, namespace)
				testEnvironment.EnsureNamespaceDeleted(t, namespace)
			}()

			testEnvironment.GetEventingAssert(g, eventingCR).Should(test.wantMatchesBeforeUpdate)

			// Update the Eventing CR with a non-empty backend

			eventingCR, err := testEnvironment.GetEventingFromK8s(eventingCR.Name, namespace)
			g.Expect(err).ShouldNot(gomega.HaveOccurred())
			eventingCR = eventingCR.DeepCopy()

			switch test.givenBackendTypeToUse {
			case operatorv1alpha1.NatsBackendType:
				{
					natsCR := natstestutils.NewNATSCR(
						natstestutils.WithNATSCRNamespace(namespace),
						natstestutils.WithNATSCRDefaults(),
						natstestutils.WithNATSStateReady(),
					)
					testEnvironment.EnsureK8sResourceCreated(t, natsCR)
					defer func() { testEnvironment.EnsureK8sResourceDeleted(t, natsCR) }()

					g.Expect(utils.WithNATSBackend()(eventingCR)).ShouldNot(gomega.HaveOccurred())
					testEnvironment.EnsureK8sResourceUpdated(t, eventingCR)
				}
			case operatorv1alpha1.EventMeshBackendType:
				{
					g.Expect(utils.WithEventingDomain(utils.Domain)(eventingCR)).ShouldNot(gomega.HaveOccurred())
					g.Expect(utils.WithEventMeshBackend("test-eventmesh-secret")(eventingCR)).ShouldNot(gomega.HaveOccurred())

					testEnvironment.EnsureEventMeshSecretCreated(t, eventingCR)
					testEnvironment.EnsureOAuthSecretCreated(t, eventingCR)
					testEnvironment.EnsureK8sResourceUpdated(t, eventingCR)
				}
			}

			testEnvironment.GetEventingAssert(g, eventingCR).Should(test.wantMatchesAfterUpdate)
		})
	}
}

func Test_DeleteEventingCR(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		givenEventing     *operatorv1alpha1.Eventing
		givenSubscription *eventingv1alpha2.Subscription
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return true
			}

			// create unique namespace for this test run.
			givenNamespace := testcase.givenEventing.GetNamespace()

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			var nats *natsv1alpha1.NATS
			if testcase.givenEventing.Spec.Backend.Type == operatorv1alpha1.NatsBackendType {
				nats = natstestutils.NewNATSCR(
					natstestutils.WithNATSCRNamespace(testcase.givenEventing.Namespace),
					natstestutils.WithNATSCRDefaults(),
					natstestutils.WithNATSStateReady(),
				)

				testEnvironment.EnsureK8sResourceCreated(t, nats)
				testEnvironment.EnsureNATSResourceStateReady(t, nats)
			} else {
				// create eventing-webhook-auth secret.
				testEnvironment.EnsureOAuthSecretCreated(t, testcase.givenEventing)
				testEnvironment.EnsureEventMeshSecretCreated(t, testcase.givenEventing)
			}
			testEnvironment.EnsureK8sResourceCreated(t, testcase.givenEventing)

			defer func() {
				if testcase.givenSubscription != nil {
					testEnvironment.EnsureSubscriptionResourceDeletion(t, testcase.givenSubscription.Name, testcase.givenSubscription.Namespace)
				}

				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)
				}
				if testcase.givenEventing.Spec.Backend.Type == operatorv1alpha1.NatsBackendType {
					testEnvironment.EnsureK8sResourceDeleted(t, nats)
				}
				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
			}()

			testEnvironment.EnsureDeploymentExists(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)
			testEnvironment.EnsureHPAExists(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)

			if testcase.givenSubscription != nil {
				// create subscriptions if given.
				testEnvironment.EnsureNamespaceCreation(t, testcase.givenSubscription.Namespace)
				testEnvironment.EnsureK8sResourceCreated(t, testcase.givenSubscription)
				testEnvironment.EnsureSubscriptionExists(t, testcase.givenSubscription.Name, testcase.givenSubscription.Namespace)

				// then
				// givenSubscription existence means deletion of Eventing CR should be blocked.
				testEnvironment.EnsureK8sResourceDeleted(t, testcase.givenEventing)
				testEnvironment.GetEventingAssert(g, testcase.givenEventing).Should(testcase.wantMatches)
			} else {
				// then
				// givenSubscription is nil means deletion of Eventing CR should be successful.
				testEnvironment.EnsureEventingResourceDeletion(t, testcase.givenEventing.Name, givenNamespace)

				// then
				if *testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentNotFound(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)
					testEnvironment.EnsureHPANotFound(t, eventing.GetPublisherDeploymentName(*testcase.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceNotFound(t,
						eventing.GetPublisherPublishServiceName(*testcase.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceNotFound(t,
						eventing.GetPublisherMetricsServiceName(*testcase.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceNotFound(t,
						eventing.GetPublisherHealthServiceName(*testcase.givenEventing), givenNamespace)
					testEnvironment.EnsureK8sServiceAccountNotFound(t,
						eventing.GetPublisherServiceAccountName(*testcase.givenEventing), givenNamespace)
				} else {
					// check if the owner reference is set.
					// if owner reference is set then these resources would be garbage collected in real k8s cluster.
					testEnvironment.EnsureEPPK8sResourcesHaveOwnerReference(t, *testcase.givenEventing)
				}
				testEnvironment.EnsureK8sClusterRoleNotFound(t,
					eventing.GetPublisherClusterRoleName(*testcase.givenEventing), givenNamespace)
				testEnvironment.EnsureK8sClusterRoleBindingNotFound(t,
					eventing.GetPublisherClusterRoleBindingName(*testcase.givenEventing), givenNamespace)
			}
		})
	}
}

func Test_HandlingMalformedEventMeshSecret(t *testing.T) {
	testcases := []struct {
		name        string
		givenData   map[string][]byte
		wantMatcher gomegatypes.GomegaMatcher
	}{
		{
			name: "EventingCR should have the `ready` status when EventMesh secret is valid",
			givenData: map[string][]byte{
				"management": []byte("foo"),
				"messaging": []byte(`[
				  {
					"broker": {
					  "type": "bar"
					},
					"oa2": {
					  "clientid": "foo",
					  "clientsecret": "foo",
					  "granttype": "client_credentials",
					  "tokenendpoint": "bar"
					},
					"protocol": [
					  "amqp10ws"
					],
					"uri": "foo"
				  },
				  {
					"broker": {
					  "type": "foo"
					},
					"oa2": {
					  "clientid": "bar",
					  "clientsecret": "bar",
					  "granttype": "client_credentials",
					  "tokenendpoint": "foo"
					},
					"protocol": [
					  "bar"
					],
					"uri": "bar"
				  },
				  {
					"broker": {
					  "type": "foo"
					},
					"oa2": {
					  "clientid": "foo",
					  "clientsecret": "bar",
					  "granttype": "client_credentials",
					  "tokenendpoint": "foo"
					},
					"protocol": [
					  "httprest"
					],
					"uri": "bar"
				  }
				]`),
				"namespace":         []byte("bar"),
				"serviceinstanceid": []byte("foo"),
				"xsappname":         []byte("bar"),
			},
			wantMatcher: gomega.And(
				matchers.HaveStatusReady(),
			),
		},
		{
			name:      "EventingCR should be have the `warning` status when EventMesh secret data is empty",
			givenData: map[string][]byte{},
			wantMatcher: gomega.And(
				matchers.HaveStatusWarning(),
			),
		},
		{
			name: "EventingCR should have the `warning` status when EventMesh secret data misses the `namespace` key",
			givenData: map[string][]byte{
				"management": []byte("foo"),
				"messaging": []byte(`[
				  {
					"broker": {
					  "type": "bar"
					},
					"oa2": {
					  "clientid": "foo",
					  "clientsecret": "foo",
					  "granttype": "client_credentials",
					  "tokenendpoint": "bar"
					},
					"protocol": [
					  "amqp10ws"
					],
					"uri": "foo"
				  },
				  {
					"broker": {
					  "type": "foo"
					},
					"oa2": {
					  "clientid": "bar",
					  "clientsecret": "bar",
					  "granttype": "client_credentials",
					  "tokenendpoint": "foo"
					},
					"protocol": [
					  "bar"
					],
					"uri": "bar"
				  },
				  {
					"broker": {
					  "type": "foo"
					},
					"oa2": {
					  "clientid": "foo",
					  "clientsecret": "bar",
					  "granttype": "client_credentials",
					  "tokenendpoint": "foo"
					},
					"protocol": [
					  "httprest"
					],
					"uri": "bar"
				  }
				]`),
				"serviceinstanceid": []byte("foo"),
				"xsappname":         []byte("bar"),
			},
			wantMatcher: gomega.And(
				matchers.HaveStatusWarning(),
			),
		},
		{
			name: "EventingCR should have the `warning` status when EventMesh secret data misses the `messaging` key",
			givenData: map[string][]byte{
				"management":        []byte("foo"),
				"serviceinstanceid": []byte("foo"),
				"xsappname":         []byte("bar"),
				"namespace":         []byte("bar"),
			},
			wantMatcher: gomega.And(
				matchers.HaveStatusWarning(),
			),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			// Given:
			// We need to mock the deployment readiness check.
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return true
			}

			// Create an eventing CR with EventMesh backend.
			givenEventingCR := utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name2"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			)

			// Create an unique namespace for this test run.
			givenNamespace := givenEventingCR.Namespace
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// Create an eventing-webhook-auth Secret.
			testEnvironment.EnsureOAuthSecretCreated(t, givenEventingCR)

			// Create EventMesh secret. This is the crucial part of the test.
			// First we need to extract the expected Secret name and namespace from the Eventing CR.
			a := strings.Split(givenEventingCR.Spec.Backend.Config.EventMeshSecret, "/")
			name, namespace := a[1], a[0]
			// Now we can assemble the EventMesh Secret with the given data.
			secret := &kcorev1.Secret{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: testcase.givenData,
				Type: "Opaque",
			}
			// Finally, we can create the EventMesh Secret on the cluster.
			testEnvironment.EnsureK8sResourceCreated(t, secret)

			// When:
			// Create the Eventing CR on the cluster.
			testEnvironment.EnsureK8sResourceCreated(t, givenEventingCR)

			// Then:
			// Check if the EventingCR status has the expected status, caused by the EventMesh Secret.
			g := gomega.NewWithT(t)
			testEnvironment.GetEventingAssert(g, givenEventingCR).Should(testcase.wantMatcher)
		})
	}
}

func Test_WatcherNATSResource(t *testing.T) {
	t.Parallel()
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			// given
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return true
			}

			givenNamespace := testcase.givenOriginalNATS.Namespace
			if testcase.givenTargetNATS != nil {
				testcase.givenTargetNATS.Namespace = givenNamespace
				testcase.givenTargetNATS.Name = testcase.givenOriginalNATS.Name
			}

			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create original NATS CR.
			originalNats := testcase.givenOriginalNATS.DeepCopy()
			testEnvironment.EnsureK8sResourceCreated(t, originalNats)

			testEnvironment.EnsureNATSResourceState(t, originalNats, testcase.givenOriginalNATS.Status.State)

			// create Eventing CR.
			var eventingResource *operatorv1alpha1.Eventing
			if testcase.isEventMesh {
				eventingResource = utils.NewEventingCR(
					utils.WithEventingCRNamespace(givenNamespace),
					utils.WithEventMeshBackend("test-secret-name"),
					utils.WithEventingPublisherData(1, 1, "199m", "99Mi", "399m", "199Mi"),
					utils.WithStatusState(testcase.wantedOriginalEventingState),
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
					utils.WithStatusState(testcase.wantedOriginalEventingState),
					utils.WithEventingDomain(utils.Domain),
				)
			}
			testEnvironment.EnsureK8sResourceCreated(t, eventingResource)

			defer func() {
				testEnvironment.EnsureEventingResourceDeletion(t, eventingResource.Name, givenNamespace)
				if !*testEnvironment.EnvTestInstance.UseExistingCluster {
					testEnvironment.EnsureDeploymentDeletion(t, eventing.GetPublisherDeploymentName(*eventingResource), givenNamespace)
				}

				if testcase.givenOriginalNATS != nil && testcase.givenTargetNATS != nil {
					testEnvironment.EnsureK8sResourceDeleted(t, testcase.givenOriginalNATS)
				}

				testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)

				if testcase.isEventMesh {
					testEnvironment.EnsureEventMeshSecretDeleted(t, eventingResource)
				}
			}()

			// update target NATS CR to target NATS CR state
			if testcase.givenOriginalNATS != nil {
				testEnvironment.EnsureNATSResourceState(t, testcase.givenOriginalNATS, testcase.givenOriginalNATS.Status.State)
			}

			// check Eventing CR status.
			testEnvironment.GetEventingAssert(g, eventingResource).Should(testcase.wantOriginalEventingMatches)

			// update target NATS CR to target NATS CR state
			if testcase.givenTargetNATS != nil {
				testEnvironment.EnsureNATSResourceState(t, testcase.givenTargetNATS, testcase.givenTargetNATS.Status.State)
			} else {
				// delete NATS CR
				testEnvironment.EnsureK8sResourceDeleted(t, testcase.givenOriginalNATS)
			}

			// check target Eventing CR status.
			testEnvironment.GetEventingAssert(g, eventingResource).Should(testcase.wantTargetEventingMatches)
		})
	}
}

func ensureEPPDeploymentAndHPAResources(t *testing.T, givenEventing *operatorv1alpha1.Eventing, testEnvironment *testutilsintegration.TestEnvironment) {
	t.Helper()
	testEnvironment.EnsureDeploymentExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureHPAExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureEventingSpecPublisherReflected(t, givenEventing)
	testEnvironment.EnsureEventingReplicasReflected(t, givenEventing)
	testEnvironment.EnsureDeploymentOwnerReferenceSet(t, givenEventing)
	testEnvironment.EnsurePublisherDeploymentENVSet(t, givenEventing)
}

func ensureK8sResources(t *testing.T, givenEventing *operatorv1alpha1.Eventing, testEnvironment *testutilsintegration.TestEnvironment) {
	t.Helper()
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
func (mkc *MockKubeClient) GetCRD(ctx context.Context, name string) (*kapiextensionsv1.CustomResourceDefinition, error) {
	notFoundError := &kerrors.StatusError{
		ErrStatus: kmetav1.Status{
			Code:    http.StatusNotFound,
			Reason:  kmetav1.StatusReasonNotFound,
			Message: fmt.Sprintf("customresourcedefinitions.apiextensions.k8s.io \"%s\" not found", name),
		},
	}
	return nil, notFoundError
}

func (mkc *MockKubeClient) PatchApply(ctx context.Context, object kctrlclient.Object) error {
	return ErrPatchApplyFailed
}
