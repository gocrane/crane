package app

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/cmd/craned/app/options"
	"github.com/gocrane/crane/pkg/controller/analytics"
	"github.com/gocrane/crane/pkg/controller/cnp"
	"github.com/gocrane/crane/pkg/controller/ehpa"
	"github.com/gocrane/crane/pkg/controller/evpa"
	"github.com/gocrane/crane/pkg/controller/recommendation"
	"github.com/gocrane/crane/pkg/controller/timeseriesprediction"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/oom"
	"github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/providers/grpc"
	"github.com/gocrane/crane/pkg/providers/metricserver"
	"github.com/gocrane/crane/pkg/providers/mock"
	"github.com/gocrane/crane/pkg/providers/prom"
	_ "github.com/gocrane/crane/pkg/querybuilder-providers/grpc"
	_ "github.com/gocrane/crane/pkg/querybuilder-providers/metricserver"
	_ "github.com/gocrane/crane/pkg/querybuilder-providers/prometheus"
	"github.com/gocrane/crane/pkg/server"
	serverconfig "github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/utils/target"
	"github.com/gocrane/crane/pkg/webhooks"
)

var (
	scheme = runtime.NewScheme()
)

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
			if errs := opts.Validate(); len(errs) != 0 {
				klog.Exit(errs)
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
	config := ctrl.GetConfigOrDie()
	config.QPS = float32(opts.ApiQps)
	config.Burst = opts.ApiBurst

	mgr, err := ctrl.NewManager(config, ctrl.Options{
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
	// initialize data sources and predictor
	realtimeDataSources, histroyDataSources, _ := initDataSources(mgr, opts)
	predictorMgr := initPredictorManager(opts, realtimeDataSources, histroyDataSources)

	initScheme()
	initWebhooks(mgr, opts)
	initControllers(ctx, mgr, opts, predictorMgr, histroyDataSources[providers.PrometheusDataSource])
	// initialize custom collector metrics
	initMetricCollector(mgr)
	runAll(ctx, mgr, predictorMgr, opts)

	return nil
}

func initScheme() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	if utilfeature.DefaultFeatureGate.Enabled(features.CraneAutoscaling) {
		utilruntime.Must(autoscalingapi.AddToScheme(scheme))
	}
	if utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource) || utilfeature.DefaultFeatureGate.Enabled(features.CraneClusterNodePrediction) {
		utilruntime.Must(ensuranceapi.AddToScheme(scheme))
	}
	if utilfeature.DefaultMutableFeatureGate.Enabled(features.CraneAnalysis) {
		utilruntime.Must(analysisapi.AddToScheme(scheme))
	}
	if utilfeature.DefaultFeatureGate.Enabled(features.CraneTimeSeriesPrediction) {
		utilruntime.Must(predictionapi.AddToScheme(scheme))
	}
}

func initMetricCollector(mgr ctrl.Manager) {
	// register as prometheus metric collector
	metrics.CustomCollectorRegister(metrics.NewTspMetricCollector(mgr.GetClient()))
}

func initWebhooks(mgr ctrl.Manager, opts *options.Options) {
	if !opts.WebhookConfig.Enabled {
		return
	}

	if certDir := os.Getenv("WEBHOOK_CERT_DIR"); len(certDir) > 0 {
		mgr.GetWebhookServer().CertDir = certDir
	}

	if err := webhooks.SetupWebhookWithManager(mgr,
		utilfeature.DefaultFeatureGate.Enabled(features.CraneAutoscaling),
		utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource),
		utilfeature.DefaultFeatureGate.Enabled(features.CraneClusterNodePrediction),
		utilfeature.DefaultMutableFeatureGate.Enabled(features.CraneAnalysis),
		utilfeature.DefaultFeatureGate.Enabled(features.CraneTimeSeriesPrediction)); err != nil {
		klog.Exit(err, "unable to create webhook", "webhook", "TimeSeriesPrediction")
	}
}

func initDataSources(mgr ctrl.Manager, opts *options.Options) (map[providers.DataSourceType]providers.RealTime, map[providers.DataSourceType]providers.History, map[providers.DataSourceType]providers.Interface) {
	realtimeDataSources := make(map[providers.DataSourceType]providers.RealTime)
	historyDataSources := make(map[providers.DataSourceType]providers.History)
	hybridDataSources := make(map[providers.DataSourceType]providers.Interface)
	for _, datasource := range opts.DataSource {
		switch strings.ToLower(datasource) {
		case "metricserver":
			provider, err := metricserver.NewProvider(mgr.GetConfig())
			if err != nil {
				klog.Exitf("unable to create datasource provider %v, err: %v", datasource, err)
			}
			realtimeDataSources[providers.MetricServerDataSource] = provider
		case "grpc":
			provider := grpc.NewProvider(&opts.DataSourceGrpcConfig)
			historyDataSources[providers.GrpcDataSource] = provider
		case "mock":
			provider, err := mock.NewProvider(&opts.DataSourceMockConfig)
			if err != nil {
				klog.Exitf("unable to create datasource provider %v, err: %v", datasource, err)
			}
			hybridDataSources[providers.MockDataSource] = provider
		case "prometheus", "prom":
			fallthrough
		default:
			// default is prom
			provider, err := prom.NewProvider(&opts.DataSourcePromConfig)
			if err != nil {
				klog.Exitf("unable to create datasource provider %v, err: %v", datasource, err)
			}
			hybridDataSources[providers.PrometheusDataSource] = provider
			realtimeDataSources[providers.PrometheusDataSource] = provider
			historyDataSources[providers.PrometheusDataSource] = provider
		}
	}
	return realtimeDataSources, historyDataSources, hybridDataSources
}

