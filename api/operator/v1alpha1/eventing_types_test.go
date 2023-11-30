package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncStatusActiveBackend(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name              string
		givenEventing     *Eventing
		wantActiveBackend BackendType
	}{
		{
			name: "it should set ActiveBackend to NATS",
			givenEventing: &Eventing{
				Spec: EventingSpec{
					Backend: &Backend{Type: NatsBackendType},
				},
				Status: EventingStatus{},
			},
			wantActiveBackend: NatsBackendType,
		},
		{
			name: "it should set ActiveBackend to EventMesh",
			givenEventing: &Eventing{
				Spec: EventingSpec{
					Backend: &Backend{Type: EventMeshBackendType},
				},
				Status: EventingStatus{},
			},
			wantActiveBackend: EventMeshBackendType,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// when
			tc.givenEventing.SyncStatusActiveBackend()

			// then
			require.Equal(t, tc.wantActiveBackend, tc.givenEventing.Status.ActiveBackend)
		})
	}
}

func TestIsSpecBackendTypeChanged(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name          string
		givenEventing *Eventing
		wantResult    bool
	}{
		{
			name: "it should return false if backend is not changed",
			givenEventing: &Eventing{
				Spec: EventingSpec{
					Backend: &Backend{Type: NatsBackendType},
				},
				Status: EventingStatus{
					ActiveBackend: NatsBackendType,
				},
			},
			wantResult: false,
		},
		{
			name: "it should return true if backend is changed",
			givenEventing: &Eventing{
				Spec: EventingSpec{
					Backend: &Backend{Type: NatsBackendType},
				},
				Status: EventingStatus{
					ActiveBackend: EventMeshBackendType,
				},
			},
			wantResult: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.wantResult, tc.givenEventing.IsSpecBackendTypeChanged())
		})
	}
}
