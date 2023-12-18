package v1alpha1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/backend/eventtype"
)

const (
	ErrorHubVersionMsg     = "hub version is not the expected v1alpha2 version"
	ErrorMultipleSourceMsg = "subscription contains more than 1 eventSource"
)

var v1alpha1TypeCleaner eventtype.Cleaner //nolint:gochecknoglobals // using global var because there is no runtime
// object to hold this instance.

func InitializeEventTypeCleaner(cleaner eventtype.Cleaner) {
	v1alpha1TypeCleaner = cleaner
}

// ConvertTo converts this Subscription in version v1 to the Hub version v2.
func (s *Subscription) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*v1alpha2.Subscription)
	if !ok {
		return errors.Errorf(ErrorHubVersionMsg)
	}
	return V1ToV2(s, dst)
}

// V1ToV2 copies the v1alpha1-type field values into v1alpha2-type field values.
func V1ToV2(src *Subscription, dst *v1alpha2.Subscription) error {
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// SPEC fields

	dst.Spec.ID = src.Spec.ID
	dst.Spec.Sink = src.Spec.Sink
	dst.Spec.Source = ""

	src.setV2TypeMatching(dst)

	// protocol fields
	src.setV2ProtocolFields(dst)

	// Types
	if err := src.setV2SpecTypes(dst); err != nil {
		return err
	}

	// Config
	src.natsSpecConfigToV2(dst)

	return nil
}

// ConvertFrom converts this Subscription from the Hub version (v2) to v1.
func (s *Subscription) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*v1alpha2.Subscription)
	if !ok {
		return errors.Errorf(ErrorHubVersionMsg)
	}
	return V2ToV1(s, src)
}

// V2ToV1 copies the v1alpha2-type field values into v1alpha1-type field values.
func V2ToV1(dst *Subscription, src *v1alpha2.Subscription) error {
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.ID = src.Spec.ID
	dst.Spec.Sink = src.Spec.Sink

	dst.setV1ProtocolFields(src)

	dst.Spec.Filter = &BEBFilters{
		Filters: []*EventMeshFilter{},
	}

	for _, eventType := range src.Spec.Types {
		filter := &EventMeshFilter{
			EventSource: &Filter{
				Property: "source",
				Type:     fmt.Sprint(v1alpha2.TypeMatchingExact),
				Value:    src.Spec.Source,
			},
			EventType: &Filter{
				Type:     fmt.Sprint(v1alpha2.TypeMatchingExact),
				Property: "type",
				Value:    eventType,
			},
		}
		dst.Spec.Filter.Filters = append(dst.Spec.Filter.Filters, filter)
	}

	if src.Spec.Config != nil {
		if err := dst.natsSpecConfigToV1(src); err != nil {
			return err
		}
	}

	// Conditions
	for _, condition := range src.Status.Conditions {
		dst.Status.Conditions = append(dst.Status.Conditions, ConditionV2ToV1(condition))
	}

	dst.Status.Ready = src.Status.Ready

	dst.setV1CleanEvenTypes(src)
	dst.bebBackendStatusToV1(src)
	dst.natsBackendStatusToV1(src)

	return nil
}

// setV2TypeMatching sets the default typeMatching on the v1alpha2 Subscription version.
func (s *Subscription) setV2TypeMatching(dst *v1alpha2.Subscription) {
	dst.Spec.TypeMatching = v1alpha2.TypeMatchingExact
}

// setV2ProtocolFields converts the protocol-related fields from v1alpha1 to v1alpha2 Subscription version.
func (s *Subscription) setV2ProtocolFields(dst *v1alpha2.Subscription) {
	dst.Spec.Config = map[string]string{}
	if s.Spec.Protocol != "" {
		dst.Spec.Config[v1alpha2.Protocol] = s.Spec.Protocol
	}
	// protocol settings
	if s.Spec.ProtocolSettings != nil {
		s.setProtocolSettings(dst)
	}
}

