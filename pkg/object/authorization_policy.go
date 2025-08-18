package object

import (
	istiosecurityv1beta1 "istio.io/api/security/v1beta1"
	istiotypev1beta1 "istio.io/api/type/v1beta1"
	istiopkgsecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const IstioIngressGatewaySA = "cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account"

// NewAuthorizationPolicy creates a AuthorizationPolicy object with default deny outside traffic.
func NewAuthorizationPolicy(namespace, name string, opts ...AuthorizationPolicyOption) *istiopkgsecurityv1beta1.AuthorizationPolicy {
	policy := &istiopkgsecurityv1beta1.AuthorizationPolicy{
		ObjectMeta: kmetav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: istiosecurityv1beta1.AuthorizationPolicy{
			Rules: []*istiosecurityv1beta1.Rule{
				{
					From: []*istiosecurityv1beta1.Rule_From{
						{
							Source: &istiosecurityv1beta1.Source{
								NotPrincipals: []string{
									IstioIngressGatewaySA,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, opt := range opts {
		opt(policy)
	}

	return policy
}

// WithSelector sets the selector of an AuthorizationPolicy.
func WithSelector(labels map[string]string) AuthorizationPolicyOption {
	return func(r *istiopkgsecurityv1beta1.AuthorizationPolicy) {
		r.Spec.Selector = &istiotypev1beta1.WorkloadSelector{
			MatchLabels: labels,
		}
	}
}

// WithAllowAction sets the policy action to ALLOW.
func WithAllowAction() AuthorizationPolicyOption {
	return func(r *istiopkgsecurityv1beta1.AuthorizationPolicy) {
		r.Spec.Action = istiosecurityv1beta1.AuthorizationPolicy_ALLOW
	}
}
