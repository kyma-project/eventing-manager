package eventmesh

import (
	"fmt"
	"net/url"
)

type HTTPStatusError struct {
	StatusCode int
}

func (e HTTPStatusError) Error() string {
	return fmt.Sprintf("%v", e.StatusCode)
}

func (e *HTTPStatusError) Is(target error) bool {
	t, ok := target.(*HTTPStatusError) //nolint: errorlint // converted to pointer and checked.
	if !ok {
		return false
	}
	return e.StatusCode == t.StatusCode
}

type OAuth2ClientCredentials struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	CertsURL     string
}

func (o OAuth2ClientCredentials) GetIssuer() (string, error) {
	// parsing it to extract hostname only.
	parsedURL, err := url.ParseRequestURI(o.CertsURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host), nil
}

func (o OAuth2ClientCredentials) GetJwkURI() string {
	return o.CertsURL
}

type Response struct {
	StatusCode int
	Error      error
}
