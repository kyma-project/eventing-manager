package v1alpha2_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"

	. "github.com/onsi/gomega"
)

func Test_InitializeSubscriptionConditions(t *testing.T) {
	tests := []struct {
		name            string
		givenConditions []v1alpha2.Condition
	}{
		{
			name:            "Conditions empty",
			givenConditions: v1alpha2.MakeSubscriptionConditions(),
		},
		{
			name: "Conditions partially initialized",
			givenConditions: []v1alpha2.Condition{
				{
					Type:               v1alpha2.ConditionSubscribed,
					LastTransitionTime: kmetav1.Now(),
					Status:             kcorev1.ConditionUnknown,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			g := NewGomegaWithT(t)
			subStatus := v1alpha2.SubscriptionStatus{}
			subStatus.Conditions = tt.givenConditions
			wantConditionTypes := []v1alpha2.ConditionType{
				v1alpha2.ConditionSubscribed,
				v1alpha2.ConditionSubscriptionActive,
				v1alpha2.ConditionAPIRuleStatus,
				v1alpha2.ConditionWebhookCallStatus,
				v1alpha2.ConditionSubscriptionSpecValid,
			}

			// when
			subStatus.InitializeConditions()

			// then
			g.Expect(subStatus.Conditions).To(HaveLen(len(wantConditionTypes)))
			foundConditionTypes := make([]v1alpha2.ConditionType, 0)
			for _, condition := range subStatus.Conditions {
				g.Expect(condition.Status).To(BeEquivalentTo(kcorev1.ConditionUnknown))
				foundConditionTypes = append(foundConditionTypes, condition.Type)
			}
			g.Expect(wantConditionTypes).To(ConsistOf(foundConditionTypes))
		})
	}
}

func Test_IsReady(t *testing.T) {
	testCases := []struct {
		name            string
		givenConditions []v1alpha2.Condition
		wantReadyStatus bool
	}{
		{
			name:            "should not be ready if conditions are nil",
			givenConditions: nil,
			wantReadyStatus: false,
		},
		{
			name:            "should not be ready if conditions are empty",
			givenConditions: []v1alpha2.Condition{{}},
			wantReadyStatus: false,
		},
		{
			name: "should not be ready if only ConditionSubscribed is available and true",
			givenConditions: []v1alpha2.Condition{{
				Type:   v1alpha2.ConditionSubscribed,
				Status: kcorev1.ConditionTrue,
			}},
			wantReadyStatus: false,
		},
		{
			name: "should not be ready if only ConditionSubscriptionActive is available and true",
			givenConditions: []v1alpha2.Condition{{
				Type:   v1alpha2.ConditionSubscriptionActive,
				Status: kcorev1.ConditionTrue,
			}},
			wantReadyStatus: false,
		},
		{
			name: "should not be ready if only ConditionAPIRuleStatus is available and true",
			givenConditions: []v1alpha2.Condition{{
				Type:   v1alpha2.ConditionAPIRuleStatus,
				Status: kcorev1.ConditionTrue,
			}},
			wantReadyStatus: false,
		},
		{
			name: "should not be ready if all conditions are unknown",
			givenConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionUnknown},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionUnknown},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionUnknown},
			},
			wantReadyStatus: false,
		},
		{
			name: "should not be ready if all conditions are false",
			givenConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionFalse},
			},
			wantReadyStatus: false,
		},
		{
			name: "should be ready if all conditions are true",
			givenConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionSubscriptionSpecValid, Status: kcorev1.ConditionTrue},
			},
			wantReadyStatus: true,
		},
	}

	status := v1alpha2.SubscriptionStatus{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status.Conditions = tc.givenConditions
			if gotReadyStatus := status.IsReady(); tc.wantReadyStatus != gotReadyStatus {
				t.Errorf("Subscription status is not valid, want: %v but got: %v", tc.wantReadyStatus, gotReadyStatus)
			}
		})
	}
}

func Test_FindCondition(t *testing.T) {
	currentTime := kmetav1.NewTime(time.Now())

	testCases := []struct {
		name              string
		givenConditions   []v1alpha2.Condition
		findConditionType v1alpha2.ConditionType
		wantCondition     *v1alpha2.Condition
	}{
		{
			name: "should be able to find the present condition",
			givenConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime},
			},
			findConditionType: v1alpha2.ConditionSubscriptionActive,
			wantCondition: &v1alpha2.Condition{
				Type:   v1alpha2.ConditionSubscriptionActive,
				Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime,
			},
		},
		{
			name: "should not be able to find the non-present condition",
			givenConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionTrue, LastTransitionTime: currentTime},
			},
			findConditionType: v1alpha2.ConditionSubscriptionActive,
			wantCondition:     nil,
		},
	}

	status := v1alpha2.SubscriptionStatus{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status.Conditions = tc.givenConditions

			if gotCondition := status.FindCondition(tc.findConditionType); !reflect.DeepEqual(tc.wantCondition, gotCondition) {
				t.Errorf("Subscription FindCondition failed, want: %v but got: %v", tc.wantCondition, gotCondition)
			}
		})
	}
}

