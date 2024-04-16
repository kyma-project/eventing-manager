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
	if err := validateTypeMatching(subscription.Spec.TypeMatching); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateSource(subscription.Spec.Source, subscription.Spec.TypeMatching); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateTypes(subscription.Spec.Types, subscription.Spec.TypeMatching); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateConfig(subscription.Spec.Config); err != nil {
		allErrs = append(allErrs, err...)
	}
	if err := validateSink(subscription.Spec.Sink, subscription.Namespace); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func validateTypeMatching(typeMatching eventingv1alpha2.TypeMatching) *field.Error {
	if typeMatching != eventingv1alpha2.TypeMatchingExact && typeMatching != eventingv1alpha2.TypeMatchingStandard {
		return makeInvalidFieldError(typeMatchingPath, string(typeMatching), invalidTypeMatchingErrDetail)
	}
	return nil
}

func validateSource(source string, typeMatching eventingv1alpha2.TypeMatching) *field.Error {
	if source == "" && typeMatching != eventingv1alpha2.TypeMatchingExact {
		return makeInvalidFieldError(sourcePath, source, emptyErrDetail)
	}
	// Check only if the source is valid for the cloud event, with a valid event type.
	if isInvalidCE(source, "") {
		return makeInvalidFieldError(sourcePath, source, invalidURIErrDetail)
	}
	return nil
}

func validateTypes(types []string, typeMatching eventingv1alpha2.TypeMatching) *field.Error {
	if len(types) == 0 {
		return makeInvalidFieldError(typesPath, "", emptyErrDetail)
	}
	if duplicates := getDuplicates(types); len(duplicates) > 0 {
		return makeInvalidFieldError(typesPath, strings.Join(duplicates, ","), duplicateTypesErrDetail)
	}
	for _, eventType := range types {
		if len(eventType) == 0 {
			return makeInvalidFieldError(typesPath, eventType, lengthErrDetail)
		}
		if segments := strings.Split(eventType, "."); len(segments) < minEventTypeSegments {
			return makeInvalidFieldError(typesPath, eventType, minSegmentErrDetail)
		}
		if typeMatching != eventingv1alpha2.TypeMatchingExact && strings.HasPrefix(eventType, validPrefix) {
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

func validateConfig(config map[string]string) field.ErrorList {
	var allErrs field.ErrorList
	if isNotInt(config[eventingv1alpha2.MaxInFlightMessages]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, config[eventingv1alpha2.MaxInFlightMessages], stringIntErrDetail))
	}
	if ifKeyExistsInConfig(config, eventingv1alpha2.ProtocolSettingsQos) && types.IsInvalidQoS(config[eventingv1alpha2.ProtocolSettingsQos]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, config[eventingv1alpha2.ProtocolSettingsQos], invalidQosErrDetail))
	}
	if ifKeyExistsInConfig(config, eventingv1alpha2.WebhookAuthType) && types.IsInvalidAuthType(config[eventingv1alpha2.WebhookAuthType]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, config[eventingv1alpha2.WebhookAuthType], invalidAuthTypeErrDetail))
	}
	if ifKeyExistsInConfig(config, eventingv1alpha2.WebhookAuthGrantType) && types.IsInvalidGrantType(config[eventingv1alpha2.WebhookAuthGrantType]) {
		allErrs = append(allErrs, makeInvalidFieldError(configPath, config[eventingv1alpha2.WebhookAuthGrantType], invalidGrantTypeErrDetail))
	}
	return allErrs
}

func validateSink(sink, namespace string) *field.Error {
	if sink == "" {
		return makeInvalidFieldError(sinkPath, sink, emptyErrDetail)
	}

	if !utils.IsValidScheme(sink) {
		return makeInvalidFieldError(sinkPath, sink, missingSchemeErrDetail)
	}

	trimmedHost, subDomains, err := utils.GetSinkData(sink)
	if err != nil {
		return makeInvalidFieldError(sinkPath, sink, err.Error())
	}

	// Validate sink URL is a cluster local URL.
	if !strings.HasSuffix(trimmedHost, clusterLocalURLSuffix) {
		return makeInvalidFieldError(sinkPath, sink, suffixMissingErrDetail)
	}

	// We expected a sink in the format "service.namespace.svc.cluster.local".
	if len(subDomains) != subdomainSegments {
		return makeInvalidFieldError(sinkPath, sink, subDomainsErrDetail+trimmedHost)
	}

	// Assumption: Subscription CR and Subscriber should be deployed in the same namespace.
	svcNs := subDomains[1]
	if namespace != svcNs {
		return makeInvalidFieldError(namespacePath, sink, namespaceMismatchErrDetail+svcNs)
	}

	return nil
}

func ifKeyExistsInConfig(config map[string]string, key string) bool {
	_, ok := config[key]
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

// getDuplicates returns the duplicate items from the given list.
func getDuplicates(items []string) []string {
	if len(items) == 0 {
		return items
	}
	const duplicatesCount = 2
	itemsMap := make(map[string]int, len(items))
	duplicates := make([]string, 0, len(items))
	for _, t := range items {
		if itemsMap[t]++; itemsMap[t] == duplicatesCount {
			duplicates = append(duplicates, t)
		}
	}
	return duplicates
}
