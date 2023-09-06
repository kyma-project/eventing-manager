package eventing

import (
	"fmt"
	"testing"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_handleEventingCRAllowedCheck(t *testing.T) {
	t.Parallel()

	givenAllowedEventingCR := testutils.NewEventingCR(
		testutils.WithEventingCRName("eventing"),
		testutils.WithEventingCRNamespace("kyma-system"),
	)

	// define test cases
	testCases := []struct {
		name            string
		givenEventing   *eventingv1alpha1.Eventing
		wantCheckResult bool
	}{
		{
			name: "should allow Eventing CR if name and namespace is correct",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("eventing"),
				testutils.WithEventingCRNamespace("kyma-system"),
			),
			wantCheckResult: true,
		},
		{
			name: "should not allow Eventing CR if name is incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("not-allowed-name"),
				testutils.WithEventingCRNamespace("kyma-system"),
			),
			wantCheckResult: false,
		},
		{
			name: "should not allow Eventing CR if namespace is incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("eventing"),
				testutils.WithEventingCRNamespace("not-allowed-namespace"),
			),
			wantCheckResult: false,
		},
		{
			name: "should not allow Eventing CR if name and namespace, both are incorrect",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingCRName("not-allowed-name"),
				testutils.WithEventingCRNamespace("not-allowed-namespace"),
			),
			wantCheckResult: false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t, tc.givenEventing)
			testEnv.Reconciler.allowedEventingCR = givenAllowedEventingCR
			logger := testEnv.Reconciler.logger.WithContext().Named(ControllerName)

			// when
			result, err := testEnv.Reconciler.handleEventingCRAllowedCheck(testEnv.Context, tc.givenEventing, logger)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantCheckResult, result)

			// if the Eventing CR is not allowed then check if the CR status is correctly updated or not.
			gotEventing, err := testEnv.GetEventing(tc.givenEventing.Name, tc.givenEventing.Namespace)
			require.NoError(t, err)
			if !tc.wantCheckResult {
				// check eventing.status.state
				require.Equal(t, eventingv1alpha1.StateError, gotEventing.Status.State)

				// check eventing.status.conditions
				wantConditions := []metav1.Condition{
					{
						Type:               string(eventingv1alpha1.ConditionPublisherProxyReady),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(eventingv1alpha1.ConditionReasonForbidden),
						Message: fmt.Sprintf("Only a single Eventing CR with name: %s and namespace: %s "+
							"is allowed to be created in a Kyma cluster.", givenAllowedEventingCR.Name,
							givenAllowedEventingCR.Namespace),
					},
				}
				require.True(t, natsv1alpha1.ConditionsEquals(wantConditions, gotEventing.Status.Conditions))
			}
		})
	}
}
