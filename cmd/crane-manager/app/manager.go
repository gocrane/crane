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
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/cmd/crane-manager/app/options"
	"github.com/gocrane/crane/pkg/controller/hpa"
	"github.com/gocrane/crane/pkg/known"
)

var (
	scheme        = runtime.NewScheme()
	managerLogger = ctrl.Log.WithName("crane-manager")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(autoscalingapi.AddToScheme(scheme))
	utilruntime.Must(predictionapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

}

// NewManagerCommand creates a *cobra.Command object with default parameters
func NewManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "crane-manager",
		Long: `The crane manager is responsible for manage controllers in crane`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				klog.Exit(err)
			}
			if err := opts.Validate(); err != nil {
				klog.Exit(err)
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

// Run runs the crane-manager with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	logger := klogr.New().WithName("crane-manager")
	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      opts.MetricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  opts.BindAddr,
		LeaderElection:          opts.LeaderElection.LeaderElect,
		LeaderElectionID:        "crane-manager",
		LeaderElectionNamespace: known.CraneSystemNamespace,
	})
	if err != nil {
		managerLogger.Error(err, "unable to start crane manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Errorf("failed to add health check endpoint: %v", err)
		return err
	}

	initializationControllers(mgr, opts)

	managerLogger.Info("Starting crane manager")
	if err := mgr.Start(ctx); err != nil {
		klog.Errorf("problem running crane manager: %v", err)
		return err
	}

	return nil
}

// initializationControllers setup controllers with manager
func initializationControllers(mgr ctrl.Manager, opts *options.Options) {
	managerLogger.Info(fmt.Sprintf("opts %v", opts))
	hpaRecorder := mgr.GetEventRecorderFor("advanced-hpa-controller")
	if err := (&hpa.AdvancedHPAController{
		Client:     mgr.GetClient(),
		Log:        mgr.GetLogger().WithName("advanced-hpa-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   hpaRecorder,
	}).SetupWithManager(mgr); err != nil {
		managerLogger.Error(err, "unable to create controller", "controller", "AdvancedHPAController")
		os.Exit(1)
	}

}
