package virtualservice

import (
	"github.com/golang/protobuf/ptypes/duration"
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	networkingapiv1alpha3 "istio.io/api/networking/v1alpha3"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type Option func(*networkingv1alpha3.VirtualService)

func NewVirtualService(name, namespace string, opts ...Option) *networkingv1alpha3.VirtualService {
	vs := &networkingv1alpha3.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1alpha3",
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingapiv1alpha3.VirtualService{},
	}

	// apply options
	for _, opt := range opts {
		opt(vs)
	}
	return vs
}

// WithOwnerReference sets the OwnerReferences of VirtualService Object.
func WithOwnerReference(subscription eventingv1alpha2.Subscription) Option {
	return func(o *networkingv1alpha3.VirtualService) {
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
	return func(o *networkingv1alpha3.VirtualService) {
		o.Labels = labels
	}
}

func WithHost(host string) Option {
	return func(o *networkingv1alpha3.VirtualService) {
		o.Spec.Hosts = []string{
			host,
		}
	}
}

func WithGateway(gateway string) Option {
	return func(o *networkingv1alpha3.VirtualService) {
		o.Spec.Gateways = []string{
			gateway,
		}
		return
	}
}

func WithDefaultHttpRoute(host string, targetHost string, targetPort uint32) Option {
	return func(o *networkingv1alpha3.VirtualService) {
		httpRoute := &networkingapiv1alpha3.HTTPRoute{
			CorsPolicy: &networkingapiv1alpha3.CorsPolicy{
				AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
				AllowHeaders: []string{"Authorization", "Content-Type", "*"},
				AllowOrigins: []*networkingapiv1alpha3.StringMatch{{
					MatchType: &networkingapiv1alpha3.StringMatch_Regex{
						Regex: ".*",
					},
				}},
			},
			Headers: &networkingapiv1alpha3.Headers{
				Request: &networkingapiv1alpha3.Headers_HeaderOperations{
					Set: map[string]string{"x-forwarded-host": host},
				},
			},
			Match: []*networkingapiv1alpha3.HTTPMatchRequest{{
				Uri: &networkingapiv1alpha3.StringMatch{
					MatchType: &networkingapiv1alpha3.StringMatch_Regex{
						Regex: ".*", // TODO: check if this is okay.
					},
				}},
			},
			Route: []*networkingapiv1alpha3.HTTPRouteDestination{{
				Weight: 100,
				Destination: &networkingapiv1alpha3.Destination{
					Host: targetHost,
					Port: &networkingapiv1alpha3.PortSelector{
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
