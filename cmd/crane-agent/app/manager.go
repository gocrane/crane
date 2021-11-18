package app

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/component-base/logs"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	ensuaranceapi "github.com/gocrane-io/api/ensurance/v1alpha1"
	"github.com/gocrane-io/crane/cmd/crane-agent/app/options"
	ensurancecontroller "github.com/gocrane-io/crane/pkg/controller/ensurance"
)

var (
	scheme        = runtime.NewScheme()
	managerLogger = ctrl.Log.WithName("crane-agent")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ensuaranceapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// NewManagerCommand creates a *cobra.Command object with default parameters
func NewManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "crane-agent",
		Long: `The crane agent is responsible agent in crane`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				managerLogger.Error(err, "opts complete failed,exit")
				os.Exit(255)
			}
			if err := opts.Validate(); err != nil {
				managerLogger.Error(err, "opts validate failed,exit")
				os.Exit(255)
			}

			if err := Run(ctx, opts); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the crane-agent with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	logs.InitLogs()
	defer logs.FlushLogs()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     opts.MetricsAddr,
		HealthProbeBindAddress: opts.BindAddr,
		Port:                   int(opts.WebhookPort),
		Host:                   opts.WebhookHost,
	})
	if err != nil {
		managerLogger.Error(err, "unable to start crane agent")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		managerLogger.Error(err, "failed to add health check endpoint")
		return err
	}

	initializationControllers(mgr, opts)

	managerLogger.Info("Starting crane agent")
	if err := mgr.Start(ctx); err != nil {
		managerLogger.Error(err, "problem running crane manager")
		return err
	}

	return nil
}

// initializationControllers setup controllers with manager
func initializationControllers(mgr ctrl.Manager, opts *options.Options) {
	managerLogger.Info(fmt.Sprintf("opts %v", opts))
	nepRecorder := mgr.GetEventRecorderFor("node-qos-ensurance-policy-controller")
	if err := (&ensurancecontroller.NodeQOSEnsurancePolicyController{
		Client:     mgr.GetClient(),
		Log:        mgr.GetLogger().WithName("node-qos-ensurance-policy-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   nepRecorder,
	}).SetupWithManager(mgr); err != nil {
		managerLogger.Error(err, "unable to create controller", "controller", "NodeQOSEnsurancePolicyController")
		os.Exit(1)
	}

}
