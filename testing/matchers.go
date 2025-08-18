package testing

import (
	"maps"
	"reflect"
	"strconv"

	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	gomegatypes "github.com/onsi/gomega/types"
	istiopkgsecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	kcorev1 "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/constants"
	"github.com/kyma-project/eventing-manager/pkg/object"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct" //nolint:stylecheck // using '.' import for convenience
)

//
// APIRule matchers
//

func HaveNotEmptyAPIRule() gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv2.APIRule) ktypes.UID {
		return a.UID
	}, Not(BeEmpty()))
}

func HaveEmptyAPIRule() gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv2.APIRule) ktypes.UID {
		return a.UID
	}, BeEmpty())
}

func HaveNotEmptyHost() gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv2.APIRule) bool {
		return a.Spec.Service != nil && a.Spec.Hosts != nil
	}, BeTrue())
}

func HaveAPIRuleSpecRules(ruleMethods []string, certsURL, path, issuerURL string) gomegatypes.GomegaMatcher {
	jwt := apigatewayv2.JwtConfig{
		Authentications: []*apigatewayv2.JwtAuthentication{
			{
				JwksUri: certsURL,
				Issuer:  issuerURL,
			},
		},
	}
	return WithTransform(func(a apigatewayv2.APIRule) []apigatewayv2.Rule {
		return a.Spec.Rules
	}, ContainElement(
		MatchFields(IgnoreExtras|IgnoreMissing, Fields{
			"Methods": ConsistOf(object.StringsToMethods(ruleMethods)),
			"Jwt":     haveAPIRuleAccessStrategies(&jwt),
			"Gateway": Equal(constants.ClusterLocalAPIGateway),
			"Path":    Equal(path),
		}),
	))
}

func haveAPIRuleAccessStrategies(authenticator *apigatewayv2.JwtConfig) gomegatypes.GomegaMatcher {
	return WithTransform(func(a *apigatewayv2.JwtConfig) *apigatewayv2.JwtConfig {
		return a
	}, Equal(authenticator))
}

func HaveAPIRuleOwnersRefs(uids ...ktypes.UID) gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv2.APIRule) []ktypes.UID {
		ownerRefUIDs := make([]ktypes.UID, 0, len(a.OwnerReferences))
		for _, ownerRef := range a.OwnerReferences {
			ownerRefUIDs = append(ownerRefUIDs, ownerRef.UID)
		}
		return ownerRefUIDs
	}, Equal(uids))
}

//
// int matchers
//

func HaveValidClientID(clientIDKey, clientID string) gomegatypes.GomegaMatcher {
	return WithTransform(func(secret *kcorev1.Secret) bool {
		if secret != nil {
			return string(secret.Data[clientIDKey]) == clientID
		}
		return false
	}, BeTrue())
}

func HaveValidClientSecret(clientSecretKey, clientSecret string) gomegatypes.GomegaMatcher {
	return WithTransform(func(secret *kcorev1.Secret) bool {
		if secret != nil {
			return string(secret.Data[clientSecretKey]) == clientSecret
		}
		return false
	}, BeTrue())
}

func HaveValidTokenEndpoint(tokenEndpointKey, tokenEndpoint string) gomegatypes.GomegaMatcher {
	return WithTransform(func(secret *kcorev1.Secret) bool {
		if secret != nil {
			return string(secret.Data[tokenEndpointKey]) == tokenEndpoint
		}
		return false
	}, BeTrue())
}

func HaveValidEMSPublishURL(emsPublishURLKey, emsPublishURL string) gomegatypes.GomegaMatcher {
	return WithTransform(func(secret *kcorev1.Secret) bool {
		if secret != nil {
			return string(secret.Data[emsPublishURLKey]) == emsPublishURL
		}
		return false
	}, BeTrue())
}

func HaveValidBEBNamespace(bebNamespaceKey, namespace string) gomegatypes.GomegaMatcher {
	return WithTransform(func(secret *kcorev1.Secret) bool {
		if secret != nil {
			return string(secret.Data[bebNamespaceKey]) == namespace
		}
		return false
	}, BeTrue())
}

func HaveSubscriptionName(name string) gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) string { return s.Name }, Equal(name))
}

func HaveSubscriptionFinalizer(finalizer string) gomegatypes.GomegaMatcher {
	return WithTransform(
		func(s *eventingv1alpha2.Subscription) []string {
			return s.ObjectMeta.Finalizers
		}, ContainElement(finalizer))
}

func IsAnEmptySubscription() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) bool {
		emptySub := eventingv1alpha2.Subscription{}
		return reflect.DeepEqual(*s, emptySub)
	}, BeTrue())
}

func HaveNoneEmptyAPIRuleName() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) string {
		return s.Status.Backend.APIRuleName
	}, Not(BeEmpty()))
}