func Test_ShouldUpdateReadyStatus(t *testing.T) {
	testCases := []struct {
		name                   string
		subscriptionReady      bool
		subscriptionConditions []v1alpha2.Condition
		wantStatus             bool
	}{
		{
			name:              "should not update if the subscription is ready and the conditions are ready",
			subscriptionReady: true,
			subscriptionConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionSubscriptionSpecValid, Status: kcorev1.ConditionTrue},
			},
			wantStatus: false,
		},
		{
			name:              "should not update if the subscription is not ready and the conditions are not ready",
			subscriptionReady: false,
			subscriptionConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionFalse},
			},
			wantStatus: false,
		},
		{
			name:              "should update if the subscription is not ready and the conditions are ready",
			subscriptionReady: false,
			subscriptionConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionSubscriptionSpecValid, Status: kcorev1.ConditionTrue},
			},
			wantStatus: true,
		},
		{
			name:              "should update if the subscription is ready and the conditions are not ready",
			subscriptionReady: true,
			subscriptionConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionFalse},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionFalse},
			},
			wantStatus: true,
		},
		{
			name:              "should update if the subscription is ready and some of the conditions are missing",
			subscriptionReady: true,
			subscriptionConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionUnknown},
			},
			wantStatus: true,
		},
		{
			name:              "should not update if the subscription is not ready and some of the conditions are missing",
			subscriptionReady: false,
			subscriptionConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionUnknown},
			},
			wantStatus: false,
		},
		{
			name:              "should update if the subscription is ready and the status of the conditions are unknown",
			subscriptionReady: true,
			subscriptionConditions: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionUnknown},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionUnknown},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionUnknown},
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionUnknown},
			},
			wantStatus: true,
		},
	}

	status := v1alpha2.SubscriptionStatus{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status.Conditions = tc.subscriptionConditions
			status.Ready = tc.subscriptionReady
			if gotStatus := status.ShouldUpdateReadyStatus(); tc.wantStatus != gotStatus {
				t.Errorf("ShouldUpdateReadyStatus is not valid, want: %v but got: %v", tc.wantStatus, gotStatus)
			}
		})
	}
}

func Test_conditionsEquals(t *testing.T) {
	testCases := []struct {
		name            string
		conditionsSet1  []v1alpha2.Condition
		conditionsSet2  []v1alpha2.Condition
		wantEqualStatus bool
	}{
		{
			name: "should not be equal if the number of conditions are not equal",
			conditionsSet1: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
			},
			conditionsSet2:  []v1alpha2.Condition{},
			wantEqualStatus: false,
		},
		{
			name: "should be equal if the conditions are the same",
			conditionsSet1: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
			},
			conditionsSet2: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
			},
			wantEqualStatus: true,
		},
		{
			name: "should not be equal if the condition types are different",
			conditionsSet1: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
			},
			conditionsSet2: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionWebhookCallStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionSubscriptionActive, Status: kcorev1.ConditionTrue},
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the condition types are the same but the status is different",
			conditionsSet1: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
			},
			conditionsSet2: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionFalse},
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the condition types are different but the status is the same",
			conditionsSet1: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionFalse},
			},
			conditionsSet2: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the condition types are different and an empty key is referenced",
			conditionsSet1: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
			},
			conditionsSet2: []v1alpha2.Condition{
				{Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue},
				{Type: v1alpha2.ConditionControllerReady, Status: kcorev1.ConditionTrue},
			},
			wantEqualStatus: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if gotEqualStatus := v1alpha2.ConditionsEquals(tc.conditionsSet1, tc.conditionsSet2); tc.wantEqualStatus != gotEqualStatus {
				t.Errorf("The list of conditions are not equal, want: %v but got: %v", tc.wantEqualStatus, gotEqualStatus)
			}
		})
	}
}

