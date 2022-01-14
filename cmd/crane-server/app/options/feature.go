package options

import (
	"github.com/gin-gonic/gin"
	serverconfig "github.com/gocrane/crane/pkg/server/config"
	"github.com/spf13/pflag"
)

// FeatureOptions contains configuration items related to API server features.
type FeatureOptions struct {
	EnableProfiling bool   `json:"enable-profiling"`
	EnableMetrics   bool   `json:"enable-metrics"`
	Mode            string `json:"mode"`
	EnableGrafana   bool   `json:"enable-grafana"`
}

// NewFeatureOptions creates a FeatureOptions object with default parameters.
func NewFeatureOptions() *FeatureOptions {
	return &FeatureOptions{
		EnableMetrics:   true,
		EnableProfiling: true,
	}
}

func (o *FeatureOptions) Complete() error {

	return nil
}

func (o *FeatureOptions) ApplyTo(cfg *serverconfig.Config) error {
	cfg.EnableProfiling = o.EnableProfiling
	cfg.EnableMetrics = o.EnableMetrics
	cfg.Mode = o.Mode
	cfg.EnableGrafana = o.EnableGrafana
	return nil
}

func (o *FeatureOptions) Validate() []error {
	return []error{}
}

// AddFlags adds flags related to features for a specific api server to the
// specified FlagSet.
func (o *FeatureOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		return
	}

	fs.BoolVar(&o.EnableProfiling, "feature.enable-profiling", o.EnableProfiling,
		"Enable profiling via web interface host:port/debug/pprof/")

	fs.BoolVar(&o.EnableMetrics, "feature.enable-metrics", o.EnableMetrics,
		"Enables metrics on the server at /metrics")

	fs.StringVar(&o.Mode, "feature.mode", gin.ReleaseMode,
		"Debug mode of the gin server, support ")

	fs.BoolVar(&o.EnableGrafana, "feature.enable-grafana", o.EnableGrafana,
		"Enable grafana will read grafana config file to requests grafana dashboard")
}
