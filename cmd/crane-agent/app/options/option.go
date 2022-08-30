package options

import (
	"time"

	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
)

// Options hold the command-line options about crane manager
type Options struct {
	// HostnameOverride is the name of k8s node
	HostnameOverride string
	// RuntimeEndpoint is the endpoint of runtime
	RuntimeEndpoint string
	// driver that the kubelet uses to manipulate cgroups on the host (cgroupfs or systemd)
	CgroupDriver string
	// Is debug/pprof endpoint enabled
	EnableProfiling bool
	// BindAddr is the address the endpoint binds to.
	BindAddr string
	// CollectInterval is the period for state collector to collect metrics
	CollectInterval time.Duration
	// MaxInactivity is the maximum time from last recorded activity before automatic restart
	MaxInactivity time.Duration
	// Ifaces is the network devices to collect metric
	Ifaces               []string
	NodeResourceReserved map[string]string
	// ExecuteExcess is the percentage of executions that exceed the gap between current usage and watermarks
	ExecuteExcess string
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
	flags.StringVar(&o.HostnameOverride, "hostname-override", "", "Which is the name of k8s node be used to filtered.")
	flags.StringVar(&o.RuntimeEndpoint, "runtime-endpoint", "", "The runtime endpoint docker: unix:///var/run/dockershim.sock, containerd: unix:///run/containerd/containerd.sock, cri-o: unix:///run/crio/crio.sock, k3s: unix:///run/k3s/containerd/containerd.sock.")
	flags.StringVar(&o.CgroupDriver, "cgroup-driver", "cgroupfs", "Driver that the kubelet uses to manipulate cgroups on the host.  Possible values: 'cgroupfs', 'systemd'. Default to 'cgroupfs'")
	flags.Bool("enable-profiling", false, "Is debug/pprof endpoint enabled, default: false")
	flags.StringVar(&o.BindAddr, "bind-address", "0.0.0.0:8081", "The address the agent binds to for metrics, health-check and pprof, default: 0.0.0.0:8081.")
	flags.DurationVar(&o.CollectInterval, "collect-interval", 10*time.Second, "Period for the state collector to collect metrics, default: 10s")
	flags.StringArrayVar(&o.Ifaces, "ifaces", []string{"eth0"}, "The network devices to collect metric, use comma to separated, default: eth0")
	flags.Var(cliflag.NewMapStringString(&o.NodeResourceReserved), "node-resource-reserved", "A set of ResourceName=Percent (e.g. cpu=40%,memory=40%)")
	flags.DurationVar(&o.MaxInactivity, "max-inactivity", 5*time.Minute, "Maximum time from last recorded activity before automatic restart, default: 5min")
	flags.StringVar(&o.ExecuteExcess, "execute-excess", "10%", "The percentage of executions that exceed the gap between current usage and watermarks, default: 10%.")
}
