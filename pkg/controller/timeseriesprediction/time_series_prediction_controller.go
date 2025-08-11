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

	// 缓存每个tsp的MetricContext，key为namespace/name
	metricContextMap sync.Map

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
		Client:           client,
		Recorder:         recorder,
		UpdatePeriod:     updatePeriod,
		predictorMgr:     predictorMgr,
		TargetFetcher:    targetFetcher,
		metricContextMap: sync.Map{}, // 初始化
	}
}

// Reconcile reconcile the time series prediction
func (tc *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got a time series prediction %v", req.NamespacedName)
	startTime := time.Now()
	defer func() {
		klog.V(1).Infof("Finished syncing tsp[%s] (%v)", req.NamespacedName, time.Since(startTime))
	}()
	p := &predictionapi.TimeSeriesPrediction{}
	err := tc.Client.Get(ctx, req.NamespacedName, p)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.V(4).Infof("Failed to get TimeSeriesPrediction %v err: %v", klog.KObj(p), err)
			return ctrl.Result{}, err
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
			klog.ErrorS(err, "Failed to remove TimeSeriesPrediction", "key", req.NamespacedName.String())
			return ctrl.Result{}, err
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

func (tc *Controller) getCachedMetricContext(key string) (*MetricContext, bool) {
	cached, ok := tc.metricContextMap.Load(key)
	if !ok {
		return nil, false
	}
	c, _ := cached.(*MetricContext)
	return c, true
}

// sync the config to predictor
func (tc *Controller) syncTimeSeriesPrediction(ctx context.Context, tsp *predictionapi.TimeSeriesPrediction) (ctrl.Result, error) {
	key := GetTimeSeriesPredictionKey(tsp)

	// 优先从缓存获取 MetricContext
	var err error
	c, ok := tc.getCachedMetricContext(key)
	if ok {
		lastTsp, ok := tc.tsPredictionMap.Load(key)
		if ok { // 存在历史的同名同命名空间的tsp，但是可能是旧的
			old, _ := lastTsp.(*predictionapi.TimeSeriesPrediction)
			// 检查 spec 是否发生变化，如果发生变化则重建 MetricContext
			if !equality.Semantic.DeepEqual(old.Spec, tsp.Spec) {
				klog.V(1).Infof("TimeSeriesPrediction %v spec changed, recreate MetricContext", klog.KObj(tsp))
				c.DeleteApiConfigs(tsp.Spec.PredictionMetrics) // 删除旧的配置
				c, err = NewMetricContext(tc.TargetFetcher, tsp, tc.predictorMgr)
				if err != nil {
					klog.ErrorS(err, "Failed to NewMetricContext.", "key", key)
					return ctrl.Result{}, err
				}
				c.WithApiConfigs(tsp.Spec.PredictionMetrics)
				tsp.Status.PredictionMetrics = nil // Spec内容变化后，清空预测状态，强制触发预测
				tc.metricContextMap.Store(key, c)
			}
		}
	} else { // first time created or system start
		klog.V(1).Infof("Creating new MetricContext for TimeSeriesPrediction %v", klog.KObj(tsp))
		c, err = NewMetricContext(tc.TargetFetcher, tsp, tc.predictorMgr)
		if err != nil {
			klog.ErrorS(err, "Failed to NewMetricContext", "key", key)
			return ctrl.Result{}, err
		}
		c.WithApiConfigs(tsp.Spec.PredictionMetrics)
		tc.metricContextMap.Store(key, c)
	}

	tc.tsPredictionMap.Store(key, tsp)

	return tc.syncPredictionStatus(ctx, tsp)
}

func (tc *Controller) removeTimeSeriesPrediction(tsp *predictionapi.TimeSeriesPrediction) error {
	klog.V(4).Infof("TimeSeriesPrediction %v deleted, delete metric config", klog.KObj(tsp))
	key := GetTimeSeriesPredictionKey(tsp)
	c, ok := tc.getCachedMetricContext(key)
	if !ok {
		klog.Errorf("Failed to load cached MetricContext for TimeSeriesPrediction %v", klog.KObj(tsp))
		return fmt.Errorf("cached MetricContext not found for TimeSeriesPrediction %v", klog.KObj(tsp))
	}
	c.DeleteApiConfigs(tsp.Spec.PredictionMetrics)
	tc.tsPredictionMap.Delete(key)
	tc.metricContextMap.Delete(key)
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
