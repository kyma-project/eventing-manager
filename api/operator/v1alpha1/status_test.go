package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClearConditions(t *testing.T) {
	t.Parallel()

	// given
	givenEventingStatus := &EventingStatus{
		Conditions: []kmetav1.Condition{
			{
				Type: "NATS",
			},
			{
				Type: "EventMesh",
			},
		},
	}
	require.NotEmpty(t, givenEventingStatus.Conditions)

	// when
	givenEventingStatus.ClearConditions()

	// then
	require.Empty(t, givenEventingStatus.Conditions)
}

func TestClearPublisherService(t *testing.T) {
	// given
	t.Parallel()
	testCases := []struct {
		name                  string
		givenStatus           EventingStatus
		givenServiceName      string
		givenServiceNamespace string
		wantStatus            EventingStatus
	}{
		{
			name: "should clear the publisher service",
			givenStatus: EventingStatus{
				PublisherService: "test-service.test-namespace",
			},
			givenServiceName:      "test-service",
			givenServiceNamespace: "test-namespace",
			wantStatus: EventingStatus{
				PublisherService: "",
			},
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			testcase.givenStatus.ClearPublisherService()

			// then
			require.Equal(t, testcase.wantStatus, testcase.givenStatus)
		})
	}
}

func TestSetPublisherService(t *testing.T) {
	// given
	t.Parallel()
	testCases := []struct {
		name                  string
		givenStatus           EventingStatus
		givenServiceName      string
		givenServiceNamespace string
		wantStatus            EventingStatus
	}{
		{
			name: "should set the correct publisher service",
			givenStatus: EventingStatus{
				PublisherService: "",
			},
			givenServiceName:      "test-service",
			givenServiceNamespace: "test-namespace",
			wantStatus: EventingStatus{
				PublisherService: "test-service.test-namespace",
			},
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			testcase.givenStatus.SetPublisherService(testcase.givenServiceName, testcase.givenServiceNamespace)

			// then
			require.Equal(t, testcase.wantStatus, testcase.givenStatus)
		})
	}
}
