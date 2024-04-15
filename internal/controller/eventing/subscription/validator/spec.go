package validator

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/utils"
)

const (
	minEventTypeSegments  = 2
	subdomainSegments     = 5
	validPrefix           = "sap.kyma.custom"
	clusterLocalURLSuffix = "svc.cluster.local"
)

func validateSpec(subscription eventingv1alpha2.Subscription) field.ErrorList {
	var allErrs field.ErrorList
	if err := validateSource(subscription); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateTypes(subscription); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateConfig(subscription); err != nil {
		allErrs = append(allErrs, err...)
	}
	if err := validateSink(subscription); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func validateSource(subscription eventingv1alpha2.Subscription) *field.Error {
	if subscription.Spec.Source == "" && subscription.Spec.TypeMatching != eventingv1alpha2.TypeMatchingExact {
		return makeInvalidFieldError(sourcePath, subscription.Spec.Source, emptyErrDetail)
	}
	// Check only if the source is valid for the cloud event, with a valid event type.
	if isInvalidCE(subscription.Spec.Source, "") {
		return makeInvalidFieldError(sourcePath, subscription.Spec.Source, invalidURIErrDetail)
	}
	return nil
}

func validateTypes(subscription eventingv1alpha2.Subscription) *field.Error {
	if subscription.Spec.Types == nil || len(subscription.Spec.Types) == 0 {
		return makeInvalidFieldError(typesPath, "", emptyErrDetail)
	}
	if duplicates := subscription.GetDuplicateTypes(); len(duplicates) > 0 {
		return makeInvalidFieldError(typesPath, strings.Join(duplicates, ","), duplicateTypesErrDetail)
	}
	for _, eventType := range subscription.Spec.Types {
		if len(eventType) == 0 {
			return makeInvalidFieldError(typesPath, eventType, lengthErrDetail)
		}
		if segments := strings.Split(eventType, "."); len(segments) < minEventTypeSegments {
			return makeInvalidFieldError(typesPath, eventType, minSegmentErrDetail)
		}
		if subscription.Spec.TypeMatching != eventingv1alpha2.TypeMatchingExact && strings.HasPrefix(eventType, validPrefix) {
			return makeInvalidFieldError(typesPath, eventType, invalidPrefixErrDetail)
		}
		// Check only is the event type is valid for the cloud event, with a valid source.
		const validSource = "source"
		if isInvalidCE(validSource, eventType) {
			return makeInvalidFieldError(typesPath, eventType, invalidURIErrDetail)
		}
	}
	return nil
}

func validateConfig(subscription eventingv1alpha2.Subscription) field.ErrorList {
	var allErrs field.ErrorList
	if isNotInt(subscription.Spec.Config[eventingv1alpha2.MaxInFlightMessages]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, subscription.Spec.Config[eventingv1alpha2.MaxInFlightMessages], stringIntErrDetail))
	}
	if ifKeyExistsInConfig(subscription, eventingv1alpha2.ProtocolSettingsQos) && types.IsInvalidQoS(subscription.Spec.Config[eventingv1alpha2.ProtocolSettingsQos]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, subscription.Spec.Config[eventingv1alpha2.ProtocolSettingsQos], invalidQosErrDetail))
	}
	if ifKeyExistsInConfig(subscription, eventingv1alpha2.WebhookAuthType) && types.IsInvalidAuthType(subscription.Spec.Config[eventingv1alpha2.WebhookAuthType]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, subscription.Spec.Config[eventingv1alpha2.WebhookAuthType], invalidAuthTypeErrDetail))
	}
	if ifKeyExistsInConfig(subscription, eventingv1alpha2.WebhookAuthGrantType) && types.IsInvalidGrantType(subscription.Spec.Config[eventingv1alpha2.WebhookAuthGrantType]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, subscription.Spec.Config[eventingv1alpha2.WebhookAuthGrantType], invalidGrantTypeErrDetail))
	}
	return allErrs
}

func validateSink(subscription eventingv1alpha2.Subscription) *field.Error {
	if subscription.Spec.Sink == "" {
		return makeInvalidFieldError(sinkPath, subscription.Spec.Sink, emptyErrDetail)
	}

	if !utils.IsValidScheme(subscription.Spec.Sink) {
		return makeInvalidFieldError(sinkPath, subscription.Spec.Sink, missingSchemeErrDetail)
	}

	trimmedHost, subDomains, err := utils.GetSinkData(subscription.Spec.Sink)
	if err != nil {
		return makeInvalidFieldError(sinkPath, subscription.Spec.Sink, err.Error())
	}

	// Validate sink URL is a cluster local URL.
	if !strings.HasSuffix(trimmedHost, clusterLocalURLSuffix) {
		return makeInvalidFieldError(sinkPath, subscription.Spec.Sink, suffixMissingErrDetail)
	}

	// We expected a sink in the format "service.namespace.svc.cluster.local".
	if len(subDomains) != subdomainSegments {
		return makeInvalidFieldError(sinkPath, subscription.Spec.Sink, subDomainsErrDetail+trimmedHost)
	}

	// Assumption: Subscription CR and Subscriber should be deployed in the same namespace.
	svcNs := subDomains[1]
	if subscription.Namespace != svcNs {
		return makeInvalidFieldError(namespacePath, subscription.Spec.Sink, namespaceMismatchErrDetail+svcNs)
	}

	return nil
}

func ifKeyExistsInConfig(subscription eventingv1alpha2.Subscription, key string) bool {
	_, ok := subscription.Spec.Config[key]
	return ok
}

func isNotInt(value string) bool {
	if _, err := strconv.Atoi(value); err != nil {
		return true
	}
	return false
}

func isInvalidCE(source, eventType string) bool {
	if source == "" {
		return false
	}
	newEvent := utils.GetCloudEvent(eventType)
	newEvent.SetSource(source)
	err := newEvent.Validate()
	return err != nil
}
