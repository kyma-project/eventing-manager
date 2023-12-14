package manager

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kyma-project/eventing-manager/pkg/env"
)

const (
	ParamNameClientID     = "client_id"
	ParamNameClientSecret = "client_secret"
	ParamNameTokenURL     = "token_url"
	ParamNameCertsURL     = "certs_url"
)

type Params map[string][]byte

// Manager defines the interface that subscription managers for different messaging backends should implement.
//
//go:generate go run github.com/vektra/mockery/v2 --name=Manager --outpkg=mocks --output=mocks --case=underscore
type Manager interface {
	// Init initializes the subscription manager and passes the controller manager to use.
	Init(mgr manager.Manager) error

	// Start runs the initialized subscription manager instance.
	Start(defaultSubsConfig env.DefaultSubscriptionConfig, params Params) error

	// Stop tells the subscription manager instance to shut down and clean-up.
	Stop(runCleanup bool) error
}
