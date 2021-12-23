package options

import (
	"time"

	"github.com/spf13/pflag"
	componentbaseconfig "k8s.io/component-base/config"

	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/webhooks"
)

// Options hold the command-line options about crane manager
type Options struct {
	// LeaderElection hold the configurations for manager leader election.
	LeaderElection componentbaseconfig.LeaderElectionConfiguration
	// MetricsAddr is The address the metric endpoint binds to.
	MetricsAddr string
	// BindAddr is The address the probe endpoint binds to.
	BindAddr string

	PredictionUpdateFrequency time.Duration
	// DataSource is the datasource of the predictor, such as prometheus, nodelocal, etc.
	DataSource string
	// DataSourcePromConfig is the prometheus datasource config
	DataSourcePromConfig providers.PromConfig
	// DataSourceMockConfig is the mock data provider
	DataSourceMockConfig providers.MockConfig

	// AlgorithmModelConfig
	AlgorithmModelConfig config.AlgorithmModelConfig

	// WebhookConfig
	WebhookConfig webhooks.WebhookConfig
}

// NewOptions builds an empty options.
func NewOptions() *Options {
	return &Options{}
}

// Complete completes all the required options.
func (o *Options) Complete() error {
	return nil
}

// Validate all required options.
func (o *Options) Validate() error {
	return nil
}

// AddFlags adds flags to the specified FlagSet.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.MetricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flags.StringVar(&o.BindAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flags.BoolVar(&o.LeaderElection.LeaderElect, "leader-elect", true, "Start a leader election client and gain leadership before executing the main loop. Enable this when running replicated components for high availability.")
	flags.DurationVar(&o.LeaderElection.LeaseDuration.Duration, "lease-duration", 15*time.Second,
		"Specifies the expiration period of lease.")
	flags.DurationVar(&o.LeaderElection.RetryPeriod.Duration, "lease-retry-period", 2*time.Second,
		"Specifies the lease renew interval.")
	flags.DurationVar(&o.LeaderElection.RenewDeadline.Duration, "lease-renew-period", 10*time.Second,
		"Specifies the lease renew interval.")

	flags.DurationVar(&o.PredictionUpdateFrequency, "prediction-update-frequency-duration", 30*time.Second,
		"Specifies the update frequency of the prediction.")
	flags.StringVar(&o.DataSource, "datasource", "prom", "data source of the predictor, prom, mock is available")
	flags.StringVar(&o.DataSourcePromConfig.Address, "prometheus-address", "", "prometheus address")
	flags.StringVar(&o.DataSourcePromConfig.Auth.Username, "prometheus-auth-username", "", "prometheus auth username")
	flags.StringVar(&o.DataSourcePromConfig.Auth.Password, "prometheus-auth-password", "", "prometheus auth password")
	flags.StringVar(&o.DataSourcePromConfig.Auth.BearerToken, "prometheus-auth-bearertoken", "", "prometheus auth bearertoken")
	flags.IntVar(&o.DataSourcePromConfig.QueryConcurrency, "prometheus-query-concurrency", 10, "prometheus query concurrency")
	flags.BoolVar(&o.DataSourcePromConfig.InsecureSkipVerify, "prometheus-insecure-skip-verify", false, "prometheus insecure skip verify")
	flags.DurationVar(&o.DataSourcePromConfig.KeepAlive, "prometheus-keepalive", 60*time.Second, "prometheus keep alive")
	flags.DurationVar(&o.DataSourcePromConfig.Timeout, "prometheus-timeout", 60*time.Second, "prometheus timeout")
	flags.BoolVar(&o.DataSourcePromConfig.BRateLimit, "prometheus-bratelimit", false, "prometheus bratelimit")
	flags.StringVar(&o.DataSourceMockConfig.SeedFile, "seed-file", "", "mock provider seed file")

	flags.DurationVar(&o.AlgorithmModelConfig.UpdateInterval, "model-update-interval", 12*time.Hour, "algorithm model update interval, now used for dsp model update interval")

	flags.BoolVar(&o.WebhookConfig.Enabled, "webhook-enabled", true, "whether enable webhook or not, default to true")
}
