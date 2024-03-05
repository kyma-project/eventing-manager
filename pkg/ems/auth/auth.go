package auth

import (
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/signals"
)

var ErrCreatingFailed = errors.New("cannot create new authenticated client")

func NewAuthenticatedClient(cfg env.Config) (*http.Client, error) {
	ctx := signals.NewReusableContext()
	config := getDefaultOauth2Config(cfg)
	// create and configure oauth2 client
	client := config.Client(ctx)

	trnsprt, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, ErrCreatingFailed
	}
	base := trnsprt.Clone()
	clntTrnsprt, ok := client.Transport.(*oauth2.Transport)
	if !ok {
		return nil, ErrCreatingFailed
	}
	clntTrnsprt.Base = base

	return client, nil
}