func initPredictorManager(opts *options.Options, realtimeDataSources map[providers.DataSourceType]providers.RealTime, historyDataSources map[providers.DataSourceType]providers.History) predictor.Manager {
	return predictor.NewManager(realtimeDataSources, historyDataSources, predictor.DefaultPredictorsConfig(opts.AlgorithmModelConfig))
}

// initControllers setup controllers with manager
func initControllers(ctx context.Context, mgr ctrl.Manager, opts *options.Options, predictorMgr predictor.Manager, historyDataSource providers.History) {
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

	targetSelectorFetcher := target.NewSelectorFetcher(mgr.GetScheme(), mgr.GetRESTMapper(), scaleClient, mgr.GetClient())

	podOOMRecorder := &oom.PodOOMRecorder{
		Client: mgr.GetClient(),
	}
	if err := podOOMRecorder.SetupWithManager(mgr); err != nil {
		klog.Exit(err, "Unable to create controller", "PodOOMRecorder")
	}
	go func() {
		if err := podOOMRecorder.Run(ctx.Done()); err != nil {
			klog.Warningf("Run oom recorder failed: %v", err)
		}
	}()

	if utilfeature.DefaultFeatureGate.Enabled(features.CraneAutoscaling) {
		if err := (&ehpa.EffectiveHPAController{
			Client:      mgr.GetClient(),
			Scheme:      mgr.GetScheme(),
			RestMapper:  mgr.GetRESTMapper(),
			Recorder:    mgr.GetEventRecorderFor("effective-hpa-controller"),
			ScaleClient: scaleClient,
			Config:      opts.EhpaControllerConfig,
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

		if err := (&ehpa.HPAObserverController{
			Client:     mgr.GetClient(),
			Scheme:     mgr.GetScheme(),
			RestMapper: mgr.GetRESTMapper(),
			Recorder:   mgr.GetEventRecorderFor("hpa-observer-controller"),
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "HPAObserverController")
		}

		if err := (&evpa.EffectiveVPAController{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			Recorder:      mgr.GetEventRecorderFor("effective-vpa-controller"),
			OOMRecorder:   podOOMRecorder,
			Predictor:     predictorMgr.GetPredictor(predictionapi.AlgorithmTypePercentile),
			TargetFetcher: targetSelectorFetcher,
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "EffectiveVPAController")
		}
	}

	// TspController
	if utilfeature.DefaultFeatureGate.Enabled(features.CraneTimeSeriesPrediction) {
		tspController := timeseriesprediction.NewController(
			mgr.GetClient(),
			mgr.GetEventRecorderFor("time-series-prediction-controller"),
			opts.PredictionUpdateFrequency,
			predictorMgr,
			targetSelectorFetcher,
		)
		if err := tspController.SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "TspController")
		}
	}

	if utilfeature.DefaultMutableFeatureGate.Enabled(features.CraneAnalysis) {
		if err := (&analytics.Controller{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			RestMapper:    mgr.GetRESTMapper(),
			Recorder:      mgr.GetEventRecorderFor("analytics-controller"),
			ConfigSetFile: opts.RecommendationConfigFile,
			ScaleClient:   scaleClient,
			PredictorMgr:  predictorMgr,
			Provider:      historyDataSource,
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "AnalyticsController")
		}

		if err := (&recommendation.Controller{
			Client:     mgr.GetClient(),
			Scheme:     mgr.GetScheme(),
			RestMapper: mgr.GetRESTMapper(),
			Recorder:   mgr.GetEventRecorderFor("recommendation-controller"),
		}).SetupWithManager(mgr); err != nil {
			klog.Exit(err, "unable to create controller", "controller", "RecommendationController")
		}
	}

	// CnpController
	if utilfeature.DefaultFeatureGate.Enabled(features.CraneClusterNodePrediction) {
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

func runAll(ctx context.Context, mgr ctrl.Manager, predictorMgr predictor.Manager, opts *options.Options) {
	var eg errgroup.Group

	eg.Go(func() error {
		predictorMgr.Start(ctx.Done())
		return nil
	})

	eg.Go(func() error {
		if err := mgr.Start(ctx); err != nil {
			klog.Error(err, "problem running crane manager")
			klog.Exit(err)
		}
		return nil
	})

	eg.Go(func() error {
		// Start the craned web server
		serverConfig := serverconfig.NewServerConfig()
		if err := opts.ApplyTo(serverConfig); err != nil {
			klog.Exit(err)
		}
		// use controller runtime rest config, we can not refer kubeconfig option directly because it is unexported variable in vendor/sigs.k8s.io/controller-runtime/pkg/client/config/config.go
		serverConfig.KubeRestConfig = mgr.GetConfig()
		craneServer, err := server.NewServer(serverConfig)
		if err != nil {
			klog.Exit(err)
		}
		craneServer.Run(ctx)
		return nil
	})

	// wait for all components exit
	if err := eg.Wait(); err != nil {
		klog.Fatal(err)
	}
}