func HaveAPIRuleName(name string) gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) bool {
		return s.Status.Backend.APIRuleName == name
	}, BeTrue())
}

func HaveSubscriptionReady() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) bool {
		return s.Status.Ready
	}, BeTrue())
}

func HaveTypes(types []string) gomegatypes.GomegaMatcher {
	return WithTransform(
		func(s *eventingv1alpha2.Subscription) []string {
			return s.Spec.Types
		},
		Equal(types))
}

func HaveMaxInFlight(maxInFlight int) gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) bool {
		return s.Spec.Config[eventingv1alpha2.MaxInFlightMessages] == strconv.Itoa(maxInFlight)
	}, BeTrue())
}

func HaveTypeMatching(typeMatching eventingv1alpha2.TypeMatching) gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) bool {
		return s.Spec.TypeMatching == typeMatching
	}, BeTrue())
}

func HaveSubscriptionNotReady() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) bool {
		return s.Status.Ready
	}, BeFalse())
}

func HaveCondition(condition eventingv1alpha2.Condition) gomegatypes.GomegaMatcher {
	return WithTransform(
		func(s *eventingv1alpha2.Subscription) []eventingv1alpha2.Condition {
			return s.Status.Conditions
		},
		ContainElement(MatchFields(IgnoreExtras|IgnoreMissing, Fields{
			"Type":    Equal(condition.Type),
			"Reason":  Equal(condition.Reason),
			"Message": Equal(condition.Message),
			"Status":  Equal(condition.Status),
		})))
}

func HaveSubscriptionActiveCondition() gomegatypes.GomegaMatcher {
	return HaveCondition(eventingv1alpha2.MakeCondition(
		eventingv1alpha2.ConditionSubscriptionActive,
		eventingv1alpha2.ConditionReasonSubscriptionActive,
		kcorev1.ConditionTrue, ""))
}

func HaveAPIRuleTrueStatusCondition() gomegatypes.GomegaMatcher {
	return HaveCondition(eventingv1alpha2.MakeCondition(
		eventingv1alpha2.ConditionAPIRuleStatus,
		eventingv1alpha2.ConditionReasonAPIRuleStatusReady,
		kcorev1.ConditionTrue,
		"",
	))
}

func HaveCleanEventTypes(cleanEventTypes []eventingv1alpha2.EventType) gomegatypes.GomegaMatcher {
	return WithTransform(
		func(s *eventingv1alpha2.Subscription) []eventingv1alpha2.EventType {
			return s.Status.Types
		},
		Equal(cleanEventTypes))
}

func DefaultReadyCondition() eventingv1alpha2.Condition {
	return eventingv1alpha2.MakeCondition(
		eventingv1alpha2.ConditionSubscriptionActive,
		eventingv1alpha2.ConditionReasonNATSSubscriptionActive,
		kcorev1.ConditionTrue, "")
}

func HaveStatusTypes(cleanEventTypes []eventingv1alpha2.EventType) gomegatypes.GomegaMatcher {
	return WithTransform(
		func(s *eventingv1alpha2.Subscription) []eventingv1alpha2.EventType {
			return s.Status.Types
		},
		Equal(cleanEventTypes))
}

func HaveNotFoundSubscription() gomegatypes.GomegaMatcher {
	return WithTransform(func(isDeleted bool) bool { return isDeleted }, BeTrue())
}

func HaveEvent(event kcorev1.Event) gomegatypes.GomegaMatcher {
	return WithTransform(
		func(l kcorev1.EventList) []kcorev1.Event {
			return l.Items
		}, ContainElement(MatchFields(IgnoreExtras|IgnoreMissing, Fields{
			"Reason":  Equal(event.Reason),
			"Message": Equal(event.Message),
			"Type":    Equal(event.Type),
		})))
}

func HaveNonZeroEv2Hash() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) int64 {
		return s.Status.Backend.Ev2hash
	}, Not(BeZero()))
}

func HaveNonZeroEventMeshHash() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) int64 {
		return s.Status.Backend.EventMeshHash
	}, Not(BeZero()))
}

func HaveNonZeroEventMeshLocalHash() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) int64 {
		return s.Status.Backend.EventMeshLocalHash
	}, Not(BeZero()))
}

func HaveNonZeroWebhookAuthHash() gomegatypes.GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha2.Subscription) int64 {
		return s.Status.Backend.WebhookAuthHash
	}, Not(BeZero()))
}

func HaveMatchingSelector(selector kcorev1.Service) gomegatypes.GomegaMatcher {
	return WithTransform(func(a *istiopkgsecurityv1beta1.AuthorizationPolicy) bool {
		return maps.Equal(a.Spec.GetSelector().GetMatchLabels(), selector.Spec.Selector)
	}, BeTrue())
}
