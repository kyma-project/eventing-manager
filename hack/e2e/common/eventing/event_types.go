package eventing

import (
	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	ecv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestSubscriptionInfo struct {
	Name        string
	Namespace   string
	Description string
	Source      string
	Types       []string
}

func (s TestSubscriptionInfo) V1Alpha1SpecFilters() []*ecv1alpha1.EventMeshFilter {
	var filters []*ecv1alpha1.EventMeshFilter
	for _, etype := range s.Types {
		filter := &ecv1alpha1.EventMeshFilter{
			EventSource: &ecv1alpha1.Filter{
				Type:     "exact",
				Property: "source",
				Value:    "",
			},
			EventType: &ecv1alpha1.Filter{
				Type:     "exact",
				Property: "type",
				Value:    etype,
			},
		}
		filters = append(filters, filter)
	}
	return filters
}

func (s TestSubscriptionInfo) ToSubscriptionV1Alpha1(sink, namespace string) *ecv1alpha1.Subscription {
	return &ecv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: namespace,
		},
		Spec: ecv1alpha1.SubscriptionSpec{
			Sink: sink,
			Filter: &ecv1alpha1.BEBFilters{
				Filters: s.V1Alpha1SpecFilters(),
			},
		},
	}
}

func (s TestSubscriptionInfo) ToSubscriptionV1Alpha2(sink, namespace string) *ecv1alpha2.Subscription {
	return &ecv1alpha2.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: namespace,
		},
		Spec: ecv1alpha2.SubscriptionSpec{
			Sink:         sink,
			TypeMatching: ecv1alpha2.TypeMatchingStandard,
			Source:       s.Source,
			Types:        s.Types,
		},
	}
}
