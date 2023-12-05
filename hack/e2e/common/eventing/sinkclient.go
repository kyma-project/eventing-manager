package eventing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	ceevent "github.com/cloudevents/sdk-go/v2/event"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/kyma-project/eventing-manager/hack/e2e/common"
)

const (
	EventsEndpointFormat = "%s/events/%s"
)

type SinkClient struct {
	ctx        context.Context
	clientHTTP *http.Client
	sinkURL    string
	logger     *zap.Logger
}

type SinkEvent struct {
	// Header stores the non CE events, e.g. X-B3-Sampled and Traceparent
	http.Header
	ceevent.Event
}

func NewSinkClient(ctx context.Context, clientHTTP *http.Client, sinkURL string, logger *zap.Logger) *SinkClient {
	return &SinkClient{
		ctx:        ctx,
		clientHTTP: clientHTTP,
		sinkURL:    sinkURL,
		logger:     logger,
	}
}

func (sc *SinkClient) EventsEndpoint(eventId string) string {
	return fmt.Sprintf(EventsEndpointFormat, sc.sinkURL, eventId)
}

func (sc *SinkClient) GetEventFromSinkWithRetries(eventId string, attempts int, interval time.Duration) (*SinkEvent, error) {
	var gotEvent *SinkEvent
	err := common.Retry(attempts, interval, func() error {
		var err1 error
		gotEvent, err1 = sc.GetEventFromSink(eventId)
		return err1
	})
	return gotEvent, err
}

func (sc *SinkClient) GetEventFromSink(eventId string) (*SinkEvent, error) {
	url := sc.EventsEndpoint(eventId)
	sc.logger.Debug(fmt.Sprintf("Fetching event with ID: %s from the sink URL: %s", eventId, url))

	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer([]byte{}))
	if err != nil {
		err = errors.Wrap(err, "Failed to create HTTP request for fetching event from sink")
		sc.logger.Debug(err.Error())
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := sc.clientHTTP.Do(req)
	if err != nil {
		err = errors.Wrap(err, "Failed to fetch event")
		sc.logger.Debug(err.Error())
		return nil, err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			sc.logger.Error(err.Error())
		}
	}()

	// read body.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "Failed to read response body")
		sc.logger.Debug(err.Error())
		return nil, err
	}

	// if not success, then return error.
	if !Is2XX(resp.StatusCode) {
		err = errors.New(fmt.Sprintf("Failed to fetch eventID:[%s] response:[%d] body:[%s]", eventId,
			resp.StatusCode, string(respBody)))
		sc.logger.Debug(err.Error())
		return nil, err
	}

	// success
	// convert to cloud event object.
	ceEvent := cloudevents.NewEvent()
	err = json.Unmarshal(respBody, &ceEvent)
	if err != nil {
		err = errors.Wrap(err, "failed to convert JSON to CloudEvent")
		sc.logger.Debug(err.Error())
		return nil, err
	}

	return &SinkEvent{
		Event: ceEvent,
	}, nil
}
