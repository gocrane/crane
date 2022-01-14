package options

import (
	"fmt"

	serverconfig "github.com/gocrane/crane/pkg/server/config"
	"github.com/spf13/pflag"
)

// InsecureServingOptions are for creating an unauthenticated, unauthorized, insecure port.
type InsecureServingOptions struct {
	BindAddress string `json:"bind-address"`
	BindPort    int    `json:"bind-port"`
}

// NewInsecureServingOptions is for creating an unauthenticated, unauthorized, insecure port.
func NewInsecureServingOptions() *InsecureServingOptions {
	return &InsecureServingOptions{
		BindAddress: "127.0.0.1",
		BindPort:    8080,
	}

}

func (o *InsecureServingOptions) Complete() error {
	return nil
}

func (o *InsecureServingOptions) ApplyTo(cfg *serverconfig.Config) error {
	cfg.BindAddress = o.BindAddress
	cfg.BindPort = o.BindPort
	return nil
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (s *InsecureServingOptions) Validate() []error {
	var errors []error

	if s.BindPort < 0 || s.BindPort > 65535 {
		errors = append(
			errors,
			fmt.Errorf(
				"--insecure.bind-port %v must be between 0 and 65535, inclusive. 0 for turning off insecure (HTTP) port",
				s.BindPort,
			),
		)
	}

	return errors
}

// AddFlags adds flags related to features for a specific api server to the
// specified FlagSet.
func (s *InsecureServingOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "insecure.bind-address", s.BindAddress, ""+
		"The IP address on which to serve the --insecure.bind-port "+
		"(set to 0.0.0.0 for all IPv4 interfaces and :: for all IPv6 interfaces).")
	fs.IntVar(&s.BindPort, "insecure.bind-port", s.BindPort, ""+
		"The port on which to serve unsecured, unauthenticated access. It is assumed "+
		"that firewall rules are set up such that this port is not reachable from outside of "+
		"the deployed machine and that port 443 on the iam public address is proxied to this "+
		"port. This is performed by nginx in the default setup. Set to zero to disable.")
}