func Test_conditionEquals(t *testing.T) {
	testCases := []struct {
		name            string
		condition1      v1alpha2.Condition
		condition2      v1alpha2.Condition
		wantEqualStatus bool
	}{
		{
			name: "should not be equal if the types are the same but the status is different",
			condition1: v1alpha2.Condition{
				Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue,
			},

			condition2: v1alpha2.Condition{
				Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionUnknown,
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the types are different but the status is the same",
			condition1: v1alpha2.Condition{
				Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue,
			},

			condition2: v1alpha2.Condition{
				Type: v1alpha2.ConditionAPIRuleStatus, Status: kcorev1.ConditionTrue,
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the message fields are different",
			condition1: v1alpha2.Condition{
				Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue, Message: "",
			},

			condition2: v1alpha2.Condition{
				Type: v1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue, Message: "some message",
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the reason fields are different",
			condition1: v1alpha2.Condition{
				Type:   v1alpha2.ConditionSubscribed,
				Status: kcorev1.ConditionTrue,
				Reason: v1alpha2.ConditionReasonSubscriptionDeleted,
			},

			condition2: v1alpha2.Condition{
				Type:   v1alpha2.ConditionSubscribed,
				Status: kcorev1.ConditionTrue,
				Reason: v1alpha2.ConditionReasonSubscriptionActive,
			},
			wantEqualStatus: false,
		},
		{
			name: "should be equal if all the fields are the same",
			condition1: v1alpha2.Condition{
				Type:    v1alpha2.ConditionAPIRuleStatus,
				Status:  kcorev1.ConditionFalse,
				Reason:  v1alpha2.ConditionReasonAPIRuleStatusNotReady,
				Message: "API Rule is not ready",
			},
			condition2: v1alpha2.Condition{
				Type:    v1alpha2.ConditionAPIRuleStatus,
				Status:  kcorev1.ConditionFalse,
				Reason:  v1alpha2.ConditionReasonAPIRuleStatusNotReady,
				Message: "API Rule is not ready",
			},
			wantEqualStatus: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if gotEqualStatus := v1alpha2.ConditionEquals(tc.condition1, tc.condition2); tc.wantEqualStatus != gotEqualStatus {
				t.Errorf("The conditions are not equal, want: %v but got: %v", tc.wantEqualStatus, gotEqualStatus)
			}
		})
	}
}

func Test_CreateMessageForConditionReasonSubscriptionCreated(t *testing.T) {
	testCases := []struct {
		name      string
		givenName string
		wantName  string
	}{
		{
			name:      "with name 1",
			givenName: "test-one",
			wantName:  "EventMesh subscription name is: test-one",
		},
		{
			name:      "with name 2",
			givenName: "test-second",
			wantName:  "EventMesh subscription name is: test-second",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantName, v1alpha2.CreateMessageForConditionReasonSubscriptionCreated(tc.givenName))
		})
	}
}

func Test_SetConditionSubscriptionActive(t *testing.T) {
	err := errors.New("some error")
	conditionReady := v1alpha2.MakeCondition(
		v1alpha2.ConditionSubscriptionActive,
		v1alpha2.ConditionReasonNATSSubscriptionActive,
		kcorev1.ConditionTrue, "")
	conditionReady.LastTransitionTime = kmetav1.NewTime(time.Now().AddDate(0, 0, -1))
	conditionNotReady := v1alpha2.MakeCondition(
		v1alpha2.ConditionSubscriptionActive,
		v1alpha2.ConditionReasonNATSSubscriptionNotActive,
		kcorev1.ConditionFalse, err.Error())
	conditionNotReady.LastTransitionTime = kmetav1.NewTime(time.Now().AddDate(0, 0, -2))
	sub := eventingtesting.NewSubscription("test", "test")

	testCases := []struct {
		name                   string
		givenConditions        []v1alpha2.Condition
		givenError             error
		wantConditions         []v1alpha2.Condition
		wantLastTransitionTime *kmetav1.Time
	}{
		{
			name:            "no error should set the condition to ready",
			givenError:      nil,
			givenConditions: []v1alpha2.Condition{conditionNotReady},
			wantConditions:  []v1alpha2.Condition{conditionReady},
		},
		{
			name:            "error should set the condition to not ready",
			givenError:      err,
			givenConditions: []v1alpha2.Condition{conditionReady},
			wantConditions:  []v1alpha2.Condition{conditionNotReady},
		},
		{
			name:            "the same condition should not change the lastTransitionTime in case of Sub active",
			givenError:      nil,
			givenConditions: []v1alpha2.Condition{conditionReady},
			wantConditions: []v1alpha2.Condition{{
				Type:               v1alpha2.ConditionSubscriptionActive,
				Reason:             v1alpha2.ConditionReasonNATSSubscriptionActive,
				Status:             kcorev1.ConditionTrue,
				LastTransitionTime: kmetav1.Now(),
			}},
			wantLastTransitionTime: &conditionReady.LastTransitionTime,
		},
		{
			name:            "the same condition should not change the lastTransitionTime in case of error",
			givenError:      err,
			givenConditions: []v1alpha2.Condition{conditionNotReady},
			wantConditions: []v1alpha2.Condition{{
				Type:               v1alpha2.ConditionSubscriptionActive,
				Reason:             v1alpha2.ConditionReasonNATSSubscriptionNotActive,
				Status:             kcorev1.ConditionFalse,
				Message:            err.Error(),
				LastTransitionTime: kmetav1.Now(),
			}},
			wantLastTransitionTime: &conditionNotReady.LastTransitionTime,
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// given
			sub.Status.Conditions = testcase.givenConditions

			// when
			v1alpha2.SetSubscriptionActiveCondition(&sub.Status, testcase.givenError)

			// then
			require.True(t, v1alpha2.ConditionsEquals(sub.Status.Conditions, testcase.wantConditions))
			if testcase.wantLastTransitionTime != nil {
				require.Equal(t, *testcase.wantLastTransitionTime, sub.Status.Conditions[0].LastTransitionTime)
			}
		})
	}
}

