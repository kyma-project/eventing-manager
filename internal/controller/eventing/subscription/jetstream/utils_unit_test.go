package jetstream

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

func Test_isInDeletion(t *testing.T) {
	testCases := []struct {
		name       string
		givenSub   *v1alpha2.Subscription
		wantResult bool
	}{
		{
			name:       "subscription with no deletion timestamp",
			givenSub:   eventingtesting.NewSubscription("test", "test"),
			wantResult: false,
		},
		{
			name: "subscription with deletion timestamp",
			givenSub: eventingtesting.NewSubscription("test", "test",
				eventingtesting.WithNonZeroDeletionTimestamp()),
			wantResult: true,
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			// when
			result := isInDeletion(tc.givenSub)

			// then
			require.Equal(t, tc.wantResult, result)
		})
	}
}

func Test_containsFinalizer(t *testing.T) {
	testCases := []struct {
		name       string
		givenSub   *v1alpha2.Subscription
		wantResult bool
	}{
		{
			name: "subscription containing finalizer",
			givenSub: eventingtesting.NewSubscription("test", "test",
				eventingtesting.WithFinalizers([]string{v1alpha2.Finalizer})),
			wantResult: true,
		},
		{
			name: "subscription containing finalizer",
			givenSub: eventingtesting.NewSubscription("test", "test",
				eventingtesting.WithFinalizers([]string{"invalid"})),
			wantResult: false,
		},
		{
			name:       "subscription not containing finalizer",
			givenSub:   eventingtesting.NewSubscription("test", "test"),
			wantResult: false,
		},
	}
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			// when
			result := containsFinalizer(tc.givenSub)

			// then
			require.Equal(t, tc.wantResult, result)
		})
	}
}
