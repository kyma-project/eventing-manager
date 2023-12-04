package object

import (
	"fmt"
	"net/url"

	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apigateway "github.com/kyma-incubator/api-gateway/api/v1beta1"

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
func NewAPIRule(ns, namePrefix string, opts ...Option) *apigateway.APIRule {
	s := &apigateway.APIRule{
		ObjectMeta: kmeta.ObjectMeta{
			Namespace:    ns,
			GenerateName: namePrefix,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// ApplyExistingAPIRuleAttributes copies some important attributes from a given
// source APIRule to a destination APIRule.
func ApplyExistingAPIRuleAttributes(src, dst *apigateway.APIRule) {
	// resourceVersion must be returned to the API server
	// unmodified for optimistic concurrency, as per Kubernetes API
	// conventions
	dst.Name = src.Name
	dst.GenerateName = ""
	dst.ResourceVersion = src.ResourceVersion
	dst.Spec.Host = src.Spec.Host
	// preserve status to avoid resetting conditions
	dst.Status = src.Status
}

func GetService(svcName string, port uint32) apigateway.Service {
	isExternal := true
	return apigateway.Service{
		Name:       &svcName,
		Port:       &port,
		IsExternal: &isExternal,
	}
}

// WithService sets the Service of an APIRule.
func WithService(host, svcName string, port uint32) Option {
	return func(r *apigateway.APIRule) {
		apiService := GetService(svcName, port)
		r.Spec.Service = &apiService
		r.Spec.Host = &host
	}
}

// WithGateway sets the gateway of an APIRule.
func WithGateway(gw string) Option {
	return func(r *apigateway.APIRule) {
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
	return func(r *apigateway.APIRule) {
		r.SetLabels(labels)
	}
}

// WithOwnerReference sets the OwnerReferences of an APIRule.
func WithOwnerReference(subs []eventingv1alpha2.Subscription) Option {
	return func(r *apigateway.APIRule) {
		ownerRefs := make([]kmeta.OwnerReference, 0)
		for _, sub := range subs {
			blockOwnerDeletion := true
			ownerRef := kmeta.OwnerReference{
				APIVersion:         sub.APIVersion,
				Kind:               sub.Kind,
				Name:               sub.Name,
				UID:                sub.UID,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}
			ownerRefs = append(ownerRefs, ownerRef)
		}
		r.SetOwnerReferences(ownerRefs)
	}
}

// WithRules sets the rules of an APIRule for all Subscriptions for a subscriber.
func WithRules(certsURL string, subs []eventingv1alpha2.Subscription, svc apigateway.Service,
	methods ...string) Option {
	return func(r *apigateway.APIRule) {
		var handler apigateway.Handler
		if featureflags.IsEventingWebhookAuthEnabled() {
			handler.Name = OAuthHandlerNameJWT
			handler.Config = &runtime.RawExtension{
				Raw: []byte(fmt.Sprintf(JWKSURLFormat, certsURL)),
			}
		} else {
			handler.Name = OAuthHandlerNameOAuth2Introspection
		}
		authenticator := &apigateway.Authenticator{
			Handler: &handler,
		}
		accessStrategies := []*apigateway.Authenticator{
			authenticator,
		}
		rules := make([]apigateway.Rule, 0)
		paths := make([]string, 0)
		for _, sub := range subs {
			hostURL, err := url.ParseRequestURI(sub.Spec.Sink)
			if err != nil {
				// It's ok as the relevant subscription will have a valid cluster local URL in the same namespace
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
			rule := apigateway.Rule{
				Path:             path,
				Methods:          methods,
				AccessStrategies: accessStrategies,
				Service:          &svc,
			}
			rules = append(rules, rule)
		}
		r.Spec.Rules = rules
	}
}
