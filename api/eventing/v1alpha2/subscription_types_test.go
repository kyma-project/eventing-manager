package v1alpha2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/env"
)

func TestGetMaxInFlightMessages(t *testing.T) {
	defaultSubConfig := env.DefaultSubscriptionConfig{MaxInFlightMessages: 5}
	testCases := []struct {
		name              string
		givenSubscription *v1alpha2.Subscription
		wantResult        int
	}{
		{
			name: "function should give the default MaxInFlight if the Subscription config is missing",
			givenSubscription: &v1alpha2.Subscription{
				Spec: v1alpha2.SubscriptionSpec{
					Config: nil,
				},
			},
			wantResult: defaultSubConfig.MaxInFlightMessages,
		},
		{
			name: "function should give the default MaxInFlight if it is missing in the Subscription config",
			givenSubscription: &v1alpha2.Subscription{
				Spec: v1alpha2.SubscriptionSpec{
					Config: map[string]string{
						"otherConfigKey": "20",
					},
				},
			},
			wantResult: defaultSubConfig.MaxInFlightMessages,
		},
		{
			name: "function should give the expectedConfig",
			givenSubscription: &v1alpha2.Subscription{
				Spec: v1alpha2.SubscriptionSpec{
					Config: map[string]string{
						v1alpha2.MaxInFlightMessages: "20",
					},
				},
			},
			wantResult: 20,
		},
		{
			name: "function should result into an error",
			givenSubscription: &v1alpha2.Subscription{
				Spec: v1alpha2.SubscriptionSpec{
					Config: map[string]string{
						v1alpha2.MaxInFlightMessages: "nonInt",
					},
				},
			},
			wantResult: defaultSubConfig.MaxInFlightMessages,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			result := testcase.givenSubscription.GetMaxInFlightMessages(&defaultSubConfig)

			assert.Equal(t, testcase.wantResult, result)
		})
	}
}

func TestGetDuplicateTypes(t *testing.T) {
	tests := []struct {
		name               string
		givenTypes         []string
		wantDuplicateTypes []string
	}{
		{
			name:               "with nil types",
			givenTypes:         nil,
			wantDuplicateTypes: nil,
		},
		{
			name:               "with empty types",
			givenTypes:         []string{},
			wantDuplicateTypes: []string{},
		},
		{
			name: "with one type",
			givenTypes: []string{
				"type0",
			},
			wantDuplicateTypes: []string{},
		},
		{
			name: "with multiple types and no duplicates",
			givenTypes: []string{
				"type0",
				"type1",
				"type2",
			},
			wantDuplicateTypes: []string{},
		},
		{
			name: "with multiple types and consequent duplicates",
			givenTypes: []string{
				"type0",
				"type1", "type1", "type1", // duplicates
				"type2", "type2", // duplicates
				"type3",
				"type4", "type4", "type4", "type4", // duplicates
				"type5",
			},
			wantDuplicateTypes: []string{
				"type1", "type2", "type4",
			},
		},
		{
			name: "with multiple types and non-consequent duplicates",
			givenTypes: []string{
				"type5", "type0", "type1", "type2",
				"type1", // duplicate
				"type3", "type4",
				"type5", // duplicate
				"type6",
				"type4", "type2", // duplicates
			},
			wantDuplicateTypes: []string{
				"type1", "type5", "type4", "type2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := v1alpha2.Subscription{Spec: v1alpha2.SubscriptionSpec{Types: tt.givenTypes}}
			gotDuplicateTypes := s.GetDuplicateTypes()
			assert.Equal(t, tt.wantDuplicateTypes, gotDuplicateTypes)
		})
	}
}
