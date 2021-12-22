package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gocrane/crane/pkg/recommend"
	"k8s.io/klog/v2"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/scale"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/cmd/craned/app/options"
	"github.com/gocrane/crane/pkg/controller/analytics"
	"github.com/gocrane/crane/pkg/controller/cnp"
	"github.com/gocrane/crane/pkg/controller/ehpa"
	"github.com/gocrane/crane/pkg/controller/noderesource"
	"github.com/gocrane/crane/pkg/controller/recommendation"
	"github.com/gocrane/crane/pkg/controller/tsp"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/log"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	"github.com/gocrane/crane/pkg/prediction/percentile"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/providers/mock"
	"github.com/gocrane/crane/pkg/providers/prom"
	"github.com/gocrane/crane/pkg/webhooks"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(predictionapi.AddToScheme(scheme))
	utilruntime.Must(analysisapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

}

// NewManagerCommand creates a *cobra.Command object with default parameters
func NewManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "craned",
		Long: `The crane manager is responsible for manage controllers in crane`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				klog.Exit(err)
			}
			if err := opts.Validate(); err != nil {
				klog.Exit(err)
			}
			if err := Run(ctx, opts); err != nil {
				klog.Exit(err)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
	utilfeature.DefaultMutableFeatureGate.AddFlag(cmd.Flags())

	return cmd
}

// Run runs the craned with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      opts.MetricsAddr,
		Port:                    9443,
		HealthProbeBindAddress:  opts.BindAddr,
		LeaderElection:          opts.LeaderElection.LeaderElect,
		LeaderElectionID:        "craned",
		LeaderElectionNamespace: known.CraneSystemNamespace,
	})
	if err != nil {
		klog.Error(err, "unable to start crane manager")
		return err
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Error(err, "failed to add health check endpoint")
		return err
	}

	if opts.WebhookConfig.Enabled {
		initializationWebhooks(mgr, opts)
	}
	initializationControllers(ctx, mgr, opts)
	klog.Info("Starting crane manager")

	// initialization custom collector metrics
	initializationMetricCollector(mgr)

	if err := mgr.Start(ctx); err != nil {
		klog.Error(err, "problem running crane manager")
		return err
	}

	return nil
}

func initializationMetricCollector(mgr ctrl.Manager) {
	// register as prometheus metric collector
	metrics.CustomCollectorRegister(metrics.NewTspMetricCollector(mgr.GetClient()))
}

func initializationWebhooks(mgr ctrl.Manager, opts *options.Options) {
	log.Logger().Info(fmt.Sprintf("opts %v", opts))

	if certDir := os.Getenv("WEBHOOK_CERT_DIR"); len(certDir) > 0 {
		mgr.GetWebhookServer().CertDir = certDir
	}

	if err := webhooks.SetupWebhookWithManager(mgr); err != nil {
		klog.Exit(err, "unable to create webhook", "webhook", "TimeSeriesPrediction")
	}
}

