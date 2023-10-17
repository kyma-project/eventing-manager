package requestauthentication

import (
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	securityv1beta1 "istio.io/api/security/v1beta1"
	clientsecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type Option func(*clientsecurityv1beta1.RequestAuthentication)

func NewRequestAuthentication(name, namespace string, opts ...Option) *clientsecurityv1beta1.RequestAuthentication {
	ra := &clientsecurityv1beta1.RequestAuthentication{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.istio.io/v1beta1",
			Kind:       "RequestAuthentication",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: securityv1beta1.RequestAuthentication{},
	}

	// apply options
	for _, opt := range opts {
		opt(ra)
	}
	return ra
}

// WithOwnerReference sets the OwnerReferences of VirtualService Object.
func WithOwnerReference(subscription eventingv1alpha2.Subscription) Option {
	return func(o *clientsecurityv1beta1.RequestAuthentication) {
		ownerReference := metav1.OwnerReference{
			APIVersion:         subscription.APIVersion,
			Kind:               subscription.Kind,
			Name:               subscription.Name,
			UID:                subscription.UID,
			BlockOwnerDeletion: pointer.Bool(false),
			Controller:         pointer.Bool(false),
		}
		// append to OwnerReferences.
		o.OwnerReferences = append(o.OwnerReferences, ownerReference)
	}
}

func WithLabels(labels map[string]string) Option {
	return func(o *clientsecurityv1beta1.RequestAuthentication) {
		o.Labels = labels
	}
}

func WithSelectorLabels(labels map[string]string) Option {
	return func(o *clientsecurityv1beta1.RequestAuthentication) {
		o.Spec.Selector.MatchLabels = labels
	}
}

func WithDefaultRules(issuer, jwkURI string) Option {
	return func(o *clientsecurityv1beta1.RequestAuthentication) {
		rule := &securityv1beta1.JWTRule{
			Issuer:  issuer,
			JwksUri: jwkURI,
		}

		// append rule.
		o.Spec.JwtRules = append(o.Spec.JwtRules, rule)
	}
}
