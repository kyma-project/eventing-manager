package auth

import (
	"net/http"

	"golang.org/x/oauth2"

	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/signals"
)

func NewAuthenticatedClient(cfg env.Config) *http.Client {
	ctx := signals.NewReusableContext()
	config := getDefaultOauth2Config(cfg)
	// create and configure oauth2 client
	client := config.Client(ctx)

	base := http.DefaultTransport.(*http.Transport).Clone()
	client.Transport.(*oauth2.Transport).Base = base

	return client
}