// initializationControllers setup controllers with manager
func initializationControllers(ctx context.Context, mgr ctrl.Manager, opts *options.Options) {
	autoscaling := utilfeature.DefaultFeatureGate.Enabled(features.CraneAutoscaling)
	nodeResource := utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource)
	clusterNodePrediction := utilfeature.DefaultFeatureGate.Enabled(features.CraneClusterNodePrediction)
	// todo: add more features

	discoveryClientSet, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		klog.Exit(err, "Unable to create discover client")
	}

	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discoveryClientSet)
	scaleClient := scale.New(
		discoveryClientSet.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scaleKindResolver,
	)

	if autoscaling {
		utilruntime.Must(autoscalingapi.AddToScheme(scheme))
		if err := (&ehpa.EffectiveHPAController{
			Client:      mgr.GetClient(),
			Scheme:      mgr.GetScheme(),
			RestMapper:  mgr.GetRESTMapper(),
			Recorder:    mgr.GetEventRecorderFor("effective-hpa-controller"),
			ScaleClient: scaleClient,
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "EffectiveHPAController")
		}

		if err := (&ehpa.SubstituteController{
			Client:      mgr.GetClient(),
			Scheme:      mgr.GetScheme(),
			RestMapper:  mgr.GetRESTMapper(),
			Recorder:    mgr.GetEventRecorderFor("substitute-controller"),
			ScaleClient: scaleClient,
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "SubstituteController")
		}

		if err := (&ehpa.HPAReplicasController{
			Client:     mgr.GetClient(),
			Scheme:     mgr.GetScheme(),
			RestMapper: mgr.GetRESTMapper(),
			Recorder:   mgr.GetEventRecorderFor("hpareplicas-controller"),
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "HPAReplicasController")
		}
	}

	// TspController
	var dataSource providers.Interface
	switch strings.ToLower(opts.DataSource) {
	case "prometheus", "prom":
		dataSource, err = prom.NewProvider(&opts.DataSourcePromConfig)
	case "mock":
		dataSource, err = mock.NewProvider(&opts.DataSourceMockConfig)
	default:
		// default is prom
		dataSource, err = prom.NewProvider(&opts.DataSourcePromConfig)
	}
	if err != nil {
		klog.Exit(err, "unable to create controller", "controller", "TspController")
	}

	// algorithm provider inject data source
	percentilePredictor := percentile.NewPrediction()
	percentilePredictor.WithProviders(map[string]providers.Interface{
		prediction.RealtimeProvider: dataSource,
		prediction.HistoryProvider:  dataSource,
	})
	go percentilePredictor.Run(ctx.Done())

	dspPredictor, err := dsp.NewPrediction(opts.AlgorithmModelConfig)
	if err != nil {
		klog.Exit(err, "unable to create controller", "controller", "TspController")
	}
	dspPredictor.WithProviders(map[string]providers.Interface{
		prediction.RealtimeProvider: dataSource,
		prediction.HistoryProvider:  dataSource,
	})
	go dspPredictor.Run(ctx.Done())

	predictors := map[predictionapi.AlgorithmType]prediction.Interface{
		predictionapi.AlgorithmTypePercentile: percentilePredictor,
		predictionapi.AlgorithmTypeDSP:        dspPredictor,
	}

	tspController := tsp.NewController(
		mgr.GetClient(),
		log.Logger().WithName("time-series-prediction-controller"),
		mgr.GetEventRecorderFor("time-series-prediction-controller"),
		opts.PredictionUpdateFrequency,
		predictors,
	)
	if err := tspController.SetupWithManager(mgr); err != nil {
		klog.Exit(err, "unable to create controller", "controller", "TspController")
	}

	if err := (&analytics.Controller{
		Client:     mgr.GetClient(),
		Logger:     log.Logger().WithName("analytics-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   mgr.GetEventRecorderFor("analytics-controller"),
	}).SetupWithManager(mgr); err != nil {
		klog.Exit(err, "unable to create controller", "controller", "AnalyticsController")
	}

	configSet, err := recommend.LoadConfigSetFromFile(opts.RecommendationConfigFile)
	if err != nil {
		klog.Errorf("Failed to load recommendation config file: %v", err)
		os.Exit(1)
	}
	if err := (&recommendation.Controller{
		Client:      mgr.GetClient(),
		ConfigSet:   configSet,
		Scheme:      mgr.GetScheme(),
		RestMapper:  mgr.GetRESTMapper(),
		Recorder:    mgr.GetEventRecorderFor("recommendation-controller"),
		ScaleClient: scaleClient,
		Predictors:  predictors,
		Provider:    dataSource,
	}).SetupWithManager(mgr); err != nil {
		klog.Exit(err, "unable to create controller", "controller", "RecommendationController")
	}

	// NodeResourceController
	if nodeResource {
		if err := (&noderesource.NodeResourceReconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor("node-resource-controller"),
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "NodeResourceController")
		}
	}

	// CnpController
	if clusterNodePrediction {
		if err := (&cnp.ClusterNodePredictionController{
			Client:     mgr.GetClient(),
			Scheme:     mgr.GetScheme(),
			RestMapper: mgr.GetRESTMapper(),
			Recorder:   mgr.GetEventRecorderFor("cnp-controller"),
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "CnpController")
		}
	}
}
