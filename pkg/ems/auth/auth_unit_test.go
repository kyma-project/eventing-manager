package auth

import (
	"net/http"
	"testing"

	"golang.org/x/oauth2"

	"github.com/kyma-project/eventing-manager/pkg/env"
)

const (
	// default value in httpTransport implementation.
	maxIdleConns        = 100
	maxIdleConnsPerHost = 0
)

func TestAuthenticator(t *testing.T) {
	t.Helper()
	t.Setenv("CLIENT_ID", "foo")
	t.Setenv("CLIENT_SECRET", "foo")
	t.Setenv("TOKEN_ENDPOINT", "foo")
	cfg := env.Config{}
	// authenticate
	client, err := NewAuthenticatedClient(cfg)
	if err != nil {
		t.Error(err)
	}

	secTransport, ok := client.Transport.(*oauth2.Transport)
	if !ok {
		t.Errorf("convert to oauth2 transport failed")
	}
	httpTransport, ok := secTransport.Base.(*http.Transport)
	if !ok {
		t.Errorf("convert to HTTP transport failed")
	}

	if httpTransport.MaxIdleConns != maxIdleConns {
		t.Errorf("HTTP Client Transport MaxIdleConns is misconfigured want: %d but got: %d", maxIdleConns, httpTransport.MaxIdleConns)
	}
	if httpTransport.MaxIdleConnsPerHost != maxIdleConnsPerHost {
		t.Errorf("HTTP Client Transport MaxIdleConnsPerHost is misconfigured want: %d but got: %d", maxIdleConnsPerHost, httpTransport.MaxIdleConnsPerHost)
	}
}
