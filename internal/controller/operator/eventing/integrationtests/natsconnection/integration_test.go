package natsconnection

import (
	"testing"

	natstestutils "github.com/kyma-project/nats-manager/testutils"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	natsconnectionerrors "github.com/kyma-project/eventing-manager/internal/connection/nats/errors"
	natsconnectionmocks "github.com/kyma-project/eventing-manager/internal/connection/nats/mocks"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/operator/eventing"
	"github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils"
	testutilsintegration "github.com/kyma-project/eventing-manager/test/utils/integration"
)

// Test_NATSConnection tests the Eventing CR status when connecting to NATS.
//
//nolint:tparallel // lets not make this test parallel for now
func Test_NATSConnection(t *testing.T) {
	// given

	ErrAny := errors.New("any")

	testCases := []struct {
		name                    string
		givenNATSConnectionMock func() *natsconnectionmocks.Connection
		wantMatches             gomegatypes.GomegaMatcher
	}{
		{
			name: "Eventing CR should be in ready state if connected to NATS",
			givenNATSConnectionMock: func() *natsconnectionmocks.Connection {
				conn := &natsconnectionmocks.Connection{}
				conn.On("Connect", mock.Anything, mock.Anything).Return(nil)
				conn.On("IsConnected").Return(true)
				return conn
			},
			wantMatches: gomega.And(
				matchers.HaveStatusReady(),
				matchers.HaveNATSAvailableCondition(),
				matchers.HavePublisherProxyReadyConditionDeployed(),
				matchers.HaveFinalizer(),
			),
		},
		{
			name: "Eventing CR should be in warning state if the connect behaviour returned a cannot connect error",
			givenNATSConnectionMock: func() *natsconnectionmocks.Connection {
				conn := &natsconnectionmocks.Connection{}
				conn.On("Connect", mock.Anything, mock.Anything).Return(natsconnectionerrors.ErrCannotConnect)
				conn.On("IsConnected").Return(false)
				return conn
			},
			wantMatches: gomega.And(
				matchers.HaveStatusWarning(),
				matchers.HaveBackendNotAvailableConditionWith(
					natsconnectionerrors.ErrCannotConnect.Error(),
					operatorv1alpha1.ConditionReasonNATSNotAvailable,
				),
				matchers.HaveFinalizer(),
			),
		},
		{
			name: "Eventing CR should be in warning state if the connect behaviour returned any error",
			givenNATSConnectionMock: func() *natsconnectionmocks.Connection {
				conn := &natsconnectionmocks.Connection{}
				conn.On("Connect", mock.Anything, mock.Anything).Return(ErrAny)
				conn.On("IsConnected").Return(false)
				return conn
			},
			wantMatches: gomega.And(
				matchers.HaveStatusWarning(),
				matchers.HaveBackendNotAvailableConditionWith(
					ErrAny.Error(),
					operatorv1alpha1.ConditionReasonNATSNotAvailable,
				),
				matchers.HaveFinalizer(),
			),
		},
	}

	for _, tc := range testCases {
		tcc := tc

		t.Run(tcc.name, func(t *testing.T) {
			t.Parallel()

			// setup environment
			testEnvironment, err := testutilsintegration.NewTestEnvironment(
				testutilsintegration.TestEnvironmentConfig{
					NATSCRDEnabled: true,
					ProjectRootDir: "../../../../../../",
				},
				tcc.givenNATSConnectionMock(),
			)
			require.NoError(t, err)
			defer func() { require.NoError(t, testEnvironment.TearDown()) }() // always cleanup
			eventingcontroller.IsDeploymentReady = func(deployment *kappsv1.Deployment) bool { return true }

			// prepare resources
			natsCR := natstestutils.NewNATSCR(natstestutils.WithNATSCRDefaults())
			eventingCR := utils.NewEventingCR(utils.WithEventingCRMinimal(), utils.WithEventingDomain(utils.Domain))
			natsCR.SetNamespace(eventingCR.Namespace)

			// create resources
			testEnvironment.EnsureNamespaceCreation(t, eventingCR.Namespace)
			testEnvironment.EnsureK8sResourceCreated(t, natsCR)
			testEnvironment.EnsureK8sResourceCreated(t, eventingCR)

			// then
			testEnvironment.GetEventingAssert(gomega.NewWithT(t), eventingCR).Should(tcc.wantMatches)
		})
	}
}
