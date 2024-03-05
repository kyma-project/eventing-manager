package http

import (
	"errors"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	ceclient "github.com/cloudevents/sdk-go/v2/client"
)

var ErrCreatingTransport = errors.New("cannot create transport")

func NewHTTPClient(transport *http.Transport) *http.Client {
	return &http.Client{Transport: transport}
}

func NewCloudEventsClient(transport *http.Transport) (*ceclient.Client, error) {
	p, err := cloudevents.NewHTTP(cloudevents.WithRoundTripper(transport))
	if err != nil {
		return nil, err
	}

	client, err := cloudevents.NewClient(p, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func NewTransport(maxIdleConns, maxConnsPerHost, maxIdleConnsPerHost int, idleConnTimeout time.Duration) (*http.Transport, error) {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, ErrCreatingTransport
	}
	transport := t.Clone()
	transport.MaxIdleConns = maxIdleConns
	transport.MaxConnsPerHost = maxConnsPerHost
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	transport.IdleConnTimeout = idleConnTimeout
	return transport, nil
}