func TestSetSubscriptionSpecValidCondition(t *testing.T) {
	t.Parallel()

	var (
		// subscription spec valid conditions
		subscriptionSpecValidTrueCondition = v1alpha2.Condition{
			Type:               v1alpha2.ConditionSubscriptionSpecValid,
			Status:             kcorev1.ConditionTrue,
			LastTransitionTime: kmetav1.Now(),
			Reason:             v1alpha2.ConditionReasonSubscriptionSpecHasNoValidationErrors,
			Message:            "",
		}
		subscriptionSpecValidFalseCondition = v1alpha2.Condition{
			Type:               v1alpha2.ConditionSubscriptionSpecValid,
			Status:             kcorev1.ConditionFalse,
			LastTransitionTime: kmetav1.Now(),
			Reason:             v1alpha2.ConditionReasonSubscriptionSpecHasValidationErrors,
			Message:            "some error",
		}

		// test conditions
		testCondition0 = v1alpha2.Condition{
			Type:               v1alpha2.ConditionType("test-0"),
			Status:             kcorev1.ConditionTrue,
			LastTransitionTime: kmetav1.Now(),
			Reason:             "test-0",
			Message:            "test-0",
		}
		testCondition1 = v1alpha2.Condition{
			Type:               v1alpha2.ConditionType("test-1"),
			Status:             kcorev1.ConditionTrue,
			LastTransitionTime: kmetav1.Now(),
			Reason:             "test-1",
			Message:            "test-1",
		}
	)

	tests := []struct {
		name                    string
		givenSubscriptionStatus v1alpha2.SubscriptionStatus
		givenError              error
		wantSubscriptionStatus  v1alpha2.SubscriptionStatus
	}{
		{
			name: "add SubscriptionSpecValid condition to nil conditions",
			givenSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: nil,
			},
			givenError: nil,
			wantSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					subscriptionSpecValidTrueCondition,
				},
			},
		},
		{
			name: "add SubscriptionSpecValid condition to empty conditions",
			givenSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{},
			},
			givenError: nil,
			wantSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					subscriptionSpecValidTrueCondition,
				},
			},
		},
		{
			name: "add SubscriptionSpecValid condition and preserve other conditions",
			givenSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					testCondition0,
					testCondition1,
				},
			},
			givenError: nil,
			wantSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					testCondition0,
					testCondition1,
					subscriptionSpecValidTrueCondition,
				},
			},
		},
		{
			name: "update existing SubscriptionSpecValid condition to true and preserve other conditions",
			givenSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					testCondition0,
					testCondition1,
					subscriptionSpecValidFalseCondition,
				},
			},
			givenError: nil,
			wantSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					testCondition0,
					testCondition1,
					subscriptionSpecValidTrueCondition,
				},
			},
		},
		{
			name: "update existing SubscriptionSpecValid condition to false and preserve other conditions",
			givenSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					testCondition0,
					testCondition1,
					subscriptionSpecValidTrueCondition,
				},
			},
			givenError: errors.New("some error"),
			wantSubscriptionStatus: v1alpha2.SubscriptionStatus{
				Conditions: []v1alpha2.Condition{
					testCondition0,
					testCondition1,
					subscriptionSpecValidFalseCondition,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// when
			test.givenSubscriptionStatus.SetSubscriptionSpecValidCondition(test.givenError)
			gotConditions := test.givenSubscriptionStatus.Conditions
			wantConditions := test.wantSubscriptionStatus.Conditions

			// then
			require.True(t, v1alpha2.ConditionsEquals(gotConditions, wantConditions))
		})
	}
}
