package eventing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/kyma-project/eventing-manager/hack/e2e/common"
)

const (
	LegacyPublishEndpointFormat     = "%s/%s/v1/events"
	CloudEventPublishEndpointFormat = "%s/publish"
)

var (
	ErrFailedSendingCE      = errors.New("failed to send cloudevent")
	ErrEncodingNotSupported = errors.New("unsupported cloudevent encoding")
)

type Publisher struct {
	clientCE     client.Client
	clientHTTP   *http.Client
	publisherURL string
	logger       *zap.Logger
}

func NewPublisher(clientCE client.Client, clientHTTP *http.Client, publisherURL string, logger *zap.Logger) *Publisher {
	return &Publisher{
		clientCE:     clientCE,
		clientHTTP:   clientHTTP,
		publisherURL: publisherURL,
		logger:       logger,
	}
}

func (p *Publisher) LegacyPublishEndpoint(source string) string {
	return fmt.Sprintf(LegacyPublishEndpointFormat, p.publisherURL, source)
}

func (p *Publisher) PublishEndpoint() string {
	return fmt.Sprintf(CloudEventPublishEndpointFormat, p.publisherURL)
}

func (p *Publisher) SendLegacyEventWithRetries(source, eventType, payload string, attempts int, interval time.Duration) error {
	return common.Retry(attempts, interval, func() error {
		return p.SendLegacyEvent(source, eventType, payload)
	})
}

func (p *Publisher) SendCloudEventWithRetries(event *cloudevents.Event, encoding binding.Encoding, attempts int, interval time.Duration) error {
	return common.Retry(attempts, interval, func() error {
		return p.SendCloudEvent(event, encoding)
	})
}

func (p *Publisher) SendLegacyEvent(source, eventType, payload string) error {
	ctx := context.Background()
	url := p.LegacyPublishEndpoint(source)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		err = errors.Wrap(err, "Failed to create HTTP request for sending legacy event")
		p.logger.Debug(err.Error())
		return err
	}

	p.logger.Debug(fmt.Sprintf("Publishing legacy event:"+
		" URL: %s,"+
		" EventSource: %s,"+
		" EventType: %s,"+
		" Payload: %s",
		url, source, eventType, payload))

	req.Header.Set("Content-Type", "application/json")
	resp, err := p.clientHTTP.Do(req)
	if err != nil {
		err = errors.Wrap(err, "Failed to send legacy-event")
		p.logger.Debug(err.Error())
		return err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			p.logger.Error(err.Error())
		}
	}()

	// if success, then return.
	if Is2XX(resp.StatusCode) {
		return nil
	}

	// read body and return error.
	if body, err2 := io.ReadAll(resp.Body); err2 != nil {
		err2 = errors.Wrap(err2, "Failed to read response body")
		p.logger.Debug(err2.Error())
		return err2
	} else {
		err2 = errors.New(fmt.Sprintf("Failed to send legacy-event:[%s] response:[%d] body:[%s]", eventType,
			resp.StatusCode, string(body)))
		p.logger.Debug(err2.Error())
		return err2
	}
}

func (p *Publisher) SendCloudEvent(event *cloudevents.Event, encoding binding.Encoding) error {
	newCtx := context.Background()
	ctx := cloudevents.ContextWithTarget(newCtx, p.PublishEndpoint())
	//nolint:exhaustive // we only support the two checked encodings. Every other encoding will result in an error.
	switch encoding {
	case binding.EncodingBinary:
		{
			ctx = binding.WithForceBinary(ctx)
		}
	case binding.EncodingStructured:
		{
			ctx = binding.WithForceStructured(ctx)
		}
	default:
		{
			return fmt.Errorf("%w:[%s]", ErrEncodingNotSupported, encoding.String())
		}
	}

	evnt := *event
	p.logger.Debug(fmt.Sprintf("Publishing cloud event:"+
		" URL: %s,"+
		" Encoding: %s,"+
		" EventID: %s,"+
		" EventSource: %s,"+
		" EventType: %s,"+
		" Payload: %s",
		p.PublishEndpoint(), encoding.String(), evnt.ID(), evnt.Source(), evnt.Type(), evnt.Data()))

	result := p.clientCE.Send(ctx, evnt)

	//nolint:errorlint // this error is used in logs
	switch {
	case cloudevents.IsUndelivered(result):
		{
			return fmt.Errorf("%w: encoding:[%s] undelivered:[%s] response:[%s]", ErrFailedSendingCE, encoding.String(), evnt.Type(), result)
		}
	case cloudevents.IsNACK(result):
		{
			return fmt.Errorf("%w: encoding:[%s] nack:[%s] response:[%s]", ErrFailedSendingCE, encoding.String(), evnt.Type(), result)
		}
	case cloudevents.IsACK(result):
		{
			p.logger.Debug(fmt.Sprintf("successfully sent cloudevent-%s [%s]", encoding.String(), evnt.Type()))
			return nil
		}
	default:
		{
			return fmt.Errorf("%w: encoding:[%s] unknown:[%s] response:[%s]", ErrFailedSendingCE, encoding.String(), evnt.Type(), result)
		}
	}
}
