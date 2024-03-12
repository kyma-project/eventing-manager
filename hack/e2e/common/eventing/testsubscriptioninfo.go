package eventing

import (
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
)

type TestSubscriptionInfo struct {
	Name         string
	Description  string
	TypeMatching eventingv1alpha2.TypeMatching
	Source       string
	Types        []string
}

func (s TestSubscriptionInfo) ToSubscriptionV1Alpha2(sink, namespace string) *eventingv1alpha2.Subscription {
	return &eventingv1alpha2.Subscription{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      s.Name,
			Namespace: namespace,
		},
		Spec: eventingv1alpha2.SubscriptionSpec{
			Sink:         sink,
			TypeMatching: s.TypeMatching,
			Source:       s.Source,
			Types:        s.Types,
		},
	}
}
