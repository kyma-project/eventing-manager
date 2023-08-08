package eventing

import (
	"context"
	"testing"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/test/utils"
	"github.com/stretchr/testify/require"
)

func Test_containsFinalizer(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name          string
		givenEventing *eventingv1alpha1.Eventing
		wantResult    bool
	}{
		{
			name:          "should return false when finalizer is missing",
			givenEventing: utils.NewEventingCR(),
			wantResult:    false,
		},
		{
			name:          "should return true when finalizer is present",
			givenEventing: utils.NewEventingCR(utils.WithEventingCRFinalizer(FinalizerName)),
			wantResult:    true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			reconciler := testEnv.Reconciler

			// when, then
			require.Equal(t, tc.wantResult, reconciler.containsFinalizer(tc.givenEventing))
		})
	}
}

func Test_addFinalizer(t *testing.T) {
	t.Parallel()

	t.Run("should add finalizer", func(t *testing.T) {

		// given
		givenEventing := utils.NewEventingCR()

		testEnv := NewMockedUnitTestEnvironment(t, givenEventing)
		reconciler := testEnv.Reconciler

		// when
		_, err := reconciler.addFinalizer(context.Background(), givenEventing)

		// then
		require.NoError(t, err)
		gotEventing, err := testEnv.GetEventing(givenEventing.GetName(), givenEventing.GetNamespace())
		require.NoError(t, err)
		require.True(t, reconciler.containsFinalizer(&gotEventing))
	})
}

func Test_removeFinalizer(t *testing.T) {
	t.Parallel()

	t.Run("should remove finalizer", func(t *testing.T) {

		// given
		givenEventing := utils.NewEventingCR(utils.WithEventingCRFinalizer(FinalizerName))

		testEnv := NewMockedUnitTestEnvironment(t, givenEventing)
		reconciler := testEnv.Reconciler

		// when
		_, err := reconciler.removeFinalizer(context.Background(), givenEventing)

		// then
		require.NoError(t, err)
		gotEventing, err := testEnv.GetEventing(givenEventing.GetName(), givenEventing.GetNamespace())
		require.NoError(t, err)
		require.False(t, reconciler.containsFinalizer(&gotEventing))
	})
}
