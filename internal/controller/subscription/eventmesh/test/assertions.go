package test

import (
	"context"
	"log"

	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
)

// getSubscriptionAssert fetches a subscription using the lookupKey and allows making assertions on it.
func getSubscriptionAssert(ctx context.Context, g *gomega.GomegaWithT,
	subscription *eventingv1alpha2.Subscription) gomega.AsyncAssertion {
	return g.Eventually(func() *eventingv1alpha2.Subscription {
		lookupKey := types.NamespacedName{
			Namespace: subscription.Namespace,
			Name:      subscription.Name,
		}
		if err := emTestEnsemble.k8sClient.Get(ctx, lookupKey, subscription); err != nil {
			log.Printf("fetch subscription %s failed: %v", lookupKey.String(), err)
			return &eventingv1alpha2.Subscription{}
		}
		return subscription
	}, bigTimeOut, bigPollingInterval)
}
