package app

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	ensuaranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	ensuaranceset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/crane/cmd/crane-agent/app/options"
	"github.com/gocrane/crane/pkg/ensurance/analyzer"
	"github.com/gocrane/crane/pkg/ensurance/avoidance"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	einformer "github.com/gocrane/crane/pkg/ensurance/informer"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/ensurance/statestore"
	"github.com/gocrane/crane/pkg/utils/log"
)

var (
	scheme = runtime.NewScheme()
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
				log.Logger().Error(err, "opts complete failed,exit")
				os.Exit(255)
			}
			if err := opts.Validate(); err != nil {
				log.Logger().Error(err, "opts validate failed,exit")
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
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     opts.MetricsAddr,
		HealthProbeBindAddress: opts.BindAddr,
		Port:                   int(opts.WebhookPort),
		Host:                   opts.WebhookHost,
		LeaderElection:         false,
	})
	if err != nil {
		log.Logger().Error(err, "Unable to start crane agent")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		log.Logger().Error(err, "Failed to add health check endpoint")
		return err
	}

	if opts.HostnameOverride == "" {
		log.Logger().Error(err, "HostnameOverride must be set as the k8s node name")
		os.Exit(1)
	}

	log.Logger().V(2).Info(fmt.Sprintf("opts %v", opts))

	// init context
	ec := initializationContext(mgr, opts)
	ec.Run()

	// init components
	components := initializationComponents(mgr, opts, ec)

	// start managers
	for _, v := range components {
		log.Logger().V(2).Info(fmt.Sprintf("Starting manager %s", v.Name()))
		v.Run(ec.GetStopChannel())
	}

	log.Logger().V(2).Info("Starting crane agent")
	if err := mgr.Start(ctx); err != nil {
		log.Logger().Error(err, "problem running crane manager")
		return err
	}

	return nil
}

func initializationComponents(mgr ctrl.Manager, opts *options.Options, ec *einformer.Context) []manager.Manager {
	log.Logger().V(2).Info(fmt.Sprintf("initializationComponents"))

	var managers []manager.Manager
	podInformer := ec.GetPodInformer()
	nodeInformer := ec.GetNodeInformer()
	nepInformer := ec.GetNepInformer()
	avoidanceInformer := ec.GetAvoidanceInformer()

	var noticeCh = make(chan executor.AvoidanceExecutor)

	// init state store manager
	stateStoreManager := statestore.NewStateStoreManager(nepInformer)
	managers = append(managers, stateStoreManager)

	// init analyzer manager
	analyzerRecorder := mgr.GetEventRecorderFor("analyzer")
	analyzerManager := analyzer.NewAnalyzerManager(opts.HostnameOverride, podInformer, nodeInformer, nepInformer, avoidanceInformer, stateStoreManager, analyzerRecorder, noticeCh)
	managers = append(managers, analyzerManager)

	// init avoidance manager
	avoidanceManager := avoidance.NewAvoidanceManager(ec.GetKubeClient(), opts.HostnameOverride, podInformer, nodeInformer, avoidanceInformer, noticeCh)
	managers = append(managers, avoidanceManager)

	return managers
}

func initializationContext(mgr ctrl.Manager, opts *options.Options) *einformer.Context {
	log.Logger().V(2).Info(fmt.Sprintf("initializationContext"))

	generatedClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	clientSet := ensuaranceset.NewForConfigOrDie(mgr.GetConfig())

	return einformer.NewContextInitWithClient(generatedClient, clientSet, opts.HostnameOverride)
}
