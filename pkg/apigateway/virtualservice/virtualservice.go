package virtualservice

import (
	"github.com/golang/protobuf/ptypes/duration"
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	networkingapiv1beta1 "istio.io/api/networking/v1beta1"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type Option func(*networkingv1beta1.VirtualService)

func NewVirtualService(name, namespace string, opts ...Option) *networkingv1beta1.VirtualService {
	vs := &networkingv1beta1.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1alpha3",
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingapiv1beta1.VirtualService{},
	}

	// apply options
	for _, opt := range opts {
		opt(vs)
	}
	return vs
}

// WithOwnerReference sets the OwnerReferences of VirtualService Object.
func WithOwnerReference(subscription eventingv1alpha2.Subscription) Option {
	return func(o *networkingv1beta1.VirtualService) {
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
	return func(o *networkingv1beta1.VirtualService) {
		o.Labels = labels
	}
}

func WithHost(host string) Option {
	return func(o *networkingv1beta1.VirtualService) {
		o.Spec.Hosts = []string{
			host,
		}
	}
}

func WithGateway(gateway string) Option {
	return func(o *networkingv1beta1.VirtualService) {
		o.Spec.Gateways = []string{
			gateway,
		}
		return
	}
}

func WithDefaultHttpRoute(host string, targetHost string, targetPort uint32) Option {
	return func(o *networkingv1beta1.VirtualService) {
		httpRoute := &networkingapiv1beta1.HTTPRoute{
			CorsPolicy: &networkingapiv1beta1.CorsPolicy{
				AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
				AllowHeaders: []string{"Authorization", "Content-Type", "*"},
				AllowOrigins: []*networkingapiv1beta1.StringMatch{{
					MatchType: &networkingapiv1beta1.StringMatch_Regex{
						Regex: ".*",
					},
				}},
			},
			Headers: &networkingapiv1beta1.Headers{
				Request: &networkingapiv1beta1.Headers_HeaderOperations{
					Set: map[string]string{"x-forwarded-host": host},
				},
			},
			Match: []*networkingapiv1beta1.HTTPMatchRequest{{
				Uri: &networkingapiv1beta1.StringMatch{
					MatchType: &networkingapiv1beta1.StringMatch_Regex{
						Regex: ".*", // TODO: check if this is okay.
					},
				}},
			},
			Route: []*networkingapiv1beta1.HTTPRouteDestination{{
				Weight: 100,
				Destination: &networkingapiv1beta1.Destination{
					Host: targetHost,
					Port: &networkingapiv1beta1.PortSelector{
						Number: targetPort,
					},
				},
			}},
			Timeout: &duration.Duration{
				Seconds: 180,
			},
		}
		// add the Virtual Service object.
		o.Spec.Http = append(o.Spec.Http, httpRoute)
	}
}
