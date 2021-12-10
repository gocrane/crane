package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/spf13/cobra"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/cmd/craned/app/options"
	"github.com/gocrane/crane/pkg/controller/ehpa"
	"github.com/gocrane/crane/pkg/controller/tsp"
	"github.com/gocrane/crane/pkg/known"
	predict "github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	"github.com/gocrane/crane/pkg/prediction/percentile"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/providers/mock"
	"github.com/gocrane/crane/pkg/providers/prom"
	"github.com/gocrane/crane/pkg/utils/log"
	webhooks "github.com/gocrane/crane/pkg/webhooks"
)

var (
	scheme = runtime.NewScheme()
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
	initializationWebhooks(mgr, opts)
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
	if err := webhooks.SetupWebhookWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create webhook", "webhook", "TimeSeriesPrediction")
		os.Exit(1)
	}
}

// initializationControllers setup controllers with manager
func initializationControllers(ctx context.Context, mgr ctrl.Manager, opts *options.Options) {
	log.Logger().Info(fmt.Sprintf("opts %v", opts))
	if err := (&ehpa.EffectiveHPAController{
		Client:     mgr.GetClient(),
		Log:        log.Logger().WithName("effective-hpa-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   mgr.GetEventRecorderFor("effective-hpa-controller"),
	}).SetupWithManager(mgr); err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "EffectiveHPAController")
		os.Exit(1)
	}

	if err := (&ehpa.SubstituteController{
		Client:     mgr.GetClient(),
		Log:        log.Logger().WithName("substitute-controller"),
		Scheme:     mgr.GetScheme(),
		RestMapper: mgr.GetRESTMapper(),
		Recorder:   mgr.GetEventRecorderFor("substitute-controller"),
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
	var err error
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
		predict.RealtimeProvider: dataSource,
		predict.HistoryProvider:  dataSource,
	})
	go percentilePredictor.Run(ctx.Done())

	dspPredictor, err := dsp.NewPrediction()
	if err != nil {
		log.Logger().Error(err, "unable to create controller", "controller", "TspController")
		os.Exit(1)
	}
	dspPredictor.WithProviders(map[string]providers.Interface{
		predict.RealtimeProvider: dataSource,
		predict.HistoryProvider:  dataSource,
	})
	go dspPredictor.Run(ctx.Done())

	predictors := map[predictionapi.AlgorithmType]predict.Interface{
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
		log.Logger().Error(err, "unable to create controller", "controller", "TspController")
		os.Exit(1)
	}

}
