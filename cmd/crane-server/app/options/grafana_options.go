package options

import (
	"fmt"
	"os"

	"gopkg.in/gcfg.v1"

	"github.com/spf13/pflag"

	serverconfig "github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/service/dashboard"
)

// GrafanaOptions contains configuration items related to grafana.
type GrafanaOptions struct {
	configFile string
	dashboard.GrafanaConfig
}

// NewGrafanaOptions creates a GrafanaOptions object with default parameters.
func NewGrafanaOptions() *GrafanaOptions {
	return &GrafanaOptions{}
}

func (o *GrafanaOptions) Complete() error {
	var gconfig dashboard.GrafanaConfig
	grafanaConfig, err := os.Open(o.configFile)
	if err != nil {
		return err
	}
	defer grafanaConfig.Close()

	if err := gcfg.FatalOnly(gcfg.ReadInto(&gconfig, grafanaConfig)); err != nil {
		return err
	}
	o.GrafanaConfig = gconfig
	return nil
}

func (o *GrafanaOptions) ApplyTo(cfg *serverconfig.Config) error {
	cfg.GrafanaConfig = &o.GrafanaConfig
	return nil
}

func (o *GrafanaOptions) Validate() []error {
	if o.APIKey == "" && o.Username == "" && o.Password == "" {
		return []error{fmt.Errorf("no apikey or username&password specified")}
	}
	return []error{}
}

// AddFlags adds flags related to features for a specific Grafana to the
// specified FlagSet.
func (o *GrafanaOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		return
	}

	fs.StringVar(&o.configFile, "grafana.configfile", o.configFile,
		"Grafana config file, file contents is grafana config")

	//fs.StringVar(&o.Scheme, "grafana.scheme", o.Scheme,
	//	"Grafana scheme")
	//
	//fs.StringVar(&o.Host, "grafana.host", o.Host,
	//	"Grafana host/domain")
	//
	//fs.StringVar(&o.Username, "grafana.username", o.Username,
	//	"Grafana username")
	//fs.StringVar(&o.Password, "grafana.password", o.Password,
	//	"Grafana password")
	//fs.StringVar(&o.APIKey, "grafana.apikey", o.APIKey,
	//	"Grafana apikey")
}
