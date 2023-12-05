package controller_test

import (
	"context"
	"os"
	"testing"

	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/operator/eventing"

	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/onsi/gomega"

	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils"

	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
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

// Test_DeletionOfPublisherResourcesWhenNATSNotEnabled tests that the publisher proxy resources are deleted when
// NATS modules is disabled. Steps of this test:
// 1. Enable NATS modules.
// 2. Create Eventing CR with NATS backend.
// 3. Make sure publisher proxy resources exists.
// 4. Delete NATS CRD and trigger reconciliation of Eventing CR.
// 5. Verify that the publisher proxy resources are deleted.
func Test_DeletionOfPublisherResourcesWhenNATSNotEnabled(t *testing.T) {
	g := gomega.NewWithT(t)

	// given
	eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool { return true }
	// define CRs.
	givenEventing := utils.NewEventingCR(
		utils.WithEventingCRMinimal(),
		utils.WithEventingStreamData("Memory", "1M", 1, 1),
		utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
		utils.WithEventingEventTypePrefix("test-prefix"),
		utils.WithEventingDomain(utils.Domain),
	)
	givenNATS := natstestutils.NewNATSCR(
		natstestutils.WithNATSCRDefaults(),
		natstestutils.WithNATSCRNamespace(givenEventing.Namespace),
	)
	// create unique namespace for this test run.
	givenNamespace := givenEventing.Namespace
	testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

	// create NATS with Ready status.
	testEnvironment.EnsureK8sResourceCreated(t, givenNATS)
	testEnvironment.EnsureNATSResourceStateReady(t, givenNATS)

	// create Eventing CR.
	testEnvironment.EnsureK8sResourceCreated(t, givenEventing)

	// wait until Eventing CR status is Ready.
	testEnvironment.GetEventingAssert(g, givenEventing).Should(matchers.HaveStatusReady())

	// check if EPP deployment, HPA resources created.
	ensureEPPDeploymentAndHPAResources(t, givenEventing, testEnvironment)
	// check if EPP resources exists.
	ensureK8sResources(t, givenEventing, testEnvironment)

	natsCRD, err := testEnvironment.KubeClient.GetCRD(context.TODO(), k8s.NatsGVK.GroupResource().String())
	require.NoError(t, err)

	// define cleanup.
	defer func() {
		// Important: install NATS CRD again.
		testEnvironment.EnsureCRDCreated(t, natsCRD)
		testEnvironment.EnsureEventingResourceDeletion(t, givenEventing.Name, givenNamespace)
		testEnvironment.EnsureNamespaceDeleted(t, givenNamespace)
	}()

	// when
	// delete NATS CR & CRD
	testEnvironment.EnsureK8sResourceDeleted(t, givenNATS)
	testEnvironment.EnsureNATSCRDDeleted(t)

	// then
	// wait until Eventing CR status is Error.
	testEnvironment.GetEventingAssert(g, givenEventing).Should(gomega.And(
		matchers.HaveStatusError(),
		matchers.HaveNATSNotAvailableConditionWith("NATS module has to be installed: customresourcedefinitions.apiextensions.k8s.io \"nats.operator.kyma-project.io\" not found"),
	))

	// verify that the publisher proxy resources are deleted.
	testEnvironment.EnsureDeploymentNotFound(t, eventing.GetPublisherDeploymentName(*givenEventing), givenNamespace)
	testEnvironment.EnsureHPANotFound(t, eventing.GetPublisherDeploymentName(*givenEventing), givenNamespace)
	testEnvironment.EnsureK8sServiceNotFound(t,
		eventing.GetPublisherPublishServiceName(*givenEventing), givenNamespace)
	testEnvironment.EnsureK8sServiceNotFound(t,
		eventing.GetPublisherMetricsServiceName(*givenEventing), givenNamespace)
	testEnvironment.EnsureK8sServiceNotFound(t,
		eventing.GetPublisherHealthServiceName(*givenEventing), givenNamespace)
	testEnvironment.EnsureK8sServiceAccountNotFound(t,
		eventing.GetPublisherServiceAccountName(*givenEventing), givenNamespace)
}

func ensureEPPDeploymentAndHPAResources(t *testing.T, givenEventing *operatorv1alpha1.Eventing, testEnvironment *testutilsintegration.TestEnvironment) {
	testEnvironment.EnsureDeploymentExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureHPAExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureEventingSpecPublisherReflected(t, givenEventing)
	testEnvironment.EnsureEventingReplicasReflected(t, givenEventing)
	testEnvironment.EnsureDeploymentOwnerReferenceSet(t, givenEventing)
	testEnvironment.EnsurePublisherDeploymentENVSet(t, givenEventing)
}

func ensureK8sResources(t *testing.T, givenEventing *operatorv1alpha1.Eventing, testEnvironment *testutilsintegration.TestEnvironment) {
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
