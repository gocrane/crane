package tsp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	predictionv1alph1 "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction"
	predconfig "github.com/gocrane/crane/pkg/prediction/config"
)

type Controller struct {
	client.Client
	Logger       logr.Logger
	Recorder     record.EventRecorder
	UpdatePeriod time.Duration

	// Per tsPredictionMap map stores last observed prediction together with a local time when it was observed.
	tsPredictionMap sync.Map

	lock sync.Mutex
	// predictors used to do predict and config, maybe the predictor should running as a independent system not as a built-in goroutines logic
	predictors map[predictionv1alph1.AlgorithmType]prediction.Interface
}

func NewController(
	client client.Client,
	logger logr.Logger,
	recorder record.EventRecorder,
	updatePeriod time.Duration,
	predictors map[predictionv1alph1.AlgorithmType]prediction.Interface,
) *Controller {
	return &Controller{
		Client:       client,
		Logger:       logger,
		Recorder:     recorder,
		UpdatePeriod: updatePeriod,
		predictors:   predictors,
	}
}

// Reconcile
func (tc *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	tc.Logger.Info("Got a time series prediction res", "tsp", req.NamespacedName)

	p := &predictionv1alph1.TimeSeriesPrediction{}

	err := tc.Client.Get(ctx, req.NamespacedName, p)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !p.DeletionTimestamp.IsZero() {
		err = tc.removeTimeSeriesPrediction(ctx, p)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return tc.syncTimeSeriesPrediction(ctx, p)
}

// SetupWithManager creates a controller and register to controller manager.
func (tc *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&predictionv1alph1.TimeSeriesPrediction{}).
		Complete(tc)
}

// sync the config to predictor
func (tc *Controller) syncTimeSeriesPrediction(ctx context.Context, prediction *predictionv1alph1.TimeSeriesPrediction) (ctrl.Result, error) {
	key := GetTimeSeriesPredictionKey(prediction)

	c := NewMetricContext(prediction)

	last, ok := tc.tsPredictionMap.Load(key)
	if !ok { // first time created or system start
		c.WithApiConfigs(prediction.Spec.PredictionMetrics)
	} else {
		if old, ok := last.(*predictionv1alph1.TimeSeriesPrediction); ok {
			// predictor need a interface to query the config and then diff.
			// now just diff the cache in the controller to decide, it can not cover all the cases when users modify the spec
			for _, newMetricConf := range prediction.Spec.PredictionMetrics {
				if !ExistsPredictionMetric(newMetricConf, old.Spec.PredictionMetrics) {
					c.WithApiConfig(&newMetricConf)
				}
			}
			for _, oldMetricConf := range old.Spec.PredictionMetrics {
				if !ExistsPredictionMetric(oldMetricConf, prediction.Spec.PredictionMetrics) {
					c.DeleteApiConfig(&oldMetricConf)
				}
			}
		} else {
			c.WithApiConfigs(prediction.Spec.PredictionMetrics)
		}
	}

	tc.tsPredictionMap.Store(key, prediction)

	return tc.syncPredictionStatus(ctx, prediction)

}

func NewMetricContext(prediction *predictionv1alph1.TimeSeriesPrediction) *predconfig.MetricContext {
	c := &predconfig.MetricContext{
		Namespace:  prediction.Namespace,
		TargetKind: prediction.Spec.TargetRef.Kind,
		Name:       prediction.Spec.TargetRef.Name,
	}
	if strings.ToLower(c.TargetKind) != predconfig.TargetKindNode && prediction.Spec.TargetRef.Namespace != "" {
		c.Namespace = prediction.Spec.TargetRef.Namespace
	}
	return c
}

func (tc *Controller) removeTimeSeriesPrediction(ctx context.Context, prediction *predictionv1alph1.TimeSeriesPrediction) error {
	c := NewMetricContext(prediction)
	c.DeleteApiConfigs(prediction.Spec.PredictionMetrics)
	return tc.Client.Delete(ctx, prediction)
}

func ExistsPredictionMetric(metric predictionv1alph1.PredictionMetric, metrics []predictionv1alph1.PredictionMetric) bool {
	for _, m := range metrics {
		if equality.Semantic.DeepEqual(&m, &metric) {
			return true
		}
	}
	return false
}

func GetTimeSeriesPredictionKey(prediction *predictionv1alph1.TimeSeriesPrediction) string {
	return fmt.Sprintf("%v/%v", prediction.Namespace, prediction.Name)
}
