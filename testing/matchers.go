package testing

import (
	"fmt"
	"reflect"
	"strconv"

	. "github.com/onsi/gomega"         //nolint:revive,stylecheck // using . import for convenience
	. "github.com/onsi/gomega/gstruct" //nolint:revive,stylecheck // using . import for convenience
	gomegatypes "github.com/onsi/gomega/types"
	kcorev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"

	apigatewayv1beta1 "github.com/kyma-incubator/api-gateway/api/v1beta1"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/constants"
	"github.com/kyma-project/eventing-manager/pkg/object"
)

//
// APIRule matchers
//

func HaveNotEmptyAPIRule() gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv1beta1.APIRule) ktypes.UID {
		return a.UID
	}, Not(BeEmpty()))
}

func HaveNotEmptyHost() gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv1beta1.APIRule) bool {
		return a.Spec.Service != nil && a.Spec.Host != nil
	}, BeTrue())
}

func HaveAPIRuleSpecRules(ruleMethods []string, accessStrategy, certsURL, path string) gomegatypes.GomegaMatcher {
	handler := apigatewayv1beta1.Handler{
		Name: accessStrategy,
		Config: &kruntime.RawExtension{
			Raw: []byte(fmt.Sprintf(object.JWKSURLFormat, certsURL)),
		},
	}
	authenticator := &apigatewayv1beta1.Authenticator{
		Handler: &handler,
	}
	return WithTransform(func(a apigatewayv1beta1.APIRule) []apigatewayv1beta1.Rule {
		return a.Spec.Rules
	}, ContainElement(
		MatchFields(IgnoreExtras|IgnoreMissing, Fields{
			"Methods":          ConsistOf(ruleMethods),
			"AccessStrategies": ConsistOf(haveAPIRuleAccessStrategies(authenticator)),
			"Gateway":          Equal(constants.ClusterLocalAPIGateway),
			"Path":             Equal(path),
		}),
	))
}

func haveAPIRuleAccessStrategies(authenticator *apigatewayv1beta1.Authenticator) gomegatypes.GomegaMatcher {
	return WithTransform(func(a *apigatewayv1beta1.Authenticator) *apigatewayv1beta1.Authenticator {
		return a
	}, Equal(authenticator))
}

func HaveAPIRuleSpecRulesWithOry(ruleMethods []string, accessStrategy, path string) gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv1beta1.APIRule) []apigatewayv1beta1.Rule {
		return a.Spec.Rules
	}, ContainElement(
		MatchFields(IgnoreExtras|IgnoreMissing, Fields{
			"Methods":          ConsistOf(ruleMethods),
			"AccessStrategies": ConsistOf(haveAPIRuleAccessStrategiesWithOry(accessStrategy)),
			"Gateway":          Equal(constants.ClusterLocalAPIGateway),
			"Path":             Equal(path),
		}),
	))
}

func haveAPIRuleAccessStrategiesWithOry(accessStrategy string) gomegatypes.GomegaMatcher {
	return WithTransform(func(a *apigatewayv1beta1.Authenticator) string {
		return a.Name
	}, Equal(accessStrategy))
}

func HaveAPIRuleOwnersRefs(uids ...ktypes.UID) gomegatypes.GomegaMatcher {
	return WithTransform(func(a apigatewayv1beta1.APIRule) []ktypes.UID {
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
