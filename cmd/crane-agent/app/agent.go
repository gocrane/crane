package app

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gocrane/crane/pkg/utils"
	"k8s.io/client-go/rest"

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
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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

	// use NODE_NAME as the default value of HostnameOverride
	if os.Getenv("NODE_NAME") != "" {
		opts.HostnameOverride = os.Getenv("NODE_NAME")
	}

	if opts.HostnameOverride == "" {
		log.Logger().Error(nil, "HostnameOverride must be set as the k8s node name")
		os.Exit(1)
	}

	log.Logger().V(2).Info(fmt.Sprintf("opts %v", opts))

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Logger().Error(err, "InClusterConfig failed")
	}

	stop := utils.SetupSignalHandler()

	// init context
	ec := initializeContext(config, opts, stop)
	ec.Run()

	// init managers
	components := initializeManagers(opts, ec)

	// start managers
	for _, v := range components {
		log.Logger().V(2).Info(fmt.Sprintf("Starting manager %s", v.Name()))
		v.Run(ec.GetStop())
	}

	return nil
}

func initializeManagers(opts *options.Options, ec *einformer.Context) []manager.Manager {
	log.Logger().V(2).Info(fmt.Sprintf("initializeManagers"))

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
	analyzerManager := analyzer.NewAnalyzerManager(opts.HostnameOverride, podInformer, nodeInformer, nepInformer, avoidanceInformer, stateStoreManager, ec.GetRecorder(), noticeCh)
	managers = append(managers, analyzerManager)

	// init avoidance manager
	avoidanceManager := avoidance.NewAvoidanceManager(ec.GetKubeClient(), opts.HostnameOverride, podInformer, nodeInformer, avoidanceInformer, noticeCh)
	managers = append(managers, avoidanceManager)

	return managers
}

func initializeContext(config *rest.Config, opts *options.Options, stop <-chan struct{}) *einformer.Context {
	log.Logger().V(2).Info(fmt.Sprintf("initializationContext"))

	generatedClient := kubernetes.NewForConfigOrDie(config)
	clientSet := ensuaranceset.NewForConfigOrDie(config)

	return einformer.NewContextWithClient(generatedClient, clientSet, opts.HostnameOverride, stop)
}
