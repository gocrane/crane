package options

import (
	"encoding/json"

	"github.com/spf13/pflag"

	serverconfig "github.com/gocrane/crane/pkg/server/config"
)

// Options runs a crane api server.
type Options struct {
	InsecureServing *InsecureServingOptions `json:"insecure"`
	FeatureOptions  *FeatureOptions         `json:"feature"`
	GrafanaOptions  *GrafanaOptions         `json:"grafana"`
	GenericOptions  *GenericOptions         `json:"generic"`
}

// NewOptions creates a new Options object with default parameters.
func NewOptions() *Options {
	o := Options{
		InsecureServing: NewInsecureServingOptions(),
		FeatureOptions:  NewFeatureOptions(),
		GrafanaOptions:  NewGrafanaOptions(),
		GenericOptions:  NewGenericOptions(),
	}

	return &o
}

// AddFlags adds flags related to features for a specific api server to the
// specified FlagSet.
func (o Options) AddFlags(fs *pflag.FlagSet) {
	o.FeatureOptions.AddFlags(fs)
	o.InsecureServing.AddFlags(fs)
	o.GrafanaOptions.AddFlags(fs)
	o.GenericOptions.AddFlags(fs)
}

// String return the json string of options
func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}

func (o *Options) ApplyTo(cfg *serverconfig.Config) error {
	_ = o.FeatureOptions.ApplyTo(cfg)
	_ = o.InsecureServing.ApplyTo(cfg)
	_ = o.GrafanaOptions.ApplyTo(cfg)
	_ = o.GenericOptions.ApplyTo(cfg)
	return nil
}

func (o *Options) Validate() error {
	return nil
}

// Complete set default Options.
func (o *Options) Complete() error {
	if o.FeatureOptions.EnableGrafana {
		return o.GrafanaOptions.Complete()
	} else {
		return nil
	}
}
