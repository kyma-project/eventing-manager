package client

import (
	"fmt"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/ems/httpclient"
)

// Perform a compile time check.
var _ PublisherManager = Client{}

const (
	PublishURL           = "/events"
	CreateURL            = "/events/subscriptions"
	ListURL              = "/events/subscriptions"
	GetURLFormat         = "/events/subscriptions/%s"
	UpdateURLFormat      = "/events/subscriptions/%s"
	DeleteURLFormat      = "/events/subscriptions/%s"
	HandshakeURLFormat   = "/events/subscriptions/%s/handshake"
	UpdateStateURLFormat = "/events/subscriptions/%s/state"
)

type EventPublisher interface {
	Publish(cloudEvent cloudevents.Event, qos types.Qos) (*types.PublishResponse, error)
}

type SubscriptionManager interface {
	Create(subscription *types.Subscription) (*types.CreateResponse, error)
	List() (*types.Subscriptions, *types.Response, error)
	Get(name string) (*types.Subscription, *types.Response, error)
	Update(name string, webhookAuth *types.WebhookAuth) (*types.UpdateResponse, error)
	Delete(name string) (*types.DeleteResponse, error)
	TriggerHandshake(name string) (*types.TriggerHandshake, error)
	UpdateState(name string, state types.State) (*types.UpdateStateResponse, error)
}

type PublisherManager interface {
	EventPublisher
	SubscriptionManager
}

//go:generate go run github.com/vektra/mockery/v2 --name PublisherManager
type Client struct {
	client httpclient.BaseURLAwareClient
}

func NewClient(client httpclient.BaseURLAwareClient) *Client {
	return &Client{
		client: client,
	}
}

func (c Client) GetHTTPClient() httpclient.BaseURLAwareClient {
	return c.client
}

func (c Client) Publish(event cloudevents.Event, qos types.Qos) (*types.PublishResponse, error) {
	req, err := c.client.NewRequest(http.MethodPost, PublishURL, event)
	if err != nil {
		return nil, err
	}

	// set required headers
	req.Header.Set("qos", string(qos)) //nolint:canonicalheader // used as required.

	var response types.PublishResponse
	status, responseBody, err := c.client.Do(req, &response)
	if err != nil {
		return nil, err
	}
	response.StatusCode = status.StatusCode
	response.Message = status.Status
	if responseBody != nil {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return &response, nil
}

func (c Client) Create(subscription *types.Subscription) (*types.CreateResponse, error) {
	req, err := c.client.NewRequest(http.MethodPost, CreateURL, subscription)
	if err != nil {
		return nil, err
	}

	var response *types.CreateResponse
	status, responseBody, err := c.client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	if response == nil {
		response = &types.CreateResponse{}
	}
	response.StatusCode = status.StatusCode
	response.Message = status.Status
	if len(responseBody) > 0 {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return response, nil
}

func (c Client) List() (*types.Subscriptions, *types.Response, error) {
	req, err := c.client.NewRequest(http.MethodGet, ListURL, nil)
	if err != nil {
		return nil, nil, err
	}

	var subscriptions *types.Subscriptions
	var response types.Response
	resp, responseBody, err := c.client.Do(req, &subscriptions)
	if err != nil {
		return nil, nil, err
	}

	response.StatusCode = resp.StatusCode
	response.Message = resp.Status
	if subscriptions == nil && len(responseBody) > 0 {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return subscriptions, &response, nil
}

func (c Client) Get(name string) (*types.Subscription, *types.Response, error) {
	req, err := c.client.NewRequest(http.MethodGet, fmt.Sprintf(GetURLFormat, name), nil)
	if err != nil {
		return nil, nil, err
	}

	var subscription *types.Subscription
	var response types.Response
	status, responseBody, err := c.client.Do(req, &subscription)
	if err != nil {
		return nil, nil, err
	}

	response.StatusCode = status.StatusCode
	response.Message = status.Status
	if subscription == nil && len(responseBody) > 0 {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return subscription, &response, nil
}

// Update updates the EventMesh Subscription WebhookAuth config.
func (c Client) Update(name string, webhookAuth *types.WebhookAuth) (*types.UpdateResponse, error) {
	subscription := types.Subscription{WebhookAuth: webhookAuth}
	req, err := c.client.NewRequest(http.MethodPatch, fmt.Sprintf(UpdateURLFormat, name), subscription)
	if err != nil {
		return nil, err
	}

	var response *types.UpdateResponse
	status, responseBody, err := c.client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	if response == nil {
		response = &types.UpdateResponse{}
	}
	response.StatusCode = status.StatusCode
	response.Message = status.Status
	if len(responseBody) > 0 {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return response, nil
}

func (c Client) Delete(name string) (*types.DeleteResponse, error) {
	req, err := c.client.NewRequest(http.MethodDelete, fmt.Sprintf(DeleteURLFormat, name), nil)
	if err != nil {
		return nil, err
	}

	var response types.DeleteResponse
	status, responseBody, err := c.client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	response.StatusCode = status.StatusCode
	response.Message = status.Status
	if len(responseBody) > 0 {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return &response, nil
}

func (c Client) TriggerHandshake(name string) (*types.TriggerHandshake, error) {
	req, err := c.client.NewRequest(http.MethodPost, fmt.Sprintf(HandshakeURLFormat, name), nil)
	if err != nil {
		return nil, err
	}

	var response types.TriggerHandshake
	status, responseBody, err := c.client.Do(req, &response)
	if err != nil {
		return nil, err
	}
	response.StatusCode = status.StatusCode
	response.Message = status.Status
	if len(responseBody) > 0 {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return &response, nil
}

func (c Client) UpdateState(name string, state types.State) (*types.UpdateStateResponse, error) {
	req, err := c.client.NewRequest(http.MethodPut, fmt.Sprintf(UpdateStateURLFormat, name), state)
	if err != nil {
		return nil, err
	}

	var response types.UpdateStateResponse
	status, responseBody, err := c.client.Do(req, &response)
	if err != nil {
		return nil, err
	}
	response.StatusCode = status.StatusCode
	response.Message = status.Status
	if len(responseBody) > 0 {
		response.Message = response.Message + ";" + string(responseBody)
	}

	return &response, nil
}
