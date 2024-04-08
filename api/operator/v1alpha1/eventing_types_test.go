package v1alpha1

import (
	"testing"


	"github.com/stretchr/testify/assert"
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// when
			testcase.givenEventing.SyncStatusActiveBackend()

			// then
			require.Equal(t, testcase.wantActiveBackend, testcase.givenEventing.Status.ActiveBackend)
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testcase.wantResult, testcase.givenEventing.IsSpecBackendTypeChanged())
		})
	}
}

func Test_getSupportedConditionsTypes(t *testing.T) {
	want := map[ConditionType]interface{}{
		ConditionBackendAvailable:         nil,
		ConditionPublisherProxyReady:      nil,
		ConditionWebhookReady:             nil,
		ConditionSubscriptionManagerReady: nil,
		ConditionDeleted:                  nil,
	}
	got := getSupportedConditionsTypes()
	require.Equal(t, want, got)
}

func TestHasEmptyBackend(t *testing.T) {
	tests := []struct {
		name                string
		givenEventingSpec   EventingSpec
		wantHasEmptyBackend bool
	}{
		{
			name: "with nil backend",
			givenEventingSpec: EventingSpec{
				Backend: nil,
			},
			wantHasEmptyBackend: true,
		},
		{
			name: "with empty backend type",
			givenEventingSpec: EventingSpec{
				Backend: &Backend{
					Type: "",
				},
			},
			wantHasEmptyBackend: true,
		},
		{
			name: "with non-empty backend type all whitespaces",
			givenEventingSpec: EventingSpec{
				Backend: &Backend{
					Type: "   ",
				},
			},
			wantHasEmptyBackend: true,
		},
		{
			name: "with non-empty backend type",
			givenEventingSpec: EventingSpec{
				Backend: &Backend{
					Type: "any",
				},
			},
			wantHasEmptyBackend: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.givenEventingSpec.HasEmptyBackend()
			assert.Equal(t, tt.wantHasEmptyBackend, got)
		})
	}
}
