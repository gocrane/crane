package tsp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	predictionv1alph1 "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction"
	predconfig "github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/utils/log"
)

var logger = log.Logger()

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

	// delayQueue is used to put the TimeSeriesPrediction based on PredictionWindowSeconds
	delayQueue workqueue.DelayingInterface
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
		delayQueue:   workqueue.NewNamedDelayingQueue("tsp-controller"),
	}
}

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

	tc.syncTimeSeriesPrediction(ctx, p)

	return ctrl.Result{}, nil
}

// Start starts an asynchronous loop that update the status of TimeSeriesPrediction.
// Scan all of the predictions in actual state of worlds, check if the prediction need to update, then get the predicted data.
func (c *Controller) Start(ctx context.Context) error {
	c.Logger.Info("Starting TimeSeriesPrediction updator")
	defer c.Logger.Info("Shutting TimeSeriesPrediction updator")

	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		if err := c.syncPredictionsStatus(); err != nil {
			c.Logger.Error(err, "Error syncPredictionsStatus")
		}
	}, c.UpdatePeriod)

	go c.updateStatusDelayQueue()

	<-ctx.Done()

	return nil
}

// SetupWithManager creates a controller and register to controller manager.
func (tc *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&predictionv1alph1.TimeSeriesPrediction{}).
		Complete(tc)
}

// sync the config to predictor
func (tc *Controller) syncTimeSeriesPrediction(ctx context.Context, prediction *predictionv1alph1.TimeSeriesPrediction) error {
	key := GetTimeSeriesPredictionKey(prediction)

	last, ok := tc.tsPredictionMap.Load(key)
	if !ok { // first time created

		predconfig.WithApiConfigs(prediction.Spec.PredictionMetrics)
		//newStatus := prediction.Status.DeepCopy()
		//cond := &predictionv1alph1.TimeSeriesPredictionCondition{
		//	Type:               predictionv1alph1.TimeSeriesPredictionConditionNotReady,
		//	Status:             metav1.ConditionTrue,
		//	LastProbeTime:      metav1.Now(),
		//	LastTransitionTime: metav1.Now(),
		//}
		//UpdateTimeSeriesPredictionCondition(newStatus, cond)
	} else {
		if old, ok := last.(*predictionv1alph1.TimeSeriesPrediction); ok {
			// predictor need a interface to query the config and then diff.
			// now just diff the cache in the controller to decide, it can not cover all the cases when users modify the spec
			for _, newMetricConf := range prediction.Spec.PredictionMetrics {
				if !ExistsPredictionMetric(newMetricConf, old.Spec.PredictionMetrics) {
					predconfig.WithApiConfig(&newMetricConf)
				}
			}
			for _, oldMetricConf := range old.Spec.PredictionMetrics {
				if !ExistsPredictionMetric(oldMetricConf, prediction.Spec.PredictionMetrics) {
					predconfig.DeleteApiConfig(&oldMetricConf)
				}
			}
		} else {
			predconfig.WithApiConfigs(prediction.Spec.PredictionMetrics)
		}
	}

	tc.tsPredictionMap.Store(key, prediction)
	// add the prediction to time delay queue for update
	tc.delayQueue.AddAfter(key, time.Duration(prediction.Spec.PredictionWindowSeconds)*time.Second)

	return nil
}

func (tc *Controller) removeTimeSeriesPrediction(ctx context.Context, prediction *predictionv1alph1.TimeSeriesPrediction) error {
	predconfig.DeleteApiConfigs(prediction.Spec.PredictionMetrics)
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
