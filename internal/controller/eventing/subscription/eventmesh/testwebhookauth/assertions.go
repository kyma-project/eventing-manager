package testwebhookauth

import (
	"context"
	"log"

	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
)

// getSubscriptionAssert fetches a subscription using the lookupKey and allows making assertions on it.
func getSubscriptionAssert(ctx context.Context, g *gomega.GomegaWithT,
	subscription *eventingv1alpha2.Subscription,
	ensemble *eventMeshTestEnsemble,
) gomega.AsyncAssertion {
	return g.Eventually(func() *eventingv1alpha2.Subscription {
		lookupKey := types.NamespacedName{
			Namespace: subscription.Namespace,
			Name:      subscription.Name,
		}
		if err := ensemble.k8sClient.Get(ctx, lookupKey, subscription); err != nil {
			log.Printf("fetch subscription %s failed: %v", lookupKey.String(), err)
			return &eventingv1alpha2.Subscription{}
		}
		return subscription
	}, bigTimeOut, bigPollingInterval)
}

// getAPIRuleAssert fetches an apiRule and allows making assertions on it.
func getAPIRuleAssert(ctx context.Context, g *gomega.GomegaWithT,
	apiRule *apigatewayv2.APIRule,
	ensemble *eventMeshTestEnsemble,
) gomega.AsyncAssertion {
	return g.Eventually(func() apigatewayv2.APIRule {
		fetchedAPIRule, err := getAPIRule(ctx, ensemble, apiRule)
		if err != nil {
			log.Printf("fetch APIRule %s/%s failed: %v", apiRule.Namespace, apiRule.Name, err)
			return apigatewayv2.APIRule{}
		}
		return *fetchedAPIRule
	}, twoMinTimeOut, bigPollingInterval)
}
