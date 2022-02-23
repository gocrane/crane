package timeseriesprediction

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/prediction"
)

const callerFormat = "TimeSeriesPredictionCaller-%s"

// check and update the status if it is needed, update each time series prediction status window length is double of the spec.PredictionWindowSeconds.
// check the actual state of world and decide if need to update the crd status,
// driven by time tick not by events, because time series prediction need to update the prediction window data to avoid the data is out of date.
// NOTE: update period is better higher resolution than the algorithm sample interval, reduce the possibility of the data is out date.
// but it is a final consistent system, so the data will be in date when next update reconcile in controller runtime.
func (tc *Controller) syncPredictionStatus(ctx context.Context, tsPrediction *v1alpha1.TimeSeriesPrediction) (ctrl.Result, error) {
	newStatus := tsPrediction.Status.DeepCopy()
	key := klog.KObj(tsPrediction)
	if err := tc.Client.Get(ctx, client.ObjectKey{Name: tsPrediction.Name, Namespace: tsPrediction.Namespace}, tsPrediction); err != nil {
		// If the prediction does not exist any more, we delete the prediction data from the map.
		if apierrors.IsNotFound(err) {
			tc.tsPredictionMap.Delete(key)
		}
		klog.Errorf("Failed to sync PredictionsStatus for %v, err: %v", key, err)
		// time driven
		return ctrl.Result{RequeueAfter: tc.UpdatePeriod}, err
	}
	// check if the prediction data is out of date, if it is, force predict and update crd status,
	// or we do nothing to avoid status update frequently, reduce the api server traffic

	windowStart := time.Now()
	windowEnd := windowStart.Add(time.Duration(tsPrediction.Spec.PredictionWindowSeconds) * time.Second)
	warnings := tc.isPredictionDataOutDated(windowStart, windowEnd, tsPrediction.Status.PredictionMetrics)
	// force predict and update the status
	if len(warnings) > 0 {
		klog.V(4).Infof("Check status predict data is out of date. range: %v, key: %v", fmt.Sprintf("[%v, %v]", windowStart, windowEnd), key)
		predictionStart := time.Now()
		// double the time to predict so that crd consumer always see time series range [now, now + PredictionWindowSeconds] in PredictionWindowSeconds window
		predictionEnd := predictionStart.Add(time.Duration(tsPrediction.Spec.PredictionWindowSeconds) * time.Second * 2)

		predictedData, err := tc.doPredict(tsPrediction, predictionStart, predictionEnd)
		if err != nil {
			tc.Recorder.Event(tsPrediction, v1.EventTypeWarning, "FailedPredict", err.Error())
			klog.Errorf("Failed to doPredict, err: %v", err)
			return ctrl.Result{RequeueAfter: tc.UpdatePeriod}, err
		}
		newStatus.PredictionMetrics = predictedData

		if len(tsPrediction.Spec.PredictionMetrics) != len(predictedData) {
			klog.V(4).Infof("DoPredict predict data is partial, predictedDataLen: %v, key: %v", len(predictedData), key)
			setCondition(newStatus, v1alpha1.TimeSeriesPredictionConditionReady, metav1.ConditionFalse, known.ReasonTimeSeriesPredictPartial, "not all metric predicted")
			err = tc.UpdateStatus(ctx, tsPrediction, newStatus)
			if err != nil {
				// todo
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: tc.UpdatePeriod}, err
		}

		windowStart := predictionStart
		windowEnd := predictionStart.Add(time.Duration(tsPrediction.Spec.PredictionWindowSeconds) * time.Second)
		warnings := tc.isPredictionDataOutDated(windowStart, windowEnd, predictedData)
		if len(warnings) > 0 {
			klog.V(4).Infof("DoPredict predict data is partial, range: %v, key: %v", fmt.Sprintf("[%v, %v]", windowStart, windowEnd), key)
			setCondition(newStatus, v1alpha1.TimeSeriesPredictionConditionReady, metav1.ConditionFalse, known.ReasonTimeSeriesPredictPartial, strings.Join(warnings, ";"))
			err = tc.UpdateStatus(ctx, tsPrediction, newStatus)
			if err != nil {
				// todo
				return ctrl.Result{}, err
			}
		} else {
			klog.V(4).Infof("DoPredict predict data is complete, range: %v, key: %v", fmt.Sprintf("[%v, %v]", windowStart, windowEnd), key)
			// status.conditions.reason in body should be at least 1 chars long
			setCondition(newStatus, v1alpha1.TimeSeriesPredictionConditionReady, metav1.ConditionTrue, known.ReasonTimeSeriesPredictSucceed, "")

			err = tc.UpdateStatus(ctx, tsPrediction, newStatus)
			if err != nil {
				// todo: update status failed, then add it again for update?
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: tc.UpdatePeriod}, nil
		}
	}
	return ctrl.Result{RequeueAfter: tc.UpdatePeriod}, nil
}

func (tc *Controller) isPredictionDataOutDated(windowStart, windowEnd time.Time, predictionMetricStatus []v1alpha1.PredictionMetricStatus) (warnings []string) {
	if len(predictionMetricStatus) == 0 {
		warnings = append(warnings, "no predict data")
		return warnings
	}
	for _, predictedData := range predictionMetricStatus {
		if len(predictedData.Prediction) == 0 {
			warnings = append(warnings, fmt.Sprintf("metric %v no predict data", predictedData.ResourceIdentifier))
		}
		for i, ts := range predictedData.Prediction {
			if !IsWindowInSamples(windowStart, windowEnd, ts.Samples) {
				warnings = append(warnings, fmt.Sprintf("metric %v, ts %v, predict data is outdated, labels: %+v", predictedData.ResourceIdentifier, i, ts.Labels))
			}
		}
	}
	return warnings
}

