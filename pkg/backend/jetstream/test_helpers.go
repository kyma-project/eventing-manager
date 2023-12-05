package jetstream

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/eventing-manager/pkg/env"

	"github.com/nats-io/nats-server/v2/server"

	"github.com/kyma-project/eventing-manager/pkg/logger"

	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	"github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

const (
	DefaultStreamName             = "kyma"
	DefaultJetStreamSubjectPrefix = "kyma"
	DefaultMaxReconnects          = 10
	DefaultMaxInFlights           = 10
)

// TestEnvironment provides mocked resources for tests.
type TestEnvironment struct {
	jsBackend  *JetStream
	logger     *logger.Logger
	natsServer *server.Server
	jsClient   *jetStreamClient
	natsConfig env.NATSConfig
	cleaner    cleaner.Cleaner
	natsPort   int
}

func StartNATSServer(serverOpts ...eventingtesting.NatsServerOpt) (*server.Server, int, error) {
	natsPort, err := eventingtesting.GetFreePort()
	if err != nil {
		return nil, 0, err
	}
	serverOpts = append(serverOpts, eventingtesting.WithPort(natsPort))
	natsServer := eventingtesting.RunNatsServerOnPort(serverOpts...)
	return natsServer, natsPort, nil
}

func NewNatsMessagePayload(data, id, source, eventTime, eventType string) string {
	jsonCE := fmt.Sprintf("{\"data\":\"%s\",\"datacontenttype\":\"application/json\""+
		",\"id\":\"%s\",\"source\":\"%s\",\"specversion\":\"1.0\",\"time\":\"%s\""+
		",\"type\":\"%s\"}", data, id, source, eventTime, eventType)
	return jsonCE
}

func SendEventToJetStream(jsClient *JetStream, data string) error {
	// assumption: the event-type used for publishing is already cleaned from none-alphanumeric characters
	// because the publisher-application should have cleaned it already before publishing
	eventType := eventingtesting.OrderCreatedCleanEvent
	eventTime := time.Now().Format(time.RFC3339)
	sampleEvent := NewNatsMessagePayload(data, "id", eventingtesting.EventSourceClean, eventTime, eventType)
	return jsClient.Conn.Publish(jsClient.GetJetStreamSubject(eventingtesting.EventSourceClean,
		eventType,
		v1alpha2.TypeMatchingStandard,
	), []byte(sampleEvent))
}

func SendCloudEventToJetStream(jetStreamClient *JetStream, subject, eventData, cetype string) error {
	// create a CE http request
	var headers http.Header
	if cetype == types.ContentModeBinary {
		headers = eventingtesting.GetBinaryMessageHeaders()
	} else {
		headers = eventingtesting.GetStructuredMessageHeaders()
	}
	req, err := http.NewRequest(http.MethodPost, "dummy", bytes.NewBuffer([]byte(eventData)))
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header[k] = v
	}
	// convert  to the CE Event
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	message := cehttp.NewMessageFromHttpRequest(req)
	defer func() { _ = message.Finish(nil) }()
	event, err := binding.ToEvent(ctx, message)
	if err != nil {
		return err
	}
	if validateErr := event.Validate(); validateErr != nil {
		return validateErr
	}
	// get a CE sender for the embedded NATS using CE-SDK
	natsOpts := cenats.NatsOptions()
	url := jetStreamClient.Config.URL
	sender, err := cenats.NewSender(url, subject, natsOpts)
	if err != nil {
		return err
	}
	client, err := cloudevents.NewClient(sender)
	if err != nil {
		return err
	}
	// force binary binding and send the event to NATS using CE-SDK
	if cetype == types.ContentModeBinary {
		ctx = binding.WithForceBinary(ctx)
	} else {
		ctx = binding.WithForceStructured(ctx)
	}
	if err := client.Send(ctx, *event); err != nil {
		return err
	}
	return nil
}

func AddJSCleanEventTypesToStatus(sub *v1alpha2.Subscription, cleaner cleaner.Cleaner) {
	cleanEventType := GetCleanEventTypes(sub, cleaner)
	sub.Status.Types = cleanEventType
}
