package authorizationpolicy

import (
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	securityv1beta1 "istio.io/api/security/v1beta1"
	istiotypesv1beta1 "istio.io/api/type/v1beta1"
	clientsecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type Option func(*clientsecurityv1beta1.AuthorizationPolicy)

func NewAuthorizationPolicy(name, namespace string, opts ...Option) *clientsecurityv1beta1.AuthorizationPolicy {
	ra := &clientsecurityv1beta1.AuthorizationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.istio.io/v1beta1",
			Kind:       "AuthorizationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: securityv1beta1.AuthorizationPolicy{},
	}

	// apply options
	for _, opt := range opts {
		opt(ra)
	}
	return ra
}

// WithOwnerReference sets the OwnerReferences of VirtualService Object.
func WithOwnerReference(subscription eventingv1alpha2.Subscription) Option {
	return func(o *clientsecurityv1beta1.AuthorizationPolicy) {
		ownerReference := metav1.OwnerReference{
			APIVersion:         subscription.APIVersion,
			Kind:               subscription.Kind,
			Name:               subscription.Name,
			UID:                subscription.UID,
			BlockOwnerDeletion: pointer.Bool(true),
			Controller:         pointer.Bool(false),
		}
		// append to OwnerReferences.
		o.OwnerReferences = append(o.OwnerReferences, ownerReference)
	}
}

func WithLabels(labels map[string]string) Option {
	return func(o *clientsecurityv1beta1.AuthorizationPolicy) {
		o.Labels = labels
	}
}

func WithSelectorLabels(labels map[string]string) Option {
	return func(o *clientsecurityv1beta1.AuthorizationPolicy) {
		o.Spec.Selector = &istiotypesv1beta1.WorkloadSelector{
			MatchLabels: labels,
		}
	}
}

func WithDefaultRules() Option {
	return func(o *clientsecurityv1beta1.AuthorizationPolicy) {
		rule := &securityv1beta1.Rule{
			From: []*securityv1beta1.Rule_From{{
				Source: &securityv1beta1.Source{
					RequestPrincipals: []string{"*"},
				},
			}},
		}

		// append rule.
		o.Spec.Rules = append(o.Spec.Rules, rule)
	}
}
