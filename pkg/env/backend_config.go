package env

import (
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// BackendConfig represents the environment config for the Backend Controller.
type BackendConfig struct {
	PublisherConfig PublisherConfig

	WebhookSecretName   string `envconfig:"WEBHOOK_SECRET_NAME" default:"eventing-webhook-server-cert"`
	MutatingWebhookName string `envconfig:"MUTATING_WEBHOOK_NAME" default:"subscription-mutating-webhook-configuration"`
	//nolint:lll
	ValidatingWebhookName string `envconfig:"VALIDATING_WEBHOOK_NAME" default:"subscription-validating-webhook-configuration"`

	DefaultSubscriptionConfig DefaultSubscriptionConfig

	//nolint:lll
	EventingWebhookAuthSecretName string `envconfig:"EVENTING_WEBHOOK_AUTH_SECRET_NAME" required:"true" default:"eventing-webhook-auth"`
	//nolint:lll
	EventingWebhookAuthSecretNamespace string `envconfig:"EVENTING_WEBHOOK_AUTH_SECRET_NAMESPACE" required:"true" default:"kyma-system"`
}

type PublisherConfig struct {
	Image             string `envconfig:"PUBLISHER_IMAGE" default:"eu.gcr.io/kyma-project/event-publisher-proxy:c06eb4fc"`
	ImagePullPolicy   string `envconfig:"PUBLISHER_IMAGE_PULL_POLICY" default:"IfNotPresent"`
	PortNum           int    `envconfig:"PUBLISHER_PORT_NUM" default:"8080"`
	MetricsPortNum    int    `envconfig:"PUBLISHER_METRICS_PORT_NUM" default:"8080"`
	ServiceAccount    string `envconfig:"PUBLISHER_SERVICE_ACCOUNT" default:"eventing-publisher-proxy"`
	RequestTimeout    string `envconfig:"PUBLISHER_REQUEST_TIMEOUT" default:"5s"`
	PriorityClassName string `envconfig:"PUBLISHER_PRIORITY_CLASS_NAME" default:""`
	// publisher takes the controller values
	AppLogFormat string `envconfig:"APP_LOG_FORMAT" default:"json"`
}

type DefaultSubscriptionConfig struct {
	MaxInFlightMessages   int           `envconfig:"DEFAULT_MAX_IN_FLIGHT_MESSAGES" default:"10"`
	DispatcherRetryPeriod time.Duration `envconfig:"DEFAULT_DISPATCHER_RETRY_PERIOD" default:"5m"`
	DispatcherMaxRetries  int           `envconfig:"DEFAULT_DISPATCHER_MAX_RETRIES" default:"10"`
}

func GetBackendConfig() BackendConfig {
	cfg := BackendConfig{}
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}
