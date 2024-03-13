package eventing

import (
	"fmt"
	"net/http"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/google/uuid"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
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

func ExtractLegacyTypeFromSubscriptionV1Alpha2Type(eventVersion, eventType string, typeMatching eventingv1alpha2.TypeMatching) string {
	if typeMatching == eventingv1alpha2.TypeMatchingStandard {
		return strings.TrimSuffix(eventType, fmt.Sprintf(".%s", eventVersion))
	}

	// Assumption: The event type consists of at least 3 parts separated by the "." character.
	parts := strings.Split(eventType, ".")
	if len(parts) < 3 {
		return ""
	}
	parts = parts[len(parts)-3 : len(parts)-1]
	return strings.Join(parts, ".")
}

func ExtractVersionFromEventType(eventType string) string {
	segments := strings.Split(eventType, ".")
	return segments[len(segments)-1]
}

func NewLegacyEvent(eventSource, eventType string, typeMatching eventingv1alpha2.TypeMatching) (string, string, string, string) {
	// If the eventType is order.created.v1 and source is noapp, then for legacy event:
	// eventSource should be: noapp
	// eventType should be: order.created
	// eventVersion should be: v1
	eventID := uuid.New().String()
	eventVersion := ExtractVersionFromEventType(eventType)
	legacyEventType := ExtractLegacyTypeFromSubscriptionV1Alpha2Type(eventVersion, eventType, typeMatching)
	eventData := LegacyEventData(eventSource, legacyEventType)
	payload := LegacyEventPayload(eventID, eventVersion, legacyEventType, eventData)

	return eventID, eventSource, legacyEventType, payload
}

func NewCloudEvent(eventSource, eventType string, encoding binding.Encoding) (*cloudevents.Event, error) {
	eventID := uuid.New().String()
	evnt := cloudevents.NewEvent()
	data := CloudEventData(eventSource, eventType, encoding)
	evnt.SetID(eventID)
	evnt.SetType(eventType)
	evnt.SetSource(eventSource)
	if err := evnt.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return nil, fmt.Errorf("failed to set cloudevent-%s data with error:[%w]", encoding.String(), err)
	}
	return &evnt, nil
}
