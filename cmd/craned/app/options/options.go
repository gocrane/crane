package options

import (
	"time"

	"github.com/spf13/pflag"
	componentbaseconfig "k8s.io/component-base/config"

	"github.com/gocrane/crane/pkg/controller/ehpa"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/providers"
	serverconfig "github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/webhooks"
)

// Options hold the command-line options about craned
type Options struct {
	// ApiQps for rest client
	ApiQps int
	// ApiBurst for rest  client
	ApiBurst int
	// LeaderElection hold the configurations for manager leader election.
	LeaderElection componentbaseconfig.LeaderElectionConfiguration
	// MetricsAddr is The address the metric endpoint binds to.
	MetricsAddr string
	// BindAddr is The address the probe endpoint binds to.
	BindAddr string

	PredictionUpdateFrequency time.Duration
	// DataSource is the datasource of the predictor, such as prometheus, nodelocal, etc.
	DataSource []string
	// DataSourcePromConfig is the prometheus datasource config
	DataSourcePromConfig providers.PromConfig
	// DataSourceMockConfig is the mock data provider
	DataSourceMockConfig providers.MockConfig
	// DataSourceGrpcConfig is the config for grpc provider
	DataSourceGrpcConfig providers.GrpcConfig

	// AlgorithmModelConfig
	AlgorithmModelConfig config.AlgorithmModelConfig

	// WebhookConfig
	WebhookConfig webhooks.WebhookConfig

	// RecommendationConfigFile is the configuration file for resource/HPA recommendations.
	// If unspecified, a default is provided.
	RecommendationConfigFile string

	// QOSConfigFile is the configuration file for QOS.
	QOSConfigFile string

	// ServerOptions hold the craned web server options
	ServerOptions *ServerOptions

	// EhpaControllerConfig is the configuration for Ehpa controller
	EhpaControllerConfig ehpa.EhpaControllerConfig

	// RecommendationConfiguration is configuration file for recommendation framework.
	// If unspecified, a default is provided.
	RecommendationConfiguration string

	// OOMRecordMaxNumber is the max number for oom record
	OOMRecordMaxNumber int

	// TimeSeriesPredictionMaxConcurrentReconciles is the max concurrent reconciles for TimeSeriesPrediction controller
	TimeSeriesPredictionMaxConcurrentReconciles int
}

// NewOptions builds an empty options.
func NewOptions() *Options {
	return &Options{
		ServerOptions: NewServerOptions(),
	}
}

// Complete completes all the required options.
func (o *Options) Complete() error {
	return o.ServerOptions.Complete()
}

// Validate all required options.
func (o *Options) Validate() []error {
	return o.ServerOptions.Validate()
}

func (o *Options) ApplyTo(cfg *serverconfig.Config) error {
	return o.ServerOptions.ApplyTo(cfg)
}

