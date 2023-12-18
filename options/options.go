package options

import (
	"flag"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	// All the available arguments.
	argNameMaxReconnects   = "max-reconnects"
	argNameMetricsAddr     = "metrics-addr"
	argNameReconnectWait   = "reconnect-wait"
	argNameReconcilePeriod = "reconcile-period"
	argNameProbeAddr       = "health-probe-bind-addr"
	argNameReadyEndpoint   = "ready-check-endpoint"
	argNameHealthEndpoint  = "health-check-endpoint"

	// All the available environment variables.
	envNameLogFormat = "APP_LOG_FORMAT"
	envNameLogLevel  = "APP_LOG_LEVEL"

	// default values
	defaultMaxReconnects   = 10
	defaultMetricsAddr     = ":8080"
	defaultReconnectWait   = 3 * time.Second
	defaultReconcilePeriod = time.Minute * 10
	defaultProbeAddr       = ":8081"
	defaultReadyEndpoint   = "readyz"
	defaultHealthEndpoint  = "healthz"
)

// Options represents the controller options.
type Options struct {
	Args
	Env
}

// Args represents the controller command-line arguments.
type Args struct {
	MaxReconnects   int
	MetricsAddr     string
	ReconnectWait   time.Duration
	ReconcilePeriod time.Duration
	ProbeAddr       string
	ReadyEndpoint   string
	HealthEndpoint  string
}

// Env represents the controller environment variables.
type Env struct {
	LogFormat string `default:"json" envconfig:"APP_LOG_FORMAT"`
	LogLevel  string `default:"warn" envconfig:"APP_LOG_LEVEL"`
}

// New returns a new Options instance.
func New() *Options {
	return &Options{}
}

// Parse parses the controller options.
func (o *Options) Parse() error {
	flag.IntVar(&o.MaxReconnects, argNameMaxReconnects, defaultMaxReconnects, "Maximum number of reconnect attempts (NATS).")
	flag.StringVar(&o.MetricsAddr, argNameMetricsAddr, defaultMetricsAddr, "The address the metric endpoint binds to.")
	flag.DurationVar(&o.ReconnectWait, argNameReconnectWait, defaultReconnectWait, "Wait time between reconnect attempts (NATS).")
	flag.DurationVar(&o.ReconcilePeriod, argNameReconcilePeriod, defaultReconcilePeriod, "Period between triggering of reconciling calls (BEB).")
	flag.StringVar(&o.ProbeAddr, argNameProbeAddr, defaultProbeAddr, "The TCP address that the controller should bind to for serving health probes.")
	flag.StringVar(&o.ReadyEndpoint, argNameReadyEndpoint, defaultReadyEndpoint, "The endpoint of the readiness probe.")
	flag.StringVar(&o.HealthEndpoint, argNameHealthEndpoint, defaultHealthEndpoint, "The endpoint of the health probe.")
	flag.Parse()

	if err := envconfig.Process("", &o.Env); err != nil {
		return err
	}

	return nil
}

// String implements the fmt.Stringer interface.
func (o Options) String() string {
	return fmt.Sprintf("--%s=%v --%s=%v --%s=%v --%s=%v --%s=%v --%s=%v %s=%v %s=%v %s=%v",
		argNameMaxReconnects, o.MaxReconnects,
		argNameMetricsAddr, o.MetricsAddr,
		argNameReconnectWait, o.ReconnectWait,
		argNameReconcilePeriod, o.ReconcilePeriod,
		argNameProbeAddr, o.ProbeAddr,
		argNameReadyEndpoint, o.ReadyEndpoint,
		argNameHealthEndpoint, o.HealthEndpoint,
		envNameLogFormat, o.LogFormat,
		envNameLogLevel, o.LogLevel,
	)
}
