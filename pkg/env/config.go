package env

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	backendKey = "BACKEND"

	backendValueBEB  = "BEB"
	backendValueNats = "NATS"
)

// Backend returns the selected backend based on the environment variable
// "BACKEND". "NATS" is the default value in case of an empty variable.
func Backend() (string, error) {
	backend := strings.ToUpper(os.Getenv(backendKey))

	switch backend {
	case backendValueBEB:
		return backendValueBEB, nil
	case backendValueNats, "":
		return backendValueNats, nil
	default:
		return "", fmt.Errorf("invalid BACKEND set: %v", backend)
	}
}

// Config represents the environment config for the Eventing Controller.
type Config struct {
	// Following details are for eventing-controller to communicate to BEB
	BEBAPIURL     string `default:"https://enterprise-messaging-pubsub.cfapps.sap.hana.ondemand.com/sap/ems/v1" envconfig:"BEB_API_URL"`
	ClientID      string `default:"client-id"                                                                   envconfig:"CLIENT_ID"`
	ClientSecret  string `default:"client-secret"                                                               envconfig:"CLIENT_SECRET"`
	TokenEndpoint string `default:"token-endpoint"                                                              envconfig:"TOKEN_ENDPOINT"`

	// Following details are for BEB to communicate to Kyma
	WebhookActivationTimeout time.Duration `default:"60s" envconfig:"WEBHOOK_ACTIVATION_TIMEOUT"`

	// Default protocol setting for BEB
	ExemptHandshake bool   `default:"true"          envconfig:"EXEMPT_HANDSHAKE"`
	Qos             string `default:"AT_LEAST_ONCE" envconfig:"QOS"`
	ContentMode     string `default:""              envconfig:"CONTENT_MODE"`

	// Default namespace for BEB
	BEBNamespace string `default:"ns" envconfig:"BEB_NAMESPACE"`

	// EventTypePrefix prefix for the EventType
	// note: eventType format is <prefix>.<application>.<event>.<version>
	EventTypePrefix string `envconfig:"EVENT_TYPE_PREFIX" required:"true"`

	// EventingWebhookAuthEnabled enable/disable the Eventing webhook auth feature flag.
	EventingWebhookAuthEnabled bool `default:"false" envconfig:"EVENTING_WEBHOOK_AUTH_ENABLED" required:"false"`

	// NATSProvisioningEnabled enable/disable the NATS resources provisioning feature flag.
	NATSProvisioningEnabled bool `default:"true" envconfig:"NATS_PROVISIONING_ENABLED" required:"false"`
}

func GetConfig() Config {
	cfg := Config{}
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}
