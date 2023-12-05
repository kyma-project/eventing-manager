package eventmesh

import (
	"testing"
	"time"

	apigatewayv1beta1 "github.com/kyma-project/api-gateway/apis/gateway/v1beta1"
	kymalogger "github.com/kyma-project/kyma/common/logging/logger"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

func Test_isInDeletion(t *testing.T) {
	testCases := []struct {
		name              string
		givenSubscription func() *eventingv1alpha2.Subscription
		wantResult        bool
	}{
		{
			name: "Deletion timestamp uninitialized",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				sub := eventingtesting.NewSubscription("some-name", "some-namespace",
					eventingtesting.WithNotCleanSource(),
					eventingtesting.WithNotCleanType())
				sub.DeletionTimestamp = nil
				return sub
			},
			wantResult: false,
		},
		{
			name: "Deletion timestamp is zero",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				zero := kmetav1.Time{}
				sub := eventingtesting.NewSubscription("some-name", "some-namespace",
					eventingtesting.WithNotCleanSource(),
					eventingtesting.WithNotCleanType())
				sub.DeletionTimestamp = &zero
				return sub
			},
			wantResult: false,
		},
		{
			name: "Deletion timestamp is set to a useful time",
			givenSubscription: func() *eventingv1alpha2.Subscription {
				newTime := kmetav1.NewTime(time.Now())
				sub := eventingtesting.NewSubscription("some-name", "some-namespace",
					eventingtesting.WithNotCleanSource(),
					eventingtesting.WithNotCleanType())
				sub.DeletionTimestamp = &newTime
				return sub
			},
			wantResult: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantResult, isInDeletion(tt.givenSubscription()))
		})
	}
}

func Test_isFinalizerSet(t *testing.T) {
	testCases := []struct {
		name              string
		givenSubscription *eventingv1alpha2.Subscription
		wantResult        bool
	}{
		{
			name:              "Finalizer not set",
			givenSubscription: &eventingv1alpha2.Subscription{},
			wantResult:        false,
		},
		{
			name: "Finalizer is set",
			givenSubscription: &eventingv1alpha2.Subscription{
				ObjectMeta: kmetav1.ObjectMeta{Finalizers: []string{eventingv1alpha2.Finalizer}},
			},
			wantResult: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantResult, isFinalizerSet(tt.givenSubscription))
		})
	}
}

func Test_addFinalizer(t *testing.T) {
	defaultLogger, err := logger.New(string(kymalogger.JSON), string(kymalogger.INFO))
	if err != nil {
		t.Fatalf("initialize logger failed: %v", err)
	}

	namedLogger := defaultLogger.WithContext().Named(reconcilerName)

	testCases := []struct {
		name              string
		givenSubscription *eventingv1alpha2.Subscription
		wantFinalizersLen int
		wantFinalizers    []string
	}{
		{
			name:              "with empty finalizers",
			givenSubscription: &eventingv1alpha2.Subscription{},
			wantFinalizersLen: 1,
			wantFinalizers:    []string{eventingv1alpha2.Finalizer},
		},
		{
			name: "with one finalizers",
			givenSubscription: &eventingv1alpha2.Subscription{
				ObjectMeta: kmetav1.ObjectMeta{Finalizers: []string{eventingv1alpha2.Finalizer}},
			},
			wantFinalizersLen: 2,
			wantFinalizers:    []string{eventingv1alpha2.Finalizer, eventingv1alpha2.Finalizer},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sub := tt.givenSubscription
			err := addFinalizer(sub, namedLogger)
			require.NoError(t, err)
			require.Len(t, sub.Finalizers, tt.wantFinalizersLen)
			require.Equal(t, tt.wantFinalizers, sub.Finalizers)
		})
	}
}

func Test_getSvcNsAndName(t *testing.T) {
	testCases := []struct {
		name          string
		givenURL      string
		wantName      string
		wantNamespace string
		wantError     bool
	}{
		{
			name:          "with complete valid svc url",
			givenURL:      "name1.namespace1.svc.cluster.local",
			wantName:      "name1",
			wantNamespace: "namespace1",
			wantError:     false,
		},
		{
			name:      "with incomplete svc url",
			givenURL:  "cluster",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			namespace, name, err := getSvcNsAndName(tc.givenURL)

			require.Equal(t, tc.wantError, err != nil)
			if !tc.wantError {
				require.Equal(t, tc.wantName, name)
				require.Equal(t, tc.wantNamespace, namespace)
			}
		})
	}
}

