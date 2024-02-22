package v1alpha1

import (
	"testing"
	"time"

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
			t.Parallel()
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
			t.Parallel()
			// when
			testcase.givenStatus.SetPublisherService(testcase.givenServiceName, testcase.givenServiceNamespace)

			// then
			require.Equal(t, testcase.wantStatus, testcase.givenStatus)
		})
	}
}

func TestRemoveUnsupportedConditions(t *testing.T) {
	t.Parallel()

	// given
	var (
		// supported conditions
		backendAvailableCondition = kmetav1.Condition{
			Type:               "BackendAvailable",
			Status:             kmetav1.ConditionStatus("BackendAvailableStatus"),
			ObservedGeneration: int64(1),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2001, 0o1, 0o1, 0o1, 0o1, 0o1, 0o00000001, time.UTC)},
			Reason:             "BackendAvailableReason",
			Message:            "BackendAvailableMessage",
		}
		publisherProxyReadyCondition = kmetav1.Condition{
			Type:               "PublisherProxyReady",
			Status:             kmetav1.ConditionStatus("PublisherProxyReadyStatus"),
			ObservedGeneration: int64(2),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2002, 0o2, 0o2, 0o2, 0o2, 0o2, 0o00000002, time.UTC)},
			Reason:             "PublisherProxyReadyReason",
			Message:            "PublisherProxyReadyMessage",
		}
		webhookReadyCondition = kmetav1.Condition{
			Type:               "WebhookReady",
			Status:             kmetav1.ConditionStatus("WebhookReadyStatus"),
			ObservedGeneration: int64(3),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2003, 0o3, 0o3, 0o3, 0o3, 0o3, 0o00000003, time.UTC)},
			Reason:             "WebhookReadyReason",
			Message:            "WebhookReadyMessage",
		}
		subscriptionManagerReadyCondition = kmetav1.Condition{
			Type:               "SubscriptionManagerReady",
			Status:             kmetav1.ConditionStatus("SubscriptionManagerReadyStatus"),
			ObservedGeneration: int64(4),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2004, 0o4, 0o4, 0o4, 0o4, 0o4, 0o00000004, time.UTC)},
			Reason:             "SubscriptionManagerReadyReason",
			Message:            "SubscriptionManagerReadyMessage",
		}
		deletedCondition = kmetav1.Condition{
			Type:               "Deleted",
			Status:             kmetav1.ConditionStatus("DeletedStatus"),
			ObservedGeneration: int64(5),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2005, 0o5, 0o5, 0o5, 0o5, 0o5, 0o00000005, time.UTC)},
			Reason:             "DeletedReason",
			Message:            "DeletedMessage",
		}

		// unsupported conditions
		unsupportedTypeCondition1 = kmetav1.Condition{
			Type:               "Unsupported1",
			Status:             kmetav1.ConditionStatus("UnsupportedStatus1"),
			ObservedGeneration: int64(-1),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2011, 11, 11, 11, 11, 11, 0o00000011, time.UTC)},
			Reason:             "UnsupportedReason1",
			Message:            "UnsupportedMessage1",
		}
		unsupportedTypeCondition2 = kmetav1.Condition{
			Type:               "Unsupported2",
			Status:             kmetav1.ConditionStatus("UnsupportedStatus2"),
			ObservedGeneration: int64(-2),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2012, 12, 12, 12, 12, 12, 0o00000012, time.UTC)},
			Reason:             "UnsupportedReason2",
			Message:            "UnsupportedMessage2",
		}
		unsupportedTypeCondition3 = kmetav1.Condition{
			Type:               "Unsupported3",
			Status:             kmetav1.ConditionStatus("UnsupportedStatus3"),
			ObservedGeneration: int64(-3),
			LastTransitionTime: kmetav1.Time{Time: time.Date(2013, 13, 13, 13, 13, 13, 0o00000013, time.UTC)},
			Reason:             "UnsupportedReason3",
			Message:            "UnsupportedMessage3",
		}
	)

	tests := []struct {
		name        string
		givenStatus *EventingStatus
		wantStatus  *EventingStatus
	}{
		{
			name: "given nil conditions",
			givenStatus: &EventingStatus{
				Conditions: nil,
			},
			wantStatus: &EventingStatus{
				Conditions: nil,
			},
		},
		{
			name: "given empty conditions",
			givenStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{},
			},
			wantStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{},
			},
		},
		{
			name: "given few supported condition",
			givenStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{
					backendAvailableCondition,
					subscriptionManagerReadyCondition,
				},
			},
			wantStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{
					backendAvailableCondition,
					subscriptionManagerReadyCondition,
				},
			},
		},
		{
			name: "given all supported conditions",
			givenStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{
					backendAvailableCondition,
					publisherProxyReadyCondition,
					webhookReadyCondition,
					subscriptionManagerReadyCondition,
					deletedCondition,
				},
			},
			wantStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{
					backendAvailableCondition,
					publisherProxyReadyCondition,
					webhookReadyCondition,
					subscriptionManagerReadyCondition,
					deletedCondition,
				},
			},
		},
		{
			name: "given all unsupported conditions",
			givenStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{
					unsupportedTypeCondition1,
					unsupportedTypeCondition2,
					unsupportedTypeCondition3,
				},
			},
			wantStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{},
			},
		},
		{
			name: "given supported and unsupported conditions",
			givenStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{
					unsupportedTypeCondition1,
					unsupportedTypeCondition2,
					unsupportedTypeCondition3,
					backendAvailableCondition,
					publisherProxyReadyCondition,
					webhookReadyCondition,
					subscriptionManagerReadyCondition,
					deletedCondition,
				},
			},
			wantStatus: &EventingStatus{
				Conditions: []kmetav1.Condition{
					backendAvailableCondition,
					publisherProxyReadyCondition,
					webhookReadyCondition,
					subscriptionManagerReadyCondition,
					deletedCondition,
				},
			},
		},
	}
	for _, tt := range tests {
		ttc := tt
		t.Run(ttc.name, func(t *testing.T) {
			t.Parallel()

			// when
			ttc.givenStatus.RemoveUnsupportedConditions()

			// then
			require.Equal(t, ttc.wantStatus, ttc.givenStatus)
		})
	}
}
