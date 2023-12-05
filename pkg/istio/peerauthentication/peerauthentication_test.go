package peerauthentication

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	"github.com/kyma-project/eventing-manager/test"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
)

func Test_SyncPeerAuthentications(t *testing.T) {
	// given
	emDeployment := testutils.NewDeployment(
		"eventing-manager",
		"kyma-system",
		map[string]string{})
	emDeployment.UID = "1234-56789-9123"

	// Define test cases
	testCases := []struct {
		name                           string
		givenPeerAuthenticationExists  bool
		givenDeploymentExists          bool
		wantPatchApplyCalled           bool
		wantGetDeploymentDynamicCalled bool
		wantError                      error
	}{
		{
			name:                           "should do nothing when CRD does not exists",
			givenPeerAuthenticationExists:  false,
			givenDeploymentExists:          true,
			wantPatchApplyCalled:           false,
			wantGetDeploymentDynamicCalled: false,
			wantError:                      nil,
		},
		{
			name:                           "should fail when deployment does not exists",
			givenPeerAuthenticationExists:  true,
			givenDeploymentExists:          false,
			wantPatchApplyCalled:           false,
			wantGetDeploymentDynamicCalled: true,
			wantError:                      errors.New("eventing-manager deployment not found"),
		},
		{
			name:                           "should succeed when CRD and deployment both exists",
			givenPeerAuthenticationExists:  true,
			givenDeploymentExists:          true,
			wantPatchApplyCalled:           true,
			wantGetDeploymentDynamicCalled: true,
			wantError:                      nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger, err := test.NewEventingLogger()
			require.NoError(t, err)
			ctx := context.Background()

			// define mocks.
			kubeClient := new(k8smocks.Client)
			kubeClient.On("PeerAuthenticationCRDExists",
				ctx).Return(tc.givenPeerAuthenticationExists, nil).Once()

			if tc.wantPatchApplyCalled {
				kubeClient.On("PatchApplyPeerAuthentication", ctx,
					mock.Anything).Return(nil).Twice()
			}

			if tc.givenDeploymentExists && tc.wantGetDeploymentDynamicCalled {
				kubeClient.On("GetDeploymentDynamic", ctx, "eventing-manager",
					"kyma-system").Return(emDeployment, nil).Once()
			} else if tc.wantGetDeploymentDynamicCalled {
				kubeClient.On("GetDeploymentDynamic", ctx, "eventing-manager",
					"kyma-system").Return(nil, nil).Once()
			}

			// when
			err = SyncPeerAuthentications(ctx, kubeClient, logger.WithContext())

			// then
			if tc.wantError != nil {
				require.Equal(t, tc.wantError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			// assert expectations of mock.
			kubeClient.AssertExpectations(t)
		})
	}
}
