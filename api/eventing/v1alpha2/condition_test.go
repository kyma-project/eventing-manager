package v1alpha2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_makeSubscriptionSpecValidCondition(t *testing.T) {
	t.Parallel()

	var (
		// subscription spec valid condition
		subscriptionSpecValidTrueCondition = Condition{
			Type:               ConditionSubscriptionSpecValid,
			Status:             kcorev1.ConditionTrue,
			LastTransitionTime: kmetav1.Now(),
			Reason:             ConditionReasonSubscriptionSpecHasNoValidationErrors,
			Message:            "",
		}

		// subscription spec invalid condition
		subscriptionSpecValidFalseCondition = Condition{
			Type:               ConditionSubscriptionSpecValid,
			Status:             kcorev1.ConditionFalse,
			LastTransitionTime: kmetav1.Now(),
			Reason:             ConditionReasonSubscriptionSpecHasValidationErrors,
			Message:            "some error",
		}
	)

	tests := []struct {
		name                               string
		givenError                         error
		wantSubscriptionSpecValidCondition Condition
	}{
		{
			name:                               "no error",
			givenError:                         nil,
			wantSubscriptionSpecValidCondition: subscriptionSpecValidTrueCondition,
		},
		{
			name:                               "error",
			givenError:                         errors.New("some error"),
			wantSubscriptionSpecValidCondition: subscriptionSpecValidFalseCondition,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// when
			gotCondition := makeSubscriptionSpecValidCondition(test.givenError)

			// then
			require.True(t, ConditionEquals(gotCondition, test.wantSubscriptionSpecValidCondition))
		})
	}
}
