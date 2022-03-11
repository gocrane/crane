package options

import (
	"time"

	"github.com/spf13/pflag"
)

// Options hold the command-line options about crane manager
type Options struct {
	// HostnameOverride is the name of k8s node
	HostnameOverride string
	// RuntimeEndpoint is the endpoint of runtime
	RuntimeEndpoint string
	// Is debug/pprof endpoint enabled
	EnableProfiling bool
	// BindAddr is the address the endpoint binds to.
	BindAddr string
	// CollectInterval is the period for state collector to collect metrics
	CollectInterval time.Duration
	// MaxInactivity is the maximum time from last recorded activity before automatic restart
	MaxInactivity time.Duration
	// Ifaces is the network devices to collect metric
	Ifaces []string
	//NodeResourceOptions is the options of nodeResource
	NodeResourceOptions NodeResourceOptions
	//UseBt is the flag of if use bt_stat
	UseBt bool
}

type NodeResourceOptions struct {
	Enabled                 bool
	CollectorNames          []string
	ReserveCpuPercentStr    string
	ReserveMemoryPercentStr string
}

// NewOptions builds an empty options.
func NewOptions() *Options {
	return &Options{
		NodeResourceOptions: NodeResourceOptions{
			CollectorNames: []string{},
		},
	}
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
	flags.StringVar(&o.HostnameOverride, "hostname-override", "", "Which is the name of k8s node be used to filtered.")
	flags.StringVar(&o.RuntimeEndpoint, "runtime-endpoint", "unix:///var/run/dockershim.sock", "The runtime endpoint, default to docker: unix:///var/run/dockershim.sock, containerd: unix:///run/containerd/containerd.sock.")
	flags.Bool("enable-profiling", false, "Is debug/pprof endpoint enabled, default: false")
	flags.StringVar(&o.BindAddr, "bind-address", "0.0.0.0:8081", "The address the agent binds to,default: 0.0.0.0:8081.")
	flags.DurationVar(&o.CollectInterval, "collect-interval", 10*time.Second, "period for the state collector to collect metrics, default: 10s")
	flags.StringArrayVar(&o.Ifaces, "ifaces", []string{"eth0"}, "The network devices to collect metric, use comma to separated, default: eth0")
	flags.DurationVar(&o.MaxInactivity, "max-inactivity", 5*time.Minute, "Maximum time from last recorded activity before automatic restart, default: 5min")
	flags.StringSliceVar(&o.NodeResourceOptions.CollectorNames, "noderesource-collector-names", []string{}, "The collectors of noderesource.")
	flags.StringVar(&o.NodeResourceOptions.ReserveCpuPercentStr, "reserve-cpu-percent", "", "reserve cpu percentage of node.")
	flags.StringVar(&o.NodeResourceOptions.ReserveMemoryPercentStr, "reserve-memory-percent", "", "reserve memory percentage of node.")
	flags.BoolVar(&o.NodeResourceOptions.Enabled, "noderesource-enabled", false, "Enable NodeResourceManager.")
	flags.BoolVar(&o.UseBt, "bt-enabled", false, "Enable bt scheduled.")
}