func (tc *Controller) getPredictor(algorithmType v1alpha1.AlgorithmType) prediction.Interface {
	tc.lock.Lock()
	defer tc.lock.Unlock()
	return tc.predictors[algorithmType]
}

func (tc *Controller) doPredict(tsPrediction *v1alpha1.TimeSeriesPrediction, start, end time.Time) ([]v1alpha1.PredictionMetricStatus, error) {
	var result []v1alpha1.PredictionMetricStatus
	c := NewMetricContext(tsPrediction)
	for _, metric := range tsPrediction.Spec.PredictionMetrics {
		predictor := tc.getPredictor(metric.Algorithm.AlgorithmType)
		if predictor == nil {
			return result, fmt.Errorf("do not support algorithm type %v for metric %v", metric.Algorithm.AlgorithmType, metric.ResourceIdentifier)
		}
		var queryExpr string

		if metric.ResourceQuery != nil {
			queryExpr = c.ResourceToPromQueryExpr(metric.ResourceQuery)
		} else if metric.MetricQuery != nil {
			//todo
			return result, fmt.Errorf("do not support query type %v, metric %v now", metric.ExpressionQuery, metric.ResourceIdentifier)
		} else {
			queryExpr = metric.ExpressionQuery.Expression
		}

		err := predictor.WithQuery(queryExpr, fmt.Sprintf(callerFormat, tsPrediction.UID))
		if err != nil {
			return result, err
		}
		var data []*common.TimeSeries
		// percentile is ok for time series
		data, err = predictor.QueryPredictedTimeSeries(queryExpr, start, end)
		if err != nil {
			return result, err
		}
		predictedData := CommonTimeSeries2ApiTimeSeries(data)
		if klog.V(6).Enabled() {
			apiDataBytes, err1 := json.Marshal(predictedData)
			dataBytes, err2 := json.Marshal(data)
			klog.V(6).Infof("DoPredict predicted data details, key: %v, queryExpr: %v, apiData: %v, predictData: %v, errs: %+v", klog.KObj(tsPrediction), queryExpr, string(apiDataBytes), string(dataBytes), []error{err1, err2})
		}
		result = append(result, v1alpha1.PredictionMetricStatus{ResourceIdentifier: metric.ResourceIdentifier, Prediction: predictedData})
	}
	return result, nil
}

func (tc *Controller) UpdateStatus(ctx context.Context, tsPrediction *v1alpha1.TimeSeriesPrediction, newStatus *v1alpha1.TimeSeriesPredictionStatus) error {
	if !equality.Semantic.DeepEqual(&tsPrediction.Status, newStatus) {
		tsPrediction.Status = *newStatus
		err := tc.Client.Status().Update(ctx, tsPrediction)
		if err != nil {
			tc.Recorder.Event(tsPrediction, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status for %v", klog.KObj(tsPrediction))
			return err
		}

		klog.V(4).Infof("Update status successful for %v", klog.KObj(tsPrediction))
	}
	return nil
}

func CommonTimeSeries2ApiTimeSeries(tsList []*common.TimeSeries) v1alpha1.MetricTimeSeriesList {
	var list v1alpha1.MetricTimeSeriesList
	for _, ts := range tsList {
		mts := v1alpha1.MetricTimeSeries{
			Labels:  make([]v1alpha1.Label, len(ts.Labels)),
			Samples: make([]v1alpha1.Sample, len(ts.Samples)),
		}
		for i, label := range ts.Labels {
			mts.Labels[i] = v1alpha1.Label{Name: label.Name, Value: label.Value}
		}
		for i, sample := range ts.Samples {
			mts.Samples[i] = v1alpha1.Sample{Timestamp: sample.Timestamp, Value: fmt.Sprintf("%.5f", sample.Value)}
		}

		list = append(list, &mts)
	}
	return list
}

func setCondition(status *v1alpha1.TimeSeriesPredictionStatus, conditionType v1alpha1.PredictionConditionType, conditionStatus metav1.ConditionStatus, reason string, message string) {
	for i := range status.Conditions {
		if status.Conditions[i].Type == string(conditionType) {
			status.Conditions[i].Status = conditionStatus
			status.Conditions[i].Reason = reason
			status.Conditions[i].Message = message
			status.Conditions[i].LastTransitionTime = metav1.Now()
			return
		}
	}
	status.Conditions = append(status.Conditions, metav1.Condition{
		Type:               string(conditionType),
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func IsWindowInSamples(start, end time.Time, samples []v1alpha1.Sample) bool {
	n := len(samples)
	if n == 0 {
		return false
	}
	sort.Slice(samples, func(i, j int) bool {
		if samples[i].Timestamp < samples[j].Timestamp {
			return true
		} else {
			return false
		}
	})
	// todo: this step param depends on data source or algorithms???
	//startTs := start.Truncate(1 * time.Minute).Unix()
	endTs := end.Truncate(1 * time.Minute).Unix()
	// only check the end, start not check, because start is always from now to predict
	if endTs <= samples[n-1].Timestamp {
		return true
	}
	return false
}
