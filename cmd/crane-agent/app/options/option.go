package options

import (
	"github.com/spf13/pflag"
)

// Options hold the command-line options about crane manager
type Options struct {
	// MetricsAddr is the address the metric endpoint binds to.
	MetricsAddr string
	// BindAddr is the address the probe endpoint binds to.
	BindAddr string
	// WebhookHost is the address webhook binds to.
	WebhookHost string
	// WebhookPort is the port webhook binds to.
	WebhookPort uint64
	// HostnameOverride is the name of k8s node
	HostnameOverride string
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
	flags.StringVar(&o.WebhookHost, "webhook-host", "0.0.0.0", "The address webhook binds to.")
	flags.Uint64Var(&o.WebhookPort, "webhook-port", 9443, "The port webhook binds to.")
	flags.StringVar(&o.HostnameOverride, "hostname-override", o.HostnameOverride, "which is the name of k8s node be used to filtered.")
}
