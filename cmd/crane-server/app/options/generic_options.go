package options

import (
	"github.com/spf13/pflag"

	serverconfig "github.com/gocrane/crane/pkg/server/config"
)

// GenericOptions
type GenericOptions struct {
	KubeConfig string `json:"kubeconfig"`
}

// NewGenericOptions is for creating an unauthenticated, unauthorized, insecure port.
func NewGenericOptions() *GenericOptions {
	return &GenericOptions{}

}

func (o *GenericOptions) Complete() error {
	return nil
}

func (o *GenericOptions) ApplyTo(cfg *serverconfig.Config) error {
	cfg.KubeConfig = o.KubeConfig
	return nil
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (s *GenericOptions) Validate() []error {
	var errors []error

	return errors
}

// AddFlags adds flags related to features for a specific api server to the
// specified FlagSet.
func (s *GenericOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.KubeConfig, "generic.kubeconfig", s.KubeConfig, "kubernetes config file")
}
