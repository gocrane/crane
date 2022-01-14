package options

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"gopkg.in/gcfg.v1"

	serverconfig "github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/service/dashboard"
	"github.com/gocrane/crane/pkg/server/store/secret"
)

// ServerOptions used for craned web server
type ServerOptions struct {
	BindAddress string
	BindPort    int

	EnableProfiling bool
	EnableMetrics   bool
	Mode            string

	EnableGrafana     bool
	GrafanaConfigFile string
	dashboard.GrafanaConfig

	StoreType string
}

func NewServerOptions() *ServerOptions {
	return &ServerOptions{}
}

func (o *ServerOptions) Complete() error {
	if o.EnableGrafana {
		var gconfig dashboard.GrafanaConfig
		grafanaConfig, err := os.Open(o.GrafanaConfigFile)
		if err != nil {
			return err
		}
		defer grafanaConfig.Close()

		if err := gcfg.FatalOnly(gcfg.ReadInto(&gconfig, grafanaConfig)); err != nil {
			return err
		}
		o.GrafanaConfig = gconfig
	}
	return nil
}

func (o *ServerOptions) ApplyTo(cfg *serverconfig.Config) error {
	cfg.BindAddress = o.BindAddress
	cfg.BindPort = o.BindPort

	cfg.Mode = o.Mode
	cfg.EnableMetrics = o.EnableMetrics
	cfg.EnableProfiling = o.EnableProfiling

	cfg.EnableGrafana = o.EnableGrafana
	cfg.GrafanaConfig = &o.GrafanaConfig
	cfg.StoreType = o.StoreType
	return nil
}

func (o *ServerOptions) Validate() []error {
	var errors []error

	if o.EnableGrafana {
		if o.APIKey == "" && o.Username == "" && o.Password == "" {
			errors = append(errors, fmt.Errorf("no apikey or username&password specified"))
		}
	}

	if o.BindPort < 0 || o.BindPort > 65535 {
		errors = append(
			errors,
			fmt.Errorf(
				"--server-bind-port %v must be between 0 and 65535, inclusive. 0 for turning off insecure (HTTP) port",
				o.BindPort,
			),
		)
	}

	if strings.ToLower(o.StoreType) != secret.StoreType {
		errors = append(errors, fmt.Errorf("--server-store only support secret now"))
	}

	return errors
}

// AddFlags adds flags related to features for a specific server option to the
// specified FlagSet.
func (o *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		return
	}

	fs.StringVar(&o.BindAddress, "server-bind-address", "0.0.0.0", ""+
		"The IP address on which to serve the --server-bind-port "+
		"(set to 0.0.0.0 for all IPv4 interfaces and :: for all IPv6 interfaces).")
	fs.IntVar(&o.BindPort, "server-bind-port", 8082,
		"The port on which to serve unsecured, unauthenticated access")

	fs.BoolVar(&o.EnableProfiling, "server-enable-profiling", o.EnableProfiling,
		"Enable profiling via web interface host:port/debug/pprof/")

	fs.BoolVar(&o.EnableMetrics, "server-enable-metrics", o.EnableMetrics,
		"Enables metrics on the server at /metrics")

	fs.StringVar(&o.Mode, "server-mode", gin.ReleaseMode,
		"Debug mode of the gin server, support release,debug,test")

	fs.BoolVar(&o.EnableGrafana, "server-enable-grafana", o.EnableGrafana,
		"Enable grafana will read grafana config file to requests grafana dashboard")

	fs.StringVar(&o.GrafanaConfigFile, "server-grafana-config", o.GrafanaConfigFile,
		"Grafana config file, file contents is grafana config")

	fs.StringVar(&o.StoreType, "server-store", secret.StoreType, "Server storage type, support secret now")

}
