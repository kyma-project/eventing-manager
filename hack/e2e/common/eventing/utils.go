package eventing

import (
	"fmt"
	"net/http"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/google/uuid"
)

const (
	keyApp  = "app"
	keyMode = "mode"
	keyType = "type"
)

func Is2XX(statusCode int) bool {
	return http.StatusOK <= statusCode && statusCode <= http.StatusIMUsed
}

func LegacyEventData(source, eventType string) string {
	return `{\"` + keyApp + `\":\"` + source + `\",\"` + keyMode + `\":\"legacy\",\"` + keyType + `\":\"` + eventType + `\"}`
}

func LegacyEventPayload(id, eventVersion, eventType, data string) string {
	return `{"data":"` + data + `","event-id":"` + id + `","event-type":"` + eventType + `","event-time":"2020-04-02T21:37:00Z","event-type-version":"` + eventVersion + `"}`
}

func CloudEventMode(encoding binding.Encoding) string {
	return fmt.Sprintf("ce-%s", encoding.String())
}

func CloudEventData(source, eventType string, encoding binding.Encoding) map[string]interface{} {
	return map[string]interface{}{keyApp: source, keyMode: CloudEventMode(encoding), keyType: eventType}
}

func ExtractSourceFromSubscriptionV1Alpha1Type(eventType string) string {
	segments := strings.Split(eventType, ".")
	return segments[3]
}

func ExtractLegacyTypeFromSubscriptionV1Alpha1Type(eventTypePrefix, eventSource, eventVersion, eventType string) string {
	tmp := strings.TrimPrefix(eventType, fmt.Sprintf("%s.%s.", eventTypePrefix, eventSource))
	return strings.TrimSuffix(tmp, fmt.Sprintf(".%s", eventVersion))
}

func ExtractLegacyTypeFromSubscriptionV1Alpha2Type(eventVersion, eventType string) string {
	return strings.TrimSuffix(eventType, fmt.Sprintf(".%s", eventVersion))
}

func ExtractVersionFromEventType(eventType string) string {
	segments := strings.Split(eventType, ".")
	return segments[len(segments)-1]
}

func NewLegacyEventForV1Alpha1(eventType, eventTypePrefix string) (string, string, string, string) {
	// If the eventType is sap.kyma.custom.noapp.order.created.v1, then for legacy event:
	// eventSource should be: noapp
	// eventType should be: order.created
	// eventVersion should be: v1
	eventID := uuid.New().String()
	eventSource := ExtractSourceFromSubscriptionV1Alpha1Type(eventType)
	eventVersion := ExtractVersionFromEventType(eventType)
	legacyEventType := ExtractLegacyTypeFromSubscriptionV1Alpha1Type(eventTypePrefix,
		eventSource, eventVersion, eventType)
	eventData := LegacyEventData(eventSource, legacyEventType)
	payload := LegacyEventPayload(eventID, eventVersion, legacyEventType, eventData)

	return eventID, eventSource, legacyEventType, payload
}

func NewLegacyEvent(eventSource, eventType string) (string, string, string, string) {
	// If the eventType is order.created.v1 and source is noapp, then for legacy event:
	// eventSource should be: noapp
	// eventType should be: order.created
	// eventVersion should be: v1
	eventID := uuid.New().String()
	eventVersion := ExtractVersionFromEventType(eventType)
	legacyEventType := ExtractLegacyTypeFromSubscriptionV1Alpha2Type(eventVersion, eventType)
	eventData := LegacyEventData(eventSource, legacyEventType)
	payload := LegacyEventPayload(eventID, eventVersion, legacyEventType, eventData)

	return eventID, eventSource, legacyEventType, payload
}

func NewCloudEvent(eventSource, eventType string, encoding binding.Encoding) (*cloudevents.Event, error) {
	eventID := uuid.New().String()
	ce := cloudevents.NewEvent()
	data := CloudEventData(eventSource, eventType, encoding)
	ce.SetID(eventID)
	ce.SetType(eventType)
	ce.SetSource(eventSource)
	if err := ce.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return nil, fmt.Errorf("failed to set cloudevent-%s data with error:[%w]", encoding.String(), err)
	}
	return &ce, nil
}
