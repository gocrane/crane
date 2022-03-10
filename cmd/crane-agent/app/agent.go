package app

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	craneinformers "github.com/gocrane/api/pkg/generated/informers/externalversions"

	"github.com/gocrane/crane/cmd/crane-agent/app/options"
	"github.com/gocrane/crane/pkg/agent"
	"github.com/gocrane/crane/pkg/metrics"
)

var (
	scheme = runtime.NewScheme()
)

const (
	nodeNameField      = "metadata.name"
	specNodeNameField  = "spec.nodeName"
	informerSyncPeriod = time.Minute
	DefaultWorkers     = 2
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ensuranceapi.AddToScheme(scheme))
}

// NewAgentCommand creates a *cobra.Command object with default parameters
func NewAgentCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "crane-agent",
		Long: `The crane agent is running in each node and responsible for QoS ensurance`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				klog.Exitf("Opts complete failed: %v", err)
			}
			if err := opts.Validate(); err != nil {
				klog.Exitf("Opts validate failed: %v", err)
			}
			if err := Run(ctx, opts); err != nil {
				klog.Exit(err)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
	return cmd
}

func Run(ctx context.Context, opts *options.Options) error {
	hostname := getHostName(opts.HostnameOverride)

	healthCheck := metrics.NewHealthCheck(opts.MaxInactivity)
	metrics.RegisterCraneAgent()

	kubeClient, craneClient, err := buildClient()
	if err != nil {
		return err
	}

	podInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, informerSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector(specNodeNameField, hostname).String()
		}),
	)

	nodeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, informerSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector(nodeNameField, hostname).String()
		}),
	)
	podInformer := podInformerFactory.Core().V1().Pods()
	nodeInformer := nodeInformerFactory.Core().V1().Nodes()
	podInformer.Informer()
	nodeInformer.Informer()

	craneInformerFactory := craneinformers.NewSharedInformerFactory(craneClient, informerSyncPeriod)
	nepInformer := craneInformerFactory.Ensurance().V1alpha1().NodeQOSEnsurancePolicies()
	actionInformer := craneInformerFactory.Ensurance().V1alpha1().AvoidanceActions()
	tspInformer := craneInformerFactory.Prediction().V1alpha1().TimeSeriesPredictions()

	nepInformer.Informer()
	actionInformer.Informer()
	tspInformer.Informer()

	agent, err := agent.NewAgent(ctx, hostname, opts.RuntimeEndpoint, kubeClient, craneClient,
		podInformer, nodeInformer, nepInformer, actionInformer, tspInformer, opts.Ifaces, healthCheck, opts.CollectInterval)

	if err != nil {
		return err
	}

	podInformerFactory.Start(ctx.Done())
	nodeInformerFactory.Start(ctx.Done())
	craneInformerFactory.Start(ctx.Done())

	podInformerFactory.WaitForCacheSync(ctx.Done())
	nodeInformerFactory.WaitForCacheSync(ctx.Done())
	craneInformerFactory.WaitForCacheSync(ctx.Done())

	agent.Run(healthCheck, opts)
	return nil
}

func buildClient() (*kubernetes.Clientset, *craneclientset.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("Failed to get InClusterConfig, %v.", err)
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("Failed to new kubernetes client, %v.", err)
		return nil, nil, err
	}
	craneClient, err := craneclientset.NewForConfig(config)
	if err != nil {
		klog.Errorf("Failed to new crane client, %v.", err)
		return nil, nil, err
	}
	return kubeClient, craneClient, nil
}

func getHostName(override string) string {
	nodeName, _ := os.Hostname()
	if os.Getenv("NODE_NAME") != "" {
		nodeName = os.Getenv("NODE_NAME")
	}
	if len(override) != 0 {
		nodeName = override
	}
	return nodeName
}
