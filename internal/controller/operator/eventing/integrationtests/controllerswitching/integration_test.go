package controllerswitching

import (
	"fmt"
	"os"
	"testing"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/operator/eventing"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
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

func Test_Switching(t *testing.T) {
	// given - common for all test cases.
	setEventMeshSecretConfig := func(eventingCR *operatorv1alpha1.Eventing, name, namespace string) {
		eventingCR.Spec.Backend.Config.EventMeshSecret = fmt.Sprintf("%s/%s", namespace, name)
	}

	// test cases.
	testCases := []struct {
		name                     string
		givenNATS                *natsv1alpha1.NATS
		givenEventMeshSecretName string
		givenEventing            *operatorv1alpha1.Eventing
		givenSwitchedEventing    *operatorv1alpha1.Eventing
		wantPreSwitchMatches     gomegatypes.GomegaMatcher
		wantPostSwitchMatches    gomegatypes.GomegaMatcher
	}{
		{
			name: "should successfully switch from NATS to EventMesh backend",
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			),
			givenEventMeshSecretName: "test-secret-name2",
			givenEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			givenSwitchedEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name2"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			wantPreSwitchMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableCondition(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
				matchers.HaveFinalizer(),
			),
			wantPostSwitchMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveEventMeshSubManagerReadyCondition(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
				matchers.HaveFinalizer(),
			),
		},
		{
			name: "should successfully switch from EventMesh to NATS backend",
			givenNATS: natstestutils.NewNATSCR(
				natstestutils.WithNATSCRDefaults(),
			),
			givenEventMeshSecretName: "test-secret-name2",
			givenEventing: utils.NewEventingCR(
				utils.WithEventMeshBackend("test-secret-name2"),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			givenSwitchedEventing: utils.NewEventingCR(
				utils.WithEventingCRMinimal(),
				utils.WithEventingStreamData("Memory", "1M", 1, 1),
				utils.WithEventingPublisherData(2, 2, "199m", "99Mi", "399m", "199Mi"),
				utils.WithEventingEventTypePrefix("test-prefix"),
				utils.WithEventingDomain(utils.Domain),
			),
			wantPreSwitchMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveEventMeshSubManagerReadyCondition(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
				matchers.HaveFinalizer(),
			),
			wantPostSwitchMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableCondition(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
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
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool {
				return true
			}
			// create unique namespace for this test run.
			givenNamespace := tc.givenEventing.Namespace
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create NATS CR.
			tc.givenNATS.SetNamespace(givenNamespace)
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)
			testEnvironment.EnsureNATSResourceStateReady(t, tc.givenNATS)

			// sync object meta for both Eventing CRs.
			tc.givenSwitchedEventing.ObjectMeta = tc.givenEventing.ObjectMeta
			setEventMeshSecretConfig(tc.givenEventing, tc.givenEventMeshSecretName, givenNamespace)
			setEventMeshSecretConfig(tc.givenSwitchedEventing, tc.givenEventMeshSecretName, givenNamespace)

			// create eventing-webhook-auth secret.
			testEnvironment.EnsureOAuthSecretCreated(t, tc.givenEventing)
			// create EventMesh secret.
			if tc.givenEventing.Spec.Backend.Type == operatorv1alpha1.EventMeshBackendType {
				testEnvironment.EnsureEventMeshSecretCreated(t, tc.givenEventing)
			} else if tc.givenSwitchedEventing.Spec.Backend.Type == operatorv1alpha1.EventMeshBackendType {
				testEnvironment.EnsureEventMeshSecretCreated(t, tc.givenSwitchedEventing)
			}

			// create Eventing CR.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenEventing)

			// ********* before switching checks ********.
			testEnvironment.GetEventingAssert(g, tc.givenEventing).Should(tc.wantPreSwitchMatches)
			ensureEPPDeploymentAndHPAResources(t, tc.givenEventing, testEnvironment)
			ensureK8sResources(t, tc.givenEventing, testEnvironment)

			// get Eventing CR from cluster.
			gotEventing, err := testEnvironment.GetEventingFromK8s(tc.givenEventing.Name, givenNamespace)
			require.NoError(t, err)

			// when: switch backend.
			tc.givenSwitchedEventing.ObjectMeta = gotEventing.ObjectMeta
			testEnvironment.EnsureK8sResourceUpdated(t, tc.givenSwitchedEventing)

			// then
			// ********* after switching checks ********.
			testEnvironment.GetEventingAssert(g, tc.givenSwitchedEventing).Should(tc.wantPostSwitchMatches)
			ensureEPPDeploymentAndHPAResources(t, tc.givenSwitchedEventing, testEnvironment)
			ensureK8sResources(t, tc.givenSwitchedEventing, testEnvironment)
		})
	}
}

func ensureEPPDeploymentAndHPAResources(t *testing.T, givenEventing *operatorv1alpha1.Eventing, testEnvironment *testutilsintegration.TestEnvironment) {
	testEnvironment.EnsureDeploymentExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureHPAExists(t, eventing.GetPublisherDeploymentName(*givenEventing), givenEventing.Namespace)
	testEnvironment.EnsureEventingSpecPublisherReflected(t, givenEventing)
	testEnvironment.EnsureEventingReplicasReflected(t, givenEventing)
	testEnvironment.EnsureDeploymentOwnerReferenceSet(t, givenEventing)
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
