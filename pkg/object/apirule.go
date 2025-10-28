package object

import (
	"net/url"
	"strings"

	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/featureflags"
)

const (
	// OAuthHandlerNameOAuth2Introspection OAuth handler name supported in Kyma for oauth2_introspection.
	OAuthHandlerNameOAuth2Introspection = "oauth2_introspection"

	// OAuthHandlerNameJWT OAuth handler name supported in Kyma for jwt.
	OAuthHandlerNameJWT = "jwt"

	// JWKSURLFormat the format of the jwks URL.
	JWKSURLFormat = `{"jwks_urls":["%s"]}`
)

// NewAPIRule creates a APIRule object.
func NewAPIRule(ns, namePrefix string, opts ...Option) *apigatewayv2.APIRule {
	rule := &apigatewayv2.APIRule{
		ObjectMeta: kmetav1.ObjectMeta{
			Namespace:    ns,
			GenerateName: namePrefix,
		},
	}

	for _, opt := range opts {
		opt(rule)
	}

	return rule
}

// ApplyExistingAPIRuleAttributes copies some important attributes from a given
// source APIRule to a destination APIRule.
func ApplyExistingAPIRuleAttributes(src, dst *apigatewayv2.APIRule) {
	// resourceVersion must be returned to the API server
	// unmodified for optimistic concurrency, as per Kubernetes API
	// conventions
	dst.Name = src.Name
	dst.GenerateName = ""
	dst.ResourceVersion = src.ResourceVersion
	dst.Spec.Hosts = src.Spec.Hosts
	// preserve status to avoid resetting conditions
	dst.Status = src.Status
}

func GetService(svcName string, port uint32) apigatewayv2.Service {
	isExternal := true
	return apigatewayv2.Service{
		Name:       &svcName,
		Port:       &port,
		IsExternal: &isExternal,
	}
}

// WithService sets the Service of an APIRule.
func WithService(hosts []*apigatewayv2.Host, svcName string, port uint32) Option {
	return func(r *apigatewayv2.APIRule) {
		apiService := GetService(svcName, port)
		r.Spec.Service = &apiService
		r.Spec.Hosts = hosts
	}
}

// WithGateway sets the gateway of an APIRule.
func WithGateway(gw string) Option {
	return func(r *apigatewayv2.APIRule) {
		r.Spec.Gateway = &gw
	}
}

// RemoveDuplicateValues appends the values if the key (values of the slice) is not equal
// to the already present value in new slice (list).
func RemoveDuplicateValues(values []string) []string {
	keys := make(map[string]bool)
	list := make([]string, 0)

	for _, entry := range values {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// WithLabels sets the labels for an APIRule.
func WithLabels(labels map[string]string) Option {
	return func(r *apigatewayv2.APIRule) {
		r.SetLabels(labels)
	}
}

// WithOwnerReference sets the OwnerReferences of an APIRule.
func WithOwnerReference(subs []eventingv1alpha2.Subscription) Option {
	return func(rule *apigatewayv2.APIRule) {
		ownerRefs := make([]kmetav1.OwnerReference, 0)
		for _, sub := range subs {
			blockOwnerDeletion := true
			ownerRef := kmetav1.OwnerReference{
				APIVersion:         sub.APIVersion,
				Kind:               sub.Kind,
				Name:               sub.Name,
				UID:                sub.UID,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}
			ownerRefs = append(ownerRefs, ownerRef)
		}
		rule.SetOwnerReferences(ownerRefs)
	}
}

// WithRules sets the rules of an APIRule for all Subscriptions for a subscriber.
func WithRules(certsURL string, subs []eventingv1alpha2.Subscription, svc apigatewayv2.Service,
	methods ...string,
) Option {
	return func(rule *apigatewayv2.APIRule) {
		rules := make([]apigatewayv2.Rule, 0)
		paths := make([]string, 0)
		for _, sub := range subs {
			hostURL, err := url.ParseRequestURI(sub.Spec.Sink)
			if err != nil {
				// It's ok as the relevant subscription will have a valid cluster local URL in the same namespace.
				continue
			}
			if hostURL.Path == "" {
				paths = append(paths, "/")
			} else {
				paths = append(paths, hostURL.Path)
			}
		}
		uniquePaths := RemoveDuplicateValues(paths)
		for _, path := range uniquePaths {
			rule := apigatewayv2.Rule{
				Path:    path,
				Methods: StringsToMethods(methods),
				Service: &svc,
			}
			if featureflags.IsEventingWebhookAuthEnabled() {
				rule.Jwt = &apigatewayv2.JwtConfig{
					Authentications: []*apigatewayv2.JwtAuthentication{
						{
							JwksUri: certsURL,
							Issuer:  strings.Replace(certsURL, "/oauth2/certs", "", 1),
						},
					},
				}
			} else {
				rule.ExtAuth = &apigatewayv2.ExtAuth{}
			}
			rules = append(rules, rule)
		}
		rule.Spec.Rules = rules
	}
}

// StringsToMethods converts a slice of strings into a slice of HttpMethod as defined by api-gateway.
func StringsToMethods(methods []string) []apigatewayv2.HttpMethod {
	httpMethodes := []apigatewayv2.HttpMethod{}
	for _, m := range methods {
		httpMethodes = append(httpMethodes, apigatewayv2.HttpMethod(m))
	}
	return httpMethodes
}