func (s *Subscription) setProtocolSettings(dst *v1alpha2.Subscription) {
	if s.Spec.ProtocolSettings.ContentMode != nil {
		dst.Spec.Config[v1alpha2.ProtocolSettingsContentMode] = *s.Spec.ProtocolSettings.ContentMode
	}
	if s.Spec.ProtocolSettings.ExemptHandshake != nil {
		dst.Spec.Config[v1alpha2.ProtocolSettingsExemptHandshake] = strconv.FormatBool(*s.Spec.ProtocolSettings.ExemptHandshake)
	}
	if s.Spec.ProtocolSettings.Qos != nil {
		dst.Spec.Config[v1alpha2.ProtocolSettingsQos] = *s.Spec.ProtocolSettings.Qos
	}
	// webhookAuth fields
	if s.Spec.ProtocolSettings.WebhookAuth != nil {
		if s.Spec.ProtocolSettings.WebhookAuth.Type != "" {
			dst.Spec.Config[v1alpha2.WebhookAuthType] = s.Spec.ProtocolSettings.WebhookAuth.Type
		}
		dst.Spec.Config[v1alpha2.WebhookAuthGrantType] = s.Spec.ProtocolSettings.WebhookAuth.GrantType
		dst.Spec.Config[v1alpha2.WebhookAuthClientID] = s.Spec.ProtocolSettings.WebhookAuth.ClientID
		dst.Spec.Config[v1alpha2.WebhookAuthClientSecret] = s.Spec.ProtocolSettings.WebhookAuth.ClientSecret
		dst.Spec.Config[v1alpha2.WebhookAuthTokenURL] = s.Spec.ProtocolSettings.WebhookAuth.TokenURL
		if s.Spec.ProtocolSettings.WebhookAuth.Scope != nil {
			dst.Spec.Config[v1alpha2.WebhookAuthScope] = strings.Join(s.Spec.ProtocolSettings.WebhookAuth.Scope, ",")
		}
	}
}

func (s *Subscription) initializeProtocolSettingsIfNil() {
	if s.Spec.ProtocolSettings == nil {
		s.Spec.ProtocolSettings = &ProtocolSettings{}
	}
}

func (s *Subscription) initializeWebhookAuthIfNil() {
	s.initializeProtocolSettingsIfNil()
	if s.Spec.ProtocolSettings.WebhookAuth == nil {
		s.Spec.ProtocolSettings.WebhookAuth = &WebhookAuth{}
	}
}

// setV1ProtocolFields converts the protocol-related fields from v1alpha1 to v1alpha2 Subscription version.
func (s *Subscription) setV1ProtocolFields(dst *v1alpha2.Subscription) {
	if protocol, ok := dst.Spec.Config[v1alpha2.Protocol]; ok {
		s.Spec.Protocol = protocol
	}

	if currentMode, ok := dst.Spec.Config[v1alpha2.ProtocolSettingsContentMode]; ok {
		s.initializeProtocolSettingsIfNil()
		s.Spec.ProtocolSettings.ContentMode = &currentMode
	}
	if qos, ok := dst.Spec.Config[v1alpha2.ProtocolSettingsQos]; ok {
		s.initializeProtocolSettingsIfNil()
		s.Spec.ProtocolSettings.Qos = &qos
	}
	if exemptHandshake, ok := dst.Spec.Config[v1alpha2.ProtocolSettingsExemptHandshake]; ok {
		handshake, err := strconv.ParseBool(exemptHandshake)
		if err != nil {
			handshake = true
		}
		s.initializeProtocolSettingsIfNil()
		s.Spec.ProtocolSettings.ExemptHandshake = &handshake
	}

	if authType, ok := dst.Spec.Config[v1alpha2.WebhookAuthType]; ok {
		s.initializeWebhookAuthIfNil()
		s.Spec.ProtocolSettings.WebhookAuth.Type = authType
	}
	if grantType, ok := dst.Spec.Config[v1alpha2.WebhookAuthGrantType]; ok {
		s.initializeWebhookAuthIfNil()
		s.Spec.ProtocolSettings.WebhookAuth.GrantType = grantType
	}
	if clientID, ok := dst.Spec.Config[v1alpha2.WebhookAuthClientID]; ok {
		s.initializeWebhookAuthIfNil()
		s.Spec.ProtocolSettings.WebhookAuth.ClientID = clientID
	}
	if secret, ok := dst.Spec.Config[v1alpha2.WebhookAuthClientSecret]; ok {
		s.initializeWebhookAuthIfNil()
		s.Spec.ProtocolSettings.WebhookAuth.ClientSecret = secret
	}
	if token, ok := dst.Spec.Config[v1alpha2.WebhookAuthTokenURL]; ok {
		s.initializeWebhookAuthIfNil()
		s.Spec.ProtocolSettings.WebhookAuth.TokenURL = token
	}
	if scope, ok := dst.Spec.Config[v1alpha2.WebhookAuthScope]; ok {
		s.initializeWebhookAuthIfNil()
		s.Spec.ProtocolSettings.WebhookAuth.Scope = strings.Split(scope, ",")
	}
}

