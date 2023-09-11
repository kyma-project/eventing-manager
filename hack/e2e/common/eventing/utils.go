package eventing

import (
	"fmt"
	"github.com/cloudevents/sdk-go/v2/binding"
	"net/http"
)

const (
	keyApp  = "app"
	keyMode = "mode"
	keyType = "type"

	version = "v1"
)

func Is2XX(statusCode int) bool {
	return http.StatusOK <= statusCode && statusCode <= http.StatusIMUsed
}

func LegacyEventData(source, eventType string) string {
	return `{\"` + keyApp + `\":\"` + source + `\",\"` + keyMode + `\":\"legacy\",\"` + keyType + `\":\"` + eventType + `\"}`
}
func LegacyEventPayload(source, eventId, eventType, data string) string {
	return `{"data":"` + data + `","event-id":"` + eventId + `","event-type":"` + eventType + `","event-time":"2020-04-02T21:37:00Z","event-type-version":"` + version + `"}`
}

func CloudEventMode(encoding binding.Encoding) string {
	return fmt.Sprintf("ce-%s", encoding.String())
}

func CloudEventData(application, eventType string, encoding binding.Encoding) map[string]interface{} {
	return map[string]interface{}{keyApp: application, keyMode: CloudEventMode(encoding), keyType: eventType}
}

func CloudEventType(prefix, application, eventType string) string {
	return fmt.Sprintf("%s.%s.%s.%s", prefix, application, eventType, version)
}
