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
	ensurancecontroller "github.com/gocrane/crane/pkg/controller/ensurance"
	"github.com/gocrane/crane/pkg/ensurance/analyzer"
	"github.com/gocrane/crane/pkg/ensurance/avoidance"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	einformer "github.com/gocrane/crane/pkg/ensurance/informer"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/ensurance/nep"
	"github.com/gocrane/crane/pkg/ensurance/statestore"
	"github.com/gocrane/crane/pkg/utils/clogs"
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
				clogs.Log().Error(err, "opts complete failed,exit")
				os.Exit(255)
			}
			if err := opts.Validate(); err != nil {
				clogs.Log().Error(err, "opts validate failed,exit")
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
		clogs.Log().Error(err, "unable to start crane agent")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		clogs.Log().Error(err, "failed to add health check endpoint")
		return err
	}

	clogs.Log().V(2).Info(fmt.Sprintf("opts %v", opts))

	// init context
	ec := initializationContext(mgr, opts)
	ec.Run()

	// init components
	components := initializationComponents(mgr, opts, ec)

	// start managers
	for _, v := range components {
		clogs.Log().V(2).Info("Starting manager %s", v.Name())
		v.Run(ec.GetStopChannel())
	}

	clogs.Log().V(2).Info("Starting crane agent")
	if err := mgr.Start(ctx); err != nil {
		clogs.Log().Error(err, "problem running crane manager")
		return err
	}

	return nil
}

func initializationComponents(mgr ctrl.Manager, opts *options.Options, ec *einformer.Context) []manager.Manager {
	clogs.Log().V(2).Info(fmt.Sprintf("initializationComponents"))

	var managers []manager.Manager
	podInformer := ec.GetPodFactory().Core().V1().Pods().Informer()
	nodeInformer := ec.GetNodeFactory().Core().V1().Nodes().Informer()
	nepInformer := ec.GetAvoidanceFactory().Ensurance().V1alpha1().NodeQOSEnsurancePolicies().Informer()
	avoidanceInformer := ec.GetAvoidanceFactory().Ensurance().V1alpha1().AvoidanceActions().Informer()

	// init state store manager
	stateStoreManager := statestore.NewStateStoreManager()
	managers = append(managers, stateStoreManager)

	// init analyzer manager
	analyzerManager := analyzer.NewAnalyzerManager(podInformer, nodeInformer, avoidanceInformer, nepInformer, noticeCh)
	managers = append(managers, analyzerManager)

	// init avoidance manager
	var noticeCh = make(chan executor.AvoidanceExecutorStruct)
	avoidanceManager := avoidance.NewAvoidanceManager(ec.GetKubeClient(), opts.HostnameOverride, podInformer, nodeInformer, avoidanceInformer, noticeCh)
	managers = append(managers, avoidanceManager)

	// init nep controller
	nepRecorder := mgr.GetEventRecorderFor("node-qos-controller")
	if err := (&ensurancecontroller.NodeQOSEnsurancePolicyController{
		Client:     mgr.GetClient(),
		Log:        clogs.Log().WithName("node-qos-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   nepRecorder,
		Cache:      &nep.NodeQOSEnsurancePolicyCache{},
		StateStore: stateStoreManager,
	}).SetupWithManager(mgr); err != nil {
		clogs.Log().Error(err, "unable to create controller", "controller", "NodeQOSEnsurancePolicyController")
		os.Exit(1)
	}

	return managers
}

func initializationContext(mgr ctrl.Manager, opts *options.Options) *einformer.Context {
	clogs.Log().V(2).Info(fmt.Sprintf("initializationContext"))

	generatedClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	clientSet := ensuaranceset.NewForConfigOrDie(mgr.GetConfig())

	return einformer.NewContextInitWithClient(generatedClient, clientSet, opts.HostnameOverride)
}
