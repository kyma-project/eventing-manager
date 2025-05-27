//nolint:canonicalheader // used as required in tracing.
package tracing

import (
	"fmt"
	"net/http"
	"testing"

	ceevent "github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	. "github.com/onsi/gomega"
)

func TestAddTracingHeadersToContext(t *testing.T) {
	g := NewWithT(t)
	testCases := []struct {
		name            string
		event           *ceevent.Event
		expectedHeaders http.Header
	}{
		{
			name: "extensions contain w3c headers",
			event: NewEventWithExtensions(map[string]string{
				traceParentCEExtensionsKey: "foo",
			}),
			expectedHeaders: func() http.Header {
				headers := http.Header{}
				headers.Add(traceParentKey, "foo")
				return headers
			}(),
		}, {
			name: "extensions contain b3 headers",
			event: NewEventWithExtensions(map[string]string{
				b3TraceIDCEExtensionsKey:      "trace",
				b3ParentSpanIDCEExtensionsKey: "parentspan",
				b3SpanIDCEExtensionsKey:       "span",
				b3SampledCEExtensionsKey:      "1",
				b3FlagsCEExtensionsKey:        "1",
			}),
			expectedHeaders: func() http.Header {
				headers := http.Header{}
				headers.Add(b3TraceIDKey, "trace")
				headers.Add(b3ParentSpanIDKey, "parentspan")
				headers.Add(b3SpanIDKey, "span")
				headers.Add(b3SampledKey, "1")
				headers.Add(b3FlagsKey, "1")
				return headers
			}(),
		}, {
			name: "extensions does not contain tracing headers",
			event: NewEventWithExtensions(map[string]string{
				"foo": "bar",
			}),
			expectedHeaders: func() http.Header {
				headers := http.Header{}
				return headers
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			gotContext := AddTracingHeadersToContext(ctx, tc.event)
			g.Expect(cehttp.HeaderFrom(gotContext)).To(Equal(tc.expectedHeaders))
			g.Expect(getTracingExtensions(tc.event)).To(BeEmpty())
		})
	}
}

func getTracingExtensions(event *ceevent.Event) map[string]string {
	traceExtensions := make(map[string]string)
	for extension, setting := range event.Extensions() {
		if extension == traceParentCEExtensionsKey ||
			extension == b3TraceIDCEExtensionsKey ||
			extension == b3ParentSpanIDCEExtensionsKey ||
			extension == b3SpanIDCEExtensionsKey ||
			extension == b3SampledCEExtensionsKey ||
			extension == b3FlagsCEExtensionsKey {
			traceExtensions[extension] = fmt.Sprintf("%v", setting)
		}
	}
	return traceExtensions
}

func NewEventWithExtensions(extensions map[string]string) *ceevent.Event {
	event := &ceevent.Event{
		Context: &ceevent.EventContextV1{},
	}
	for k, v := range extensions {
		event.SetExtension(k, v)
	}
	return event
}
