package v1alpha2

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/utils"
)

const (
	DefaultMaxInFlightMessages = "10"
	minEventTypeSegments       = 2
	subdomainSegments          = 5
	InvalidPrefix              = "sap.kyma.custom"
	ClusterLocalURLSuffix      = "svc.cluster.local"
	ValidSource                = "source"
)

func (s *Subscription) ValidateSpec() field.ErrorList {
	var allErrs field.ErrorList
	if err := s.validateSubscriptionSource(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := s.validateSubscriptionTypes(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := s.validateSubscriptionConfig(); err != nil {
		allErrs = append(allErrs, err...)
	}
	if err := s.validateSubscriptionSink(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func (s *Subscription) validateSubscriptionSource() *field.Error {
	if s.Spec.Source == "" && s.Spec.TypeMatching != TypeMatchingExact {
		return MakeInvalidFieldError(SourcePath, s.Spec.Source, EmptyErrDetail)
	}
	// Check only if the source is valid for the cloud event, with a valid event type.
	if IsInvalidCE(s.Spec.Source, "") {
		return MakeInvalidFieldError(SourcePath, s.Spec.Source, InvalidURIErrDetail)
	}
	return nil
}

func (s *Subscription) validateSubscriptionTypes() *field.Error {
	if s.Spec.Types == nil || len(s.Spec.Types) == 0 {
		return MakeInvalidFieldError(TypesPath, "", EmptyErrDetail)
	}
	if duplicates := s.GetDuplicateTypes(); len(duplicates) > 0 {
		return MakeInvalidFieldError(TypesPath, strings.Join(duplicates, ","), DuplicateTypesErrDetail)
	}
	for _, etype := range s.Spec.Types {
		if len(etype) == 0 {
			return MakeInvalidFieldError(TypesPath, etype, LengthErrDetail)
		}
		if segments := strings.Split(etype, "."); len(segments) < minEventTypeSegments {
			return MakeInvalidFieldError(TypesPath, etype, MinSegmentErrDetail)
		}
		if s.Spec.TypeMatching != TypeMatchingExact && strings.HasPrefix(etype, InvalidPrefix) {
			return MakeInvalidFieldError(TypesPath, etype, InvalidPrefixErrDetail)
		}
		// Check only is the event type is valid for the cloud event, with a valid source.
		if IsInvalidCE(ValidSource, etype) {
			return MakeInvalidFieldError(TypesPath, etype, InvalidURIErrDetail)
		}
	}
	return nil
}

func (s *Subscription) validateSubscriptionConfig() field.ErrorList {
	var allErrs field.ErrorList
	if isNotInt(s.Spec.Config[MaxInFlightMessages]) {
		allErrs = append(allErrs, MakeInvalidFieldError(ConfigPath, s.Spec.Config[MaxInFlightMessages], StringIntErrDetail))
	}
	if s.ifKeyExistsInConfig(ProtocolSettingsQos) && types.IsInvalidQoS(s.Spec.Config[ProtocolSettingsQos]) {
		allErrs = append(allErrs, MakeInvalidFieldError(ConfigPath, s.Spec.Config[ProtocolSettingsQos], InvalidQosErrDetail))
	}
	if s.ifKeyExistsInConfig(WebhookAuthType) && types.IsInvalidAuthType(s.Spec.Config[WebhookAuthType]) {
		allErrs = append(allErrs, MakeInvalidFieldError(ConfigPath, s.Spec.Config[WebhookAuthType], InvalidAuthTypeErrDetail))
	}
	if s.ifKeyExistsInConfig(WebhookAuthGrantType) && types.IsInvalidGrantType(s.Spec.Config[WebhookAuthGrantType]) {
		allErrs = append(allErrs, MakeInvalidFieldError(ConfigPath, s.Spec.Config[WebhookAuthGrantType], InvalidGrantTypeErrDetail))
	}
	return allErrs
}

func (s *Subscription) validateSubscriptionSink() *field.Error {
	if s.Spec.Sink == "" {
		return MakeInvalidFieldError(SinkPath, s.Spec.Sink, EmptyErrDetail)
	}

	if !utils.IsValidScheme(s.Spec.Sink) {
		return MakeInvalidFieldError(SinkPath, s.Spec.Sink, MissingSchemeErrDetail)
	}

	trimmedHost, subDomains, err := utils.GetSinkData(s.Spec.Sink)
	if err != nil {
		return MakeInvalidFieldError(SinkPath, s.Spec.Sink, err.Error())
	}

	// Validate sink URL is a cluster local URL.
	if !strings.HasSuffix(trimmedHost, ClusterLocalURLSuffix) {
		return MakeInvalidFieldError(SinkPath, s.Spec.Sink, SuffixMissingErrDetail)
	}

	// We expected a sink in the format "service.namespace.svc.cluster.local".
	if len(subDomains) != subdomainSegments {
		return MakeInvalidFieldError(SinkPath, s.Spec.Sink, SubDomainsErrDetail+trimmedHost)
	}

	// Assumption: Subscription CR and Subscriber should be deployed in the same namespace.
	svcNs := subDomains[1]
	if s.Namespace != svcNs {
		return MakeInvalidFieldError(NSPath, s.Spec.Sink, NSMismatchErrDetail+svcNs)
	}

	return nil
}

func (s *Subscription) ifKeyExistsInConfig(key string) bool {
	_, ok := s.Spec.Config[key]
	return ok
}

func isNotInt(value string) bool {
	if _, err := strconv.Atoi(value); err != nil {
		return true
	}
	return false
}

func IsInvalidCE(source, eventType string) bool {
	if source == "" {
		return false
	}
	newEvent := utils.GetCloudEvent(eventType)
	newEvent.SetSource(source)
	err := newEvent.Validate()
	return err != nil
}
