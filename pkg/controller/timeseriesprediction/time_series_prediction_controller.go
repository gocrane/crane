package timeseriesprediction

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction"
)

type Controller struct {
	client.Client
	Recorder     record.EventRecorder
	UpdatePeriod time.Duration

	// Per tsPredictionMap map stores last observed prediction together with a local time when it was observed.
	tsPredictionMap sync.Map

	lock sync.Mutex
	// predictors used to do predict and config, maybe the predictor should running as a independent system not as a built-in goroutines evaluator
	predictors map[predictionapi.AlgorithmType]prediction.Interface
}

func NewController(
	client client.Client,
	recorder record.EventRecorder,
	updatePeriod time.Duration,
	predictors map[predictionapi.AlgorithmType]prediction.Interface,
) *Controller {
	return &Controller{
		Client:       client,
		Recorder:     recorder,
		UpdatePeriod: updatePeriod,
		predictors:   predictors,
	}
}

// Reconcile reconcile the time series prediction
func (tc *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got a time series prediction %v", req.NamespacedName)

	p := &predictionapi.TimeSeriesPrediction{}
	err := tc.Client.Get(ctx, req.NamespacedName, p)
	if err != nil {
		if errors.IsNotFound(err) {
			if last, ok := tc.tsPredictionMap.Load(req.NamespacedName.String()); ok {
				if tsp, ok := last.(*predictionapi.TimeSeriesPrediction); ok {
					tc.removeTimeSeriesPrediction(tsp)
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, fmt.Errorf("assert tsp failed for tsp %v in map", req.String())
			}
			klog.V(4).Infof("Failed to load exist tsp %v", req.String())
			return ctrl.Result{}, nil
		}
		klog.V(4).Infof("Failed to get TimeSeriesPrediction %v err: %v", klog.KObj(p), err)
		return ctrl.Result{Requeue: true}, err
	}

	return tc.syncTimeSeriesPrediction(ctx, p)
}

// SetupWithManager creates a controller and register to controller manager.
func (tc *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&predictionapi.TimeSeriesPrediction{}).
		Complete(tc)
}

// sync the config to predictor
func (tc *Controller) syncTimeSeriesPrediction(ctx context.Context, tsp *predictionapi.TimeSeriesPrediction) (ctrl.Result, error) {
	key := GetTimeSeriesPredictionKey(tsp)

	c := NewMetricContext(tsp, tc.predictors)

	last, ok := tc.tsPredictionMap.Load(key)
	if !ok { // first time created or system start
		c.WithApiConfigs(tsp.Spec.PredictionMetrics)
	} else {
		if old, ok := last.(*predictionapi.TimeSeriesPrediction); ok {
			// predictor need a interface to query the config and then diff.
			// now just diff the cache in the controller to decide, it can not cover all the cases when users modify the spec
			for _, newMetricConf := range tsp.Spec.PredictionMetrics {
				if !ExistsPredictionMetric(newMetricConf, old.Spec.PredictionMetrics) {
					c.WithApiConfig(&newMetricConf)
				}
			}
			for _, oldMetricConf := range old.Spec.PredictionMetrics {
				if !ExistsPredictionMetric(oldMetricConf, tsp.Spec.PredictionMetrics) {
					c.DeleteApiConfig(&oldMetricConf)
				}
			}
		} else {
			c.WithApiConfigs(tsp.Spec.PredictionMetrics)
		}
	}

	tc.tsPredictionMap.Store(key, tsp)

	return tc.syncPredictionStatus(ctx, tsp)

}

func (tc *Controller) removeTimeSeriesPrediction(tsp *predictionapi.TimeSeriesPrediction) {
	klog.V(4).Infof("TimeSeriesPrediction %v deleted, delete metric config", klog.KObj(tsp))
	c := NewMetricContext(tsp, tc.predictors)
	c.DeleteApiConfigs(tsp.Spec.PredictionMetrics)
	key := GetTimeSeriesPredictionKey(tsp)
	tc.tsPredictionMap.Delete(key)
}

func ExistsPredictionMetric(metric predictionapi.PredictionMetric, metrics []predictionapi.PredictionMetric) bool {
	for _, m := range metrics {
		if equality.Semantic.DeepEqual(&m, &metric) {
			return true
		}
	}
	return false
}

func GetTimeSeriesPredictionKey(tsp *predictionapi.TimeSeriesPrediction) string {
	return fmt.Sprintf("%v/%v", tsp.Namespace, tsp.Name)
}