// AddFlags adds flags to the specified FlagSet.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	o.ServerOptions.AddFlags(flags)

	flags.IntVar(&o.ApiQps, "api-qps", 300, "QPS of rest config.")
	flags.IntVar(&o.ApiBurst, "api-burst", 400, "Burst of rest config.")
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
	flags.StringSliceVar(&o.DataSource, "datasource", []string{"prom"}, "data source of the predictor, prom, mock is available")
	flags.StringVar(&o.DataSourcePromConfig.Address, "prometheus-address", "", "prometheus address")
	flags.StringVar(&o.DataSourcePromConfig.AdapterConfigMapNS, "prometheus-adapter-configmap-namespace", "", "prometheus adapter-configmap namespace")
	flags.StringVar(&o.DataSourcePromConfig.AdapterConfigMapName, "prometheus-adapter-configmap-name", "", "prometheus adapter-configmap name")
	flags.StringVar(&o.DataSourcePromConfig.AdapterConfigMapKey, "prometheus-adapter-configmap-key", "", "prometheus adapter-configmap key")
	flags.StringVar(&o.DataSourcePromConfig.AdapterConfig, "prometheus-adapter-config", "", "prometheus adapter-config path")
	flags.StringVar(&o.DataSourcePromConfig.AdapterExtensionLabels, "prometheus-adapter-extension-labels", "", "prometheus adapter extension-labels for expressionQuery")
	flags.StringVar(&o.DataSourcePromConfig.Auth.Username, "prometheus-auth-username", "", "prometheus auth username")
	flags.StringVar(&o.DataSourcePromConfig.Auth.Password, "prometheus-auth-password", "", "prometheus auth password")
	flags.StringVar(&o.DataSourcePromConfig.Auth.BearerToken, "prometheus-auth-bearertoken", "", "prometheus auth bearertoken")
	flags.IntVar(&o.DataSourcePromConfig.QueryConcurrency, "prometheus-query-concurrency", 10, "prometheus query concurrency")
	flags.BoolVar(&o.DataSourcePromConfig.InsecureSkipVerify, "prometheus-insecure-skip-verify", false, "prometheus insecure skip verify")
	flags.DurationVar(&o.DataSourcePromConfig.KeepAlive, "prometheus-keepalive", 60*time.Second, "prometheus keep alive")
	flags.DurationVar(&o.DataSourcePromConfig.Timeout, "prometheus-timeout", 3*time.Minute, "prometheus timeout")
	flags.BoolVar(&o.DataSourcePromConfig.BRateLimit, "prometheus-bratelimit", false, "prometheus bratelimit")
	flags.IntVar(&o.DataSourcePromConfig.MaxPointsLimitPerTimeSeries, "prometheus-maxpoints", 11000, "prometheus max points limit per time series")
	flags.StringVar(&o.DataSourceMockConfig.SeedFile, "seed-file", "", "mock provider seed file")
	flags.StringVar(&o.DataSourceGrpcConfig.Address, "grpc-ds-address", "localhost:50051", "grpc data source server address")
	flags.DurationVar(&o.DataSourceGrpcConfig.Timeout, "grpc-ds-timeout", time.Minute, "grpc timeout")
	flags.DurationVar(&o.AlgorithmModelConfig.UpdateInterval, "model-update-interval", 12*time.Hour, "algorithm model update interval, now used for dsp model update interval")
	flags.BoolVar(&o.WebhookConfig.Enabled, "webhook-enabled", true, "whether enable webhook or not, default to true")
	flags.StringVar(&o.RecommendationConfigFile, "recommendation-config-file", "", "recommendation configuration file")
	flags.StringVar(&o.RecommendationConfiguration, "recommendation-configuration-file", "/tmp/recommendation-framework/recommendation_configuration.yaml", "recommendation configuration file")
	flags.StringVar(&o.QOSConfigFile, "qos-config-file", "", "qos configuration file")
	flags.StringSliceVar(&o.EhpaControllerConfig.PropagationConfig.LabelPrefixes, "ehpa-propagation-label-prefixes", []string{}, "propagate labels whose key has the prefix to hpa")
	flags.StringSliceVar(&o.EhpaControllerConfig.PropagationConfig.AnnotationPrefixes, "ehpa-propagation-annotation-prefixes", []string{}, "propagate annotations whose key has the prefix to hpa")
	flags.StringSliceVar(&o.EhpaControllerConfig.PropagationConfig.Labels, "ehpa-propagation-labels", []string{}, "propagate labels whose key is complete matching to hpa")
	flags.StringSliceVar(&o.EhpaControllerConfig.PropagationConfig.Annotations, "ehpa-propagation-annotations", []string{}, "propagate annotations whose key is complete matching to hpa")
	flags.IntVar(&o.OOMRecordMaxNumber, "oom-record-max-number", 10000, "Max number for oom records to store in configmap")
	flags.IntVar(&o.TimeSeriesPredictionMaxConcurrentReconciles, "time-series-prediction-max-concurrent-reconciles", 10, "Max concurrent reconciles for TimeSeriesPrediction controller")
}