func Test_computeAPIRuleReadyStatus(t *testing.T) {
	testCases := []struct {
		name         string
		givenAPIRule *apigatewayv1beta1.APIRule
		wantResult   bool
	}{
		{
			name:         "with uninitialised ApiRule",
			givenAPIRule: &apigatewayv1beta1.APIRule{},
			wantResult:   false,
		},
		{
			name:         "with nil ApiRule",
			givenAPIRule: nil,
			wantResult:   false,
		},
		{
			name: "with nil apiRule.Status.APIRuleStatus",
			givenAPIRule: &apigatewayv1beta1.APIRule{
				Status: apigatewayv1beta1.APIRuleStatus{
					APIRuleStatus: nil,
				},
			},
			wantResult: false,
		},
		{
			name: "with nil apiRule.Status.AccessRuleStatus",
			givenAPIRule: &apigatewayv1beta1.APIRule{
				Status: apigatewayv1beta1.APIRuleStatus{
					AccessRuleStatus: nil,
				},
			},
			wantResult: false,
		},
		{
			name: "with nil apiRule.Status.VirtualServiceStatus",
			givenAPIRule: &apigatewayv1beta1.APIRule{
				Status: apigatewayv1beta1.APIRuleStatus{
					VirtualServiceStatus: nil,
				},
			},
			wantResult: false,
		},
		{
			name: "with StatusOK apiRule",
			givenAPIRule: &apigatewayv1beta1.APIRule{
				Status: apigatewayv1beta1.APIRuleStatus{
					APIRuleStatus: &apigatewayv1beta1.APIRuleResourceStatus{
						Code: apigatewayv1beta1.StatusOK,
					},
					AccessRuleStatus: &apigatewayv1beta1.APIRuleResourceStatus{
						Code: apigatewayv1beta1.StatusOK,
					},
					VirtualServiceStatus: &apigatewayv1beta1.APIRuleResourceStatus{
						Code: apigatewayv1beta1.StatusOK,
					},
				},
			},
			wantResult: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantResult, computeAPIRuleReadyStatus(tt.givenAPIRule))
		})
	}
}

func Test_setSubscriptionStatusExternalSink(t *testing.T) {
	host1 := "kyma-project.io"

	testCases := []struct {
		name              string
		givenSubscription *eventingv1alpha2.Subscription
		givenAPIRule      *apigatewayv1beta1.APIRule
		wantExternalSink  string
		wantError         bool
	}{
		{
			name: "with valid sink and apiRule",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Sink: "http://name1.namespace1.svc.cluster.local/test1",
				},
			},
			givenAPIRule: &apigatewayv1beta1.APIRule{
				Spec: apigatewayv1beta1.APIRuleSpec{
					Host:    &host1,
					Service: &apigatewayv1beta1.Service{},
				},
			},
			wantExternalSink: "https://kyma-project.io/test1",
			wantError:        false,
		},
		{
			name: "with invalid sink and apiRule",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Sink: "name1",
				},
			},
			givenAPIRule: &apigatewayv1beta1.APIRule{
				Spec: apigatewayv1beta1.APIRuleSpec{
					Host:    &host1,
					Service: &apigatewayv1beta1.Service{},
				},
			},
			wantError: true,
		},
		{
			name: "with nil host in apiRule",
			givenSubscription: &eventingv1alpha2.Subscription{
				Spec: eventingv1alpha2.SubscriptionSpec{
					Sink: "http://name1.namespace1.svc.cluster.local/test1",
				},
			},
			givenAPIRule: &apigatewayv1beta1.APIRule{
				Spec: apigatewayv1beta1.APIRuleSpec{
					Host:    nil,
					Service: &apigatewayv1beta1.Service{},
				},
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sub := tc.givenSubscription
			err := setSubscriptionStatusExternalSink(sub, tc.givenAPIRule)

			require.Equal(t, tc.wantError, err != nil)
			if !tc.wantError {
				require.Equal(t, tc.wantExternalSink, sub.Status.Backend.ExternalSink)
			}
		})
	}
}
