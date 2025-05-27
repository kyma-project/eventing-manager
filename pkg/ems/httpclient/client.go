package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Perform a compile time check.
var _ BaseURLAwareClient = Client{}

// BaseURLAwareClient is a http client that can build requests not from a full URL, but from a path relative to a configured base url
// this is useful for REST-APIs that always connect to the same host, but on different paths.
type BaseURLAwareClient interface {
	NewRequest(method, path string, body any) (*http.Request, error)
	Do(req *http.Request, result any) (Status, []byte, error)
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type Status struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200
}

// NewHTTPClient creates a new client and ensures that the given baseURL ends with a trailing '/'.
// The trailing '/' is required later for constructing the full URL using a relative path.
func NewHTTPClient(baseURL string, client *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)

	// add trailing '/' to the url path, so that we can combine the url with other paths according to standards
	if !strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path += "/"
	}
	if err != nil {
		return nil, err
	}
	return &Client{
		httpClient: client,
		baseURL:    parsedURL,
	}, nil
}

func (c Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

func (c Client) NewRequest(method, path string, body any) (*http.Request, error) {
	var jsonBody io.ReadWriter
	if body != nil {
		jsonBody = new(bytes.Buffer)
		if err := json.NewEncoder(jsonBody).Encode(body); err != nil {
			return nil, NewError(err)
		}
	}

	pu, err := url.Parse(path)
	if err != nil {
		return nil, NewError(err)
	}
	u := resolveReferenceAsRelative(c.baseURL, pu)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), jsonBody)
	if err != nil {
		return nil, NewError(err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func resolveReferenceAsRelative(base, ref *url.URL) *url.URL {
	return base.ResolveReference(&url.URL{Path: strings.TrimPrefix(ref.Path, "/")})
}

func (c Client) Do(req *http.Request, result any) (Status, []byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp == nil {
			return Status{}, nil, NewError(err)
		}
		return Status{
			resp.Status,
			resp.StatusCode,
		}, nil, NewError(err, WithStatusCode(resp.StatusCode))
	}
	defer func() { _ = resp.Body.Close() }()
	defer c.httpClient.CloseIdleConnections()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Status{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
		}, nil, NewError(err, WithStatusCode(resp.StatusCode))
	}
	if len(body) == 0 {
		return Status{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
		}, nil, nil
	}

	if err := json.Unmarshal(body, result); err != nil {
		return Status{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
		}, nil, fmt.Errorf("unmarshal response failed: %w", NewError(err, WithStatusCode(resp.StatusCode), WithMessage(string(body))))
	}

	return Status{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}, body, nil
}