// setV2SpecTypes sets event types in the Subscription Spec in the v1alpha2 way.
func (s *Subscription) setV2SpecTypes(dst *v1alpha2.Subscription) error {
	if v1alpha1TypeCleaner == nil {
		return errors.New("event type cleaner is not initialized")
	}

	if s.Spec.Filter != nil {
		for _, filter := range s.Spec.Filter.Filters {
			if dst.Spec.Source == "" {
				dst.Spec.Source = filter.EventSource.Value
			}
			if dst.Spec.Source != "" && filter.EventSource.Value != dst.Spec.Source {
				return errors.New(ErrorMultipleSourceMsg)
			}
			// clean the type and merge segments if needed
			cleanedType, err := v1alpha1TypeCleaner.Clean(filter.EventType.Value)
			if err != nil {
				return err
			}

			// add the type to spec
			dst.Spec.Types = append(dst.Spec.Types, cleanedType)
		}
	}
	return nil
}

// natsSpecConfigToV2 converts the v1alpha2 Spec config to v1alpha1.
func (s *Subscription) natsSpecConfigToV1(dst *v1alpha2.Subscription) error {
	if maxInFlightMessages, ok := dst.Spec.Config[v1alpha2.MaxInFlightMessages]; ok {
		intVal, err := strconv.Atoi(maxInFlightMessages)
		if err != nil {
			return err
		}
		s.Spec.Config = &SubscriptionConfig{
			MaxInFlightMessages: intVal,
		}
	}
	return nil
}

// natsSpecConfigToV2 converts the hardcoded v1alpha1 Spec config to v1alpha2 generic config version.
func (s *Subscription) natsSpecConfigToV2(dst *v1alpha2.Subscription) {
	if s.Spec.Config != nil {
		if dst.Spec.Config == nil {
			dst.Spec.Config = map[string]string{}
		}
		dst.Spec.Config[v1alpha2.MaxInFlightMessages] = strconv.Itoa(s.Spec.Config.MaxInFlightMessages)
	}
}

// setBEBBackendStatus moves the BEB-related to Backend fields of the Status in the v1alpha2.
func (s *Subscription) bebBackendStatusToV1(dst *v1alpha2.Subscription) {
	s.Status.Ev2hash = dst.Status.Backend.Ev2hash
	s.Status.Emshash = dst.Status.Backend.EventMeshHash
	s.Status.ExternalSink = dst.Status.Backend.ExternalSink
	s.Status.FailedActivation = dst.Status.Backend.FailedActivation
	s.Status.APIRuleName = dst.Status.Backend.APIRuleName
	if dst.Status.Backend.EventMeshSubscriptionStatus != nil {
		s.Status.EmsSubscriptionStatus = &EmsSubscriptionStatus{
			SubscriptionStatus:       dst.Status.Backend.EventMeshSubscriptionStatus.Status,
			SubscriptionStatusReason: dst.Status.Backend.EventMeshSubscriptionStatus.StatusReason,
			LastSuccessfulDelivery:   dst.Status.Backend.EventMeshSubscriptionStatus.LastSuccessfulDelivery,
			LastFailedDelivery:       dst.Status.Backend.EventMeshSubscriptionStatus.LastFailedDelivery,
			LastFailedDeliveryReason: dst.Status.Backend.EventMeshSubscriptionStatus.LastFailedDeliveryReason,
		}
	}
}

// natsBackendStatusToV1 moves the NATS-related to Backend fields of the Status in the v1alpha2.
func (s *Subscription) natsBackendStatusToV1(dst *v1alpha2.Subscription) {
	if maxInFlightMessages, ok := dst.Spec.Config[v1alpha2.MaxInFlightMessages]; ok {
		intVal, err := strconv.Atoi(maxInFlightMessages)
		if err == nil {
			s.Status.Config = &SubscriptionConfig{}
			s.Status.Config.MaxInFlightMessages = intVal
		}
	}
}

// setV1CleanEvenTypes sets the clean event types to v1alpha1 Subscription Status.
func (s *Subscription) setV1CleanEvenTypes(dst *v1alpha2.Subscription) {
	s.Status.InitializeCleanEventTypes()
	for _, eventType := range dst.Status.Types {
		s.Status.CleanEventTypes = append(s.Status.CleanEventTypes, eventType.CleanType)
	}
}

// ConditionV2ToV1 converts the v1alpha2 Condition to v1alpha1 version.
func ConditionV2ToV1(condition v1alpha2.Condition) Condition {
	return Condition{
		Type:               ConditionType(condition.Type),
		Status:             condition.Status,
		LastTransitionTime: condition.LastTransitionTime,
		Reason:             ConditionReason(condition.Reason),
		Message:            condition.Message,
	}
}
