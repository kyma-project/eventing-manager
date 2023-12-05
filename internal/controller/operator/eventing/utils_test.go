package eventing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/test/utils"
)

func Test_containsFinalizer(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name          string
		givenEventing *operatorv1alpha1.Eventing
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

func TestReconciler_getEventMeshBackendConfigHash(t *testing.T) {
	hash, err := getEventMeshBackendConfigHash("kyma-system/eventing-backend", "sap.kyma.custom", "domain.com")
	assert.NoError(t, err)
	assert.NotEqual(t, 0, hash)
}

func TestReconciler_getEventMeshBackendConfigHash_EnsureConsistencyAndUniqueness(t *testing.T) {
	hash1, err1 := getEventMeshBackendConfigHash("kyma-system/eventing-backend", "sap.kyma.custom", "domain.com")
	hash2, err2 := getEventMeshBackendConfigHash("kyma-system/eventing-backend", "sap.kyma.custom", "domain.com")
	hash3, err3 := getEventMeshBackendConfigHash("kyma-system/eventing-backen", "dsap.kyma.cust", "omdomain.com")
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
}
