package eventing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/kyma-project/eventing-manager/hack/e2e/common"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
