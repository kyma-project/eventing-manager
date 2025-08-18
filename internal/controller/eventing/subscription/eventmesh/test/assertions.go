package test

import (
	"context"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"log"

	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	"github.com/onsi/gomega"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
)

// getSubscriptionAssert fetches a subscription using the lookupKey and allows making assertions on it.
func getSubscriptionAssert(ctx context.Context, g *gomega.GomegaWithT,
	subscription *eventingv1alpha2.Subscription,
) gomega.AsyncAssertion {
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

// getAPIRuleForASvcAssert fetches an apiRule for a given service and allows making assertions on it.
func getAPIRuleForASvcAssert(ctx context.Context, g *gomega.GomegaWithT, svc *kcorev1.Service) gomega.AsyncAssertion {
	return g.Eventually(func() apigatewayv2.APIRule {
		apiRules, err := getAPIRulesList(ctx, svc)
		g.Expect(err).ShouldNot(gomega.HaveOccurred())
		return filterAPIRulesForASvc(apiRules, svc)
	}, smallTimeOut, smallPollingInterval)
}

// getAPIRuleAssert fetches an apiRule and allows making assertions on it.
func getAPIRuleAssert(ctx context.Context, g *gomega.GomegaWithT,
	apiRule *apigatewayv2.APIRule,
) gomega.AsyncAssertion {
	return g.Eventually(func() apigatewayv2.APIRule {
		fetchedAPIRule, err := getAPIRule(ctx, apiRule)
		if err != nil {
			log.Printf("fetch APIRule %s/%s failed: %v", apiRule.Namespace, apiRule.Name, err)
			return apigatewayv2.APIRule{}
		}
		return *fetchedAPIRule
	}, twoMinTimeOut, bigPollingInterval)
}

func getAuthorizationPolicyAssert(ctx context.Context, g *gomega.GomegaWithT,
	authPolicy *istiosecurityv1beta1.AuthorizationPolicy,
) gomega.AsyncAssertion {
	return g.Eventually(func() *istiosecurityv1beta1.AuthorizationPolicy {
		fetchedPolicy, err := getAuthorizationPolicy(ctx, authPolicy)
		if err != nil {
			log.Printf("fetch authPolicy %s/%s failed: %v", authPolicy.Namespace, authPolicy.Name, err)
			return &istiosecurityv1beta1.AuthorizationPolicy{}
		}
		return fetchedPolicy
	}, twoMinTimeOut, bigPollingInterval)
}
