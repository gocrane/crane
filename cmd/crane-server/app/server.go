package app

import (
	"context"
	"flag"

	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/cmd/crane-server/app/options"
	"github.com/gocrane/crane/pkg/server"
	serverconfig "github.com/gocrane/crane/pkg/server/config"
)

// NewServerCommand creates a *cobra.Command object with default parameters
func NewServerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "crane-server",
		Long: `The crane server is the dashboard backend for crane`,
		Run: func(cmd *cobra.Command, args []string) {
			printFlags(cmd.Flags())
			if err := opts.Complete(); err != nil {
				klog.Exitf("Opts complete failed: %v", err)
			}
			if err := opts.Validate(); err != nil {
				klog.Exitf("Opts validate failed: %v", err)
			}
			cfg := serverconfig.NewServerConfig()
			if err := opts.ApplyTo(cfg); err != nil {
				klog.Exitf("Opts apply failed: %v", err)
			}
			klog.Infof("cfg: %+v", cfg)
			server, err := server.NewAPIServer(cfg)
			if err != nil {
				klog.Exitf("New server failed: %v", err)
			}
			server.Run(ctx)
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())

	return cmd
}
func printFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		klog.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
