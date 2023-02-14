package timeseriesprediction

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/utils/target"
)

type Controller struct {
	client.Client
	Recorder      record.EventRecorder
	UpdatePeriod  time.Duration
	TargetFetcher target.SelectorFetcher
	Scheme        *runtime.Scheme
	RestMapper    meta.RESTMapper
	ScaleClient   scale.ScalesGetter

	// Per tsPredictionMap map stores last observed prediction together with a local time when it was observed.
	tsPredictionMap sync.Map

	lock sync.Mutex
	// predictors used to do predict and config, maybe the predictor should running as a independent system not as a built-in goroutines evaluator
	predictorMgr predictormgr.Manager
}

func NewController(
	client client.Client,
	recorder record.EventRecorder,
	updatePeriod time.Duration,
	predictorMgr predictormgr.Manager,
	targetFetcher target.SelectorFetcher,
) *Controller {
	return &Controller{
		Client:        client,
		Recorder:      recorder,
		UpdatePeriod:  updatePeriod,
		predictorMgr:  predictorMgr,
		TargetFetcher: targetFetcher,
	}
}

// Reconcile reconcile the time series prediction
func (tc *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got a time series prediction %v", req.NamespacedName)

	p := &predictionapi.TimeSeriesPrediction{}
	err := tc.Client.Get(ctx, req.NamespacedName, p)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.V(4).Infof("Failed to get TimeSeriesPrediction %v err: %v", klog.KObj(p), err)
			return ctrl.Result{Requeue: true}, err
		}

		last, ok := tc.tsPredictionMap.Load(req.NamespacedName.String())
		if !ok {
			klog.V(4).Infof("Failed to load exist tsp %v", req.String())
			return ctrl.Result{}, nil
		}

		tsp, ok := last.(*predictionapi.TimeSeriesPrediction)
		if !ok {
			return ctrl.Result{}, fmt.Errorf("assert tsp failed for tsp %v in map", req.String())
		}

		err := tc.removeTimeSeriesPrediction(tsp)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	return tc.syncTimeSeriesPrediction(ctx, p)
}

// SetupWithManager creates a controller and register to controller manager.
func (tc *Controller) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		For(&predictionapi.TimeSeriesPrediction{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(tc)
}

// sync the config to predictor
func (tc *Controller) syncTimeSeriesPrediction(ctx context.Context, tsp *predictionapi.TimeSeriesPrediction) (ctrl.Result, error) {
	key := GetTimeSeriesPredictionKey(tsp)

	c, err := NewMetricContext(tc.TargetFetcher, tsp, tc.predictorMgr)
	if err != nil {
		klog.ErrorS(err, "Failed to NewMetricContext.")
		return ctrl.Result{}, err
	}

	func() {
		last, ok := tc.tsPredictionMap.Load(key)
		if !ok { // first time created or system start
			c.WithApiConfigs(tsp.Spec.PredictionMetrics)
			return
		}
		old, ok := last.(*predictionapi.TimeSeriesPrediction)
		if !ok {
			c.WithApiConfigs(tsp.Spec.PredictionMetrics)
			return
		}
		// predictor needs an interface to query the config and then diff.
		// now just diff the cache in the controller to decide, it can not cover all the cases when users modify the spec
		for _, oldMetricConf := range old.Spec.PredictionMetrics {
			if !ExistsPredictionMetric(oldMetricConf, tsp.Spec.PredictionMetrics) {
				c.DeleteApiConfig(&oldMetricConf)
			}
		}
		for _, newMetricConf := range tsp.Spec.PredictionMetrics {
			c.WithApiConfig(&newMetricConf)
		}
	}()

	tc.tsPredictionMap.Store(key, tsp)

	return tc.syncPredictionStatus(ctx, tsp)

}

func (tc *Controller) removeTimeSeriesPrediction(tsp *predictionapi.TimeSeriesPrediction) error {
	klog.V(4).Infof("TimeSeriesPrediction %v deleted, delete metric config", klog.KObj(tsp))
	c, err := NewMetricContext(tc.TargetFetcher, tsp, tc.predictorMgr)
	if err != nil {
		klog.Errorf("Failed to delete TimeSeriesPrediction %v: %v", klog.KObj(tsp), err)
		return err
	}
	c.DeleteApiConfigs(tsp.Spec.PredictionMetrics)
	key := GetTimeSeriesPredictionKey(tsp)
	tc.tsPredictionMap.Delete(key)
	return nil
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
