package eventing

import (
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha1"
	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
)

type TestSubscriptionInfo struct {
	Name        string
	Description string
	Source      string
	Types       []string
}

func (s TestSubscriptionInfo) V1Alpha1SpecFilters() []*eventingv1alpha1.EventMeshFilter {
	var filters []*eventingv1alpha1.EventMeshFilter
	for _, etype := range s.Types {
		filter := &eventingv1alpha1.EventMeshFilter{
			EventSource: &eventingv1alpha1.Filter{
				Type:     "exact",
				Property: "source",
				Value:    "",
			},
			EventType: &eventingv1alpha1.Filter{
				Type:     "exact",
				Property: "type",
				Value:    etype,
			},
		}
		filters = append(filters, filter)
	}
	return filters
}

func (s TestSubscriptionInfo) ToSubscriptionV1Alpha1(sink, namespace string) *eventingv1alpha1.Subscription {
	return &eventingv1alpha1.Subscription{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      s.Name,
			Namespace: namespace,
		},
		Spec: eventingv1alpha1.SubscriptionSpec{
			Sink: sink,
			Filter: &eventingv1alpha1.BEBFilters{
				Filters: s.V1Alpha1SpecFilters(),
			},
		},
	}
}

func (s TestSubscriptionInfo) ToSubscriptionV1Alpha2(sink, namespace string) *eventingv1alpha2.Subscription {
	return &eventingv1alpha2.Subscription{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      s.Name,
			Namespace: namespace,
		},
		Spec: eventingv1alpha2.SubscriptionSpec{
			Sink:         sink,
			TypeMatching: eventingv1alpha2.TypeMatchingStandard,
			Source:       s.Source,
			Types:        s.Types,
		},
	}
}
