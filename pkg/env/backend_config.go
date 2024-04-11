package env

import (
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// BackendConfig represents the environment config for the Backend Controller.
type BackendConfig struct {
	PublisherConfig PublisherConfig

	// namespace where eventing-manager is deployed.
	Namespace           string `default:"kyma-system" envconfig:"NAMESPACE"`
	EventingCRName      string `default:"eventing"    envconfig:"EVENTING_CR_NAME"`
	EventingCRNamespace string `default:"kyma-system" envconfig:"EVENTING_CR_NAMESPACE"`

	DefaultSubscriptionConfig DefaultSubscriptionConfig

	//nolint:lll
	EventingWebhookAuthSecretName string `default:"eventing-webhook-auth" envconfig:"EVENTING_WEBHOOK_AUTH_SECRET_NAME" required:"true"`
	//nolint:lll
	EventingWebhookAuthSecretNamespace string `default:"kyma-system" envconfig:"EVENTING_WEBHOOK_AUTH_SECRET_NAMESPACE" required:"true"`
}

type PublisherConfig struct {
	Image             string `default:"eu.gcr.io/kyma-project/event-publisher-proxy:c06eb4fc" envconfig:"PUBLISHER_IMAGE"`
	ImagePullPolicy   string `default:"IfNotPresent"                                          envconfig:"PUBLISHER_IMAGE_PULL_POLICY"`
	PortNum           int    `default:"8080"                                                  envconfig:"PUBLISHER_PORT_NUM"`
	MetricsPortNum    int    `default:"8080"                                                  envconfig:"PUBLISHER_METRICS_PORT_NUM"`
	ServiceAccount    string `default:"eventing-publisher-proxy"                              envconfig:"PUBLISHER_SERVICE_ACCOUNT"`
	RequestTimeout    string `default:"5s"                                                    envconfig:"PUBLISHER_REQUEST_TIMEOUT"`
	PriorityClassName string `default:""                                                      envconfig:"PUBLISHER_PRIORITY_CLASS_NAME"`
	// publisher takes the controller values
	AppLogFormat          string `default:"json" envconfig:"APP_LOG_FORMAT"`
	ApplicationCRDEnabled bool
}

type DefaultSubscriptionConfig struct {
	MaxInFlightMessages   int           `default:"10" envconfig:"DEFAULT_MAX_IN_FLIGHT_MESSAGES"`
	DispatcherRetryPeriod time.Duration `default:"5m" envconfig:"DEFAULT_DISPATCHER_RETRY_PERIOD"`
	DispatcherMaxRetries  int           `default:"10" envconfig:"DEFAULT_DISPATCHER_MAX_RETRIES"`
}

func GetBackendConfig() BackendConfig {
	cfg := BackendConfig{}
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}
