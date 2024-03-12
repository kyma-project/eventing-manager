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

func ExtractLegacyTypeFromEventType(eventSource, eventVersion, eventType string) string {
	if len(strings.TrimSpace(eventType)) == 0 {
		return ""
	}

	startIndex := -1
	start := fmt.Sprintf("%s.", eventSource)
	if len(strings.TrimSpace(eventSource)) == 0 {
		startIndex = 0
	} else if startIndex = strings.Index(eventType, start); startIndex < 0 {
		startIndex = 0
	} else {
		startIndex += len(start)
	}

	endIndex := -1
	end := fmt.Sprintf(".%s", eventVersion)
	if len(strings.TrimSpace(eventVersion)) == 0 {
		endIndex = len(eventType)
	} else if endIndex = strings.LastIndex(eventType, end); endIndex < 0 {
		endIndex = len(eventType)
	}

	return eventType[startIndex:endIndex]
}

func ExtractVersionFromEventType(eventType string) string {
	segments := strings.Split(eventType, ".")
	return segments[len(segments)-1]
}

func NewLegacyEvent(eventSource, eventType string) (string, string, string, string) {
	// - If the eventType is order.created.v1 and source is noapp, then for legacy event:
	//   - eventSource should be: noapp
	//   - eventType should be: order.created
	//   - eventVersion should be: v1
	// - If the eventType is sap.kyma.custom.noapp.order.created.v1 and source is noapp, then for legacy event:
	//   - eventSource should be: noapp
	//   - eventType should be: order.created
	//   - eventVersion should be: v1
	eventID := uuid.New().String()
	eventVersion := ExtractVersionFromEventType(eventType)
	legacyEventType := ExtractLegacyTypeFromEventType(eventSource, eventVersion, eventType)
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
