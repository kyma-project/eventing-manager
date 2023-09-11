package eventing

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/kyma-project/eventing-manager/hack/e2e/common"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

const (
	LegacyPublishEndpointFormat     = "%s/%s/v1/events"
	CloudEventPublishEndpointFormat = "%s/%s/v1/events"
)

type Publisher struct {
	ctx          context.Context
	clientCE     client.Client
	clientHTTP   *http.Client
	publisherURL string
	logger       *zap.Logger
}

func NewPublisher(ctx context.Context, clientCE client.Client, clientHTTP *http.Client, publisherURL string, logger *zap.Logger) *Publisher {
	return &Publisher{
		ctx:          ctx,
		clientCE:     clientCE,
		clientHTTP:   clientHTTP,
		publisherURL: publisherURL,
		logger:       logger,
	}
}

func (p *Publisher) LegacyPublishEndpoint(source string) string {
	return fmt.Sprintf(LegacyPublishEndpointFormat, p.publisherURL, source)
}

func (p *Publisher) SendLegacyEventWithRetries(source, eventType, payload string, attempts int, interval time.Duration) error {
	return common.Retry(attempts, interval, func() error {
		return p.SendLegacyEvent(source, eventType, payload)
	})
}

func (p *Publisher) SendLegacyEvent(source, eventType, payload string) error {
	url := p.LegacyPublishEndpoint(source)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(payload)))
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

//func (p *Publisher) sendCloudEvent(application, eventType string, encoding binding.Encoding) {
//	ce := cloudevents.NewEvent()
//	eventType = format.CloudEventType(p.conf.EventTypePrefix, application, eventType)
//	data := format.CloudEventData(application, eventType, encoding)
//	ce.SetType(eventType)
//	ce.SetSource(p.conf.EventSource)
//	if err := ce.SetData(cloudevents.ApplicationJSON, data); err != nil {
//		log.Printf("Failed to set cloudevent-%s data with error:[%s]", encoding.String(), err)
//		return
//	}
//
//	ctx := cloudevents.ContextWithTarget(p.ctx, p.conf.PublishEndpointCloudEvents)
//	switch encoding {
//	case binding.EncodingBinary:
//		{
//			ctx = binding.WithForceBinary(ctx)
//		}
//	case binding.EncodingStructured:
//		{
//			ctx = binding.WithForceStructured(ctx)
//		}
//	default:
//		{
//			log.Printf("Failed to use unsupported cloudevent encoding:[%s]", encoding.String())
//			return
//		}
//	}
//
//	result := p.clientCE.Send(ctx, ce)
//	switch {
//	case cloudevents.IsUndelivered(result):
//		{
//			log.Printf("Failed to send cloudevent-%s undelivered:[%s] response:[%s]", encoding.String(), eventType, result)
//			return
//		}
//	case cloudevents.IsNACK(result):
//		{
//			log.Printf("Failed to send cloudevent-%s nack:[%s] response:[%s]", encoding.String(), eventType, result)
//			return
//		}
//	case cloudevents.IsACK(result):
//		{
//			log.Printf("Sent cloudevent-%s [%s]", encoding.String(), eventType)
//			return
//		}
//	default:
//		{
//			log.Printf("Failed to send cloudevent-%s unknown:[%s] response:[%s]", encoding.String(), eventType, result)
//			return
//		}
//	}
//}
