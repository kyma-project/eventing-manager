package http

import (
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudeventsclient "github.com/cloudevents/sdk-go/v2/client"
)

func NewHttpClient(transport *http.Transport) *http.Client {
	return &http.Client{Transport: transport}
}

func NewCloudEventsClient(transport *http.Transport) (*cloudeventsclient.Client, error) {
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

func NewTransport(maxIdleConns, maxConnsPerHost, maxIdleConnsPerHost int, idleConnTimeout time.Duration) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = maxIdleConns
	transport.MaxConnsPerHost = maxConnsPerHost
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	transport.IdleConnTimeout = idleConnTimeout
	return transport
}
