package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gocrane/crane/pkg/controller/noderesource"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
	"github.com/gocrane/crane/pkg/controller/ehpa"
	"github.com/gocrane/crane/pkg/controller/recommendation"
	"github.com/gocrane/crane/pkg/controller/tsp"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/log"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	"github.com/gocrane/crane/pkg/prediction/percentile"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/providers/mock"
	"github.com/gocrane/crane/pkg/providers/prom"
	webhooks "github.com/gocrane/crane/pkg/webhooks"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(autoscalingapi.AddToScheme(scheme))
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
				log.Logger().Error(err, "opts complete failed,exit")
				os.Exit(255)
			}
			if err := opts.Validate(); err != nil {
				log.Logger().Error(err, "opts validate failed,exit")
				os.Exit(255)
			}

			if err := Run(ctx, opts); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
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
		log.Logger().Error(err, "unable to start crane manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		log.Logger().Error(err, "failed to add health check endpoint")
		return err
	}
	if opts.WebhookConfig.Enabled {
		initializationWebhooks(mgr, opts)
	}
	initializationControllers(ctx, mgr, opts)
	log.Logger().Info("Starting crane manager")

	if err := mgr.Start(ctx); err != nil {
		log.Logger().Error(err, "problem running crane manager")
		return err
	}

	return nil
}

func initializationWebhooks(mgr ctrl.Manager, opts *options.Options) {
	log.Logger().Info(fmt.Sprintf("opts %v", opts))

	if certDir := os.Getenv("WEBHOOK_CERT_DIR"); len(certDir) > 0 {
		mgr.GetWebhookServer().CertDir = certDir
	}

	if err := webhooks.SetupWebhookWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create webhook", "webhook", "TimeSeriesPrediction")
		os.Exit(1)
	}
}

// initializationControllers setup controllers with manager
func initializationControllers(ctx context.Context, mgr ctrl.Manager, opts *options.Options) {
	log.Logger().Info(fmt.Sprintf("opts %v", opts))

	discoveryClientSet, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		log.Logger().Error(err, "unable to create discover client")
		os.Exit(1)
	}

	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discoveryClientSet)
	scaleClient := scale.New(
		discoveryClientSet.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scaleKindResolver,
	)

	if err := (&ehpa.EffectiveHPAController{
		Client:      mgr.GetClient(),
		Log:         log.Logger().WithName("effective-hpa-controller"),
		Scheme:      mgr.GetScheme(),
		RestMapper:  mgr.GetRESTMapper(),
		Recorder:    mgr.GetEventRecorderFor("effective-hpa-controller"),
		ScaleClient: scaleClient,
	}).SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "EffectiveHPAController")
		os.Exit(1)
	}

	if err := (&ehpa.SubstituteController{
		Client:      mgr.GetClient(),
		Log:         log.Logger().WithName("substitute-controller"),
		Scheme:      mgr.GetScheme(),
		RestMapper:  mgr.GetRESTMapper(),
		Recorder:    mgr.GetEventRecorderFor("substitute-controller"),
		ScaleClient: scaleClient,
	}).SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "SubstituteController")
		os.Exit(1)
	}

	if err := (&ehpa.HPAReplicasController{
		Client:     mgr.GetClient(),
		Log:        log.Logger().WithName("hpa-replicas-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   mgr.GetEventRecorderFor("hpareplicas-controller"),
	}).SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "HPAReplicasController")
		os.Exit(1)
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
		log.Logger().Error(err, "unable to create controller", "controller", "TspController")
		os.Exit(1)
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
		log.Logger().Error(err, "unable to create controller", "controller", "TspController")
		os.Exit(1)
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
	// register as prometheus metric collector
	tspController.RegisterMetric()
	if err := tspController.SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "TspController")
		os.Exit(1)
	}

	if err := (&analytics.Controller{
		Client:     mgr.GetClient(),
		Logger:     log.Logger().WithName("analytics-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   mgr.GetEventRecorderFor("analytics-controller"),
	}).SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "AnalyticsController")
		os.Exit(1)
	}

	if err := (&recommendation.Controller{
		Client:      mgr.GetClient(),
		Log:         log.Logger().WithName("recommendation-controller"),
		Scheme:      mgr.GetScheme(),
		RestMapper:  mgr.GetRESTMapper(),
		Recorder:    mgr.GetEventRecorderFor("recommendation-controller"),
		ScaleClient: scaleClient,
		Predictors:  predictors,
	}).SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "RecommendationController")
		os.Exit(1)
	}
	// NodeResourceController
	if err := (&noderesource.NodeResourceReconciler{
		Client:   mgr.GetClient(),
		Log:      log.Logger().WithName("node-resource-controller"),
		Recorder: mgr.GetEventRecorderFor("node-resource-controller"),
	}).SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "NodeResourceController")
		os.Exit(1)
	}

}
