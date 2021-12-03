package tsp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/prediction"
)

// Scan all TimeSeriesPredictions and update the status if it is needed,
// update each time series prediction status window length is double of the spec.PredictionWindowSeconds.
// check the actual state of world and decide if need to update the crd status, it is periodic check to meet updateStatusDelayQueue's flaw.
func (tc *Controller) syncPredictionsStatus() error {
	predictionList := &v1alpha1.TimeSeriesPredictionList{}
	if err := tc.Client.List(context.TODO(), predictionList); err != nil {
		return err
	}

	predictions := predictionList.Items
	for i := range predictions {
		tsPrediction := &predictions[i]
		newStatus := tsPrediction.Status.DeepCopy()
		key := GetTimeSeriesPredictionKey(tsPrediction)
		if err := tc.Client.Get(context.TODO(), client.ObjectKey{Name: tsPrediction.Name, Namespace: tsPrediction.Namespace}, tsPrediction); err != nil {
			// If the prediction does not exist any more, we delete the prediction data from the map.
			if apierrors.IsNotFound(err) {
				tc.tsPredictionMap.Delete(key)
			}
			tc.Logger.Error(err, "", key, err)
			continue
		}
		// dsw check if the prediction data is out of date, if it is, force predict and update crd status,
		// or we do nothing to avoid status update frequently, reduce the api server traffic

		windowStart := time.Now()
		windowEnd := windowStart.Add(time.Duration(tsPrediction.Spec.PredictionWindowSeconds) * time.Second)
		_, warnings := tc.isPredictionDataOutDated(windowStart, windowEnd, tsPrediction.Status.PredictionMetrics)
		// force predict and update the status
		if len(warnings) > 0 {
			start := time.Now()
			// double the time to predict so that crd consumer always see time series range [now, now + PredictionWindowSeconds] in PredictionWindowSeconds window
			end := start.Add(time.Duration(tsPrediction.Spec.PredictionWindowSeconds) * time.Second * 2)

			predictedData, err := tc.doPredict(tsPrediction, start, end)
			if err != nil {
				tc.Logger.Error(err, "failed to doPredict")
			}

			if len(tsPrediction.Spec.PredictionMetrics) != len(predictedData) {
				cond := &metav1.Condition{
					Type:               string(v1alpha1.TimeSeriesPredictionConditionReady),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Message:            "Not all Metric Predicted",
					Reason:             known.ReasonTimeSeriesPredictPartial,
				}
				UpdateTimeSeriesPredictionCondition(newStatus, cond)
				err = tc.UpdateStatus(context.TODO(), tsPrediction, newStatus)
				if err != nil {
					// todo: update status failed, then add it again for update?
				}
				continue
			}

			newStatus.PredictionMetrics = predictedData
			err = tc.UpdateStatus(context.TODO(), tsPrediction, newStatus)
			if err != nil {
				// todo: update status failed, then add it again for update?
			}
			continue
		}
	}
	return nil
}

// Update the CRD status based on each crd's PredictionWindowSeconds to reduce the api server traffic
func (tc *Controller) updateStatusDelayQueue() {
	for {
		// block if no item now
		key, shutdown := tc.delayQueue.Get()
		if shutdown {
			return
		}
		pkey, ok := key.(string)
		if !ok {
			tc.Logger.Error(fmt.Errorf("wrong type key: %+v", key), "updateStatusDelayQueue")
			continue
		}

		ns, name, err := cache.SplitMetaNamespaceKey(pkey)
		if err != nil {
			tc.Logger.Error(err, "updateStatusDelayQueue")
		}
		tsPrediction := &v1alpha1.TimeSeriesPrediction{}
		err = tc.Client.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: ns}, tsPrediction)
		if err != nil {
			// If the prediction does not exist any more, we delete the prediction data from the map.
			if apierrors.IsNotFound(err) {
				tc.tsPredictionMap.Delete(key)
			}
			tc.Logger.Error(err, "updateStatusDelayQueue")
			continue
		}
		newStatus := tsPrediction.Status.DeepCopy()
		start := time.Now()
		// double the time to predict
		end := start.Add(time.Duration(tsPrediction.Spec.PredictionWindowSeconds) * time.Second * 2)
		predictionMetricsData, err := tc.doPredict(tsPrediction, start, end)
		if err != nil {
			tc.Logger.Error(err, "failed to doPredict")
			cond := &metav1.Condition{
				Type:               string(v1alpha1.TimeSeriesPredictionConditionReady),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Message:            err.Error(),
				Reason:             known.ReasonTimeSeriesPredictFailed,
			}
			UpdateTimeSeriesPredictionCondition(newStatus, cond)
			err = tc.UpdateStatus(context.TODO(), tsPrediction, newStatus)
			if err != nil {
				// todo: update status failed, then add it again for update?
				//tc.delayQueue.Add(key)
			}
			continue
		}
		if len(tsPrediction.Spec.PredictionMetrics) != len(predictionMetricsData) {
			cond := &metav1.Condition{
				Type:               string(v1alpha1.TimeSeriesPredictionConditionReady),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Message:            "Not all Metric Predicted",
				Reason:             known.ReasonTimeSeriesPredictPartial,
			}
			UpdateTimeSeriesPredictionCondition(newStatus, cond)
			err = tc.UpdateStatus(context.TODO(), tsPrediction, newStatus)
			if err != nil {
				// todo
			}
			continue
		}

		windowStart := start
		windowEnd := start.Add(time.Duration(tsPrediction.Spec.PredictionWindowSeconds) * time.Second)
		_, warnings := tc.isPredictionDataOutDated(windowStart, windowEnd, tsPrediction.Status.PredictionMetrics)
		if len(warnings) > 0 {
			cond := &metav1.Condition{
				Type:               string(v1alpha1.TimeSeriesPredictionConditionReady),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Message:            strings.Join(warnings, ";"),
				Reason:             known.ReasonTimeSeriesPredictPartial,
			}
			UpdateTimeSeriesPredictionCondition(newStatus, cond)
			err = tc.UpdateStatus(context.TODO(), tsPrediction, newStatus)
			if err != nil {
				// todo
				//tc.delayQueue.Add(key)
			}
		} else {
			cond := &metav1.Condition{
				Type:               string(v1alpha1.TimeSeriesPredictionConditionReady),
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
			}
			UpdateTimeSeriesPredictionCondition(newStatus, cond)

			err = tc.UpdateStatus(context.TODO(), tsPrediction, newStatus)
			if err != nil {
				tc.Logger.Error(err, "updateStatusDelayQueue")
			}
			// add again for next PredictionWindowSeconds to update status
			tc.delayQueue.AddAfter(key, time.Duration(tsPrediction.Spec.PredictionWindowSeconds)*time.Second)
		}
	}
}

func (tc *Controller) isPredictionDataOutDated(windowStart, windowEnd time.Time, predictionData map[string]v1alpha1.MetricTimeSeriesList) (outdated bool, warnings []string) {
	for id, metricTsList := range predictionData {
		for i, ts := range metricTsList {
			if !IsWindowInSamples(windowStart, windowEnd, ts.Samples) {
				warnings = append(warnings, fmt.Sprintf("Metric %v, ts %v, predict data is outdated, ts: %+v", id, i, ts))
				outdated = true
			}
		}
	}
	return outdated, warnings
}

func (tc *Controller) getPredictor(algorithmType v1alpha1.AlgorithmType) prediction.Interface {
	tc.lock.Lock()
	defer tc.lock.Unlock()
	return tc.predictors[algorithmType]
}

func (tc *Controller) doPredict(tsPrediction *v1alpha1.TimeSeriesPrediction, start, end time.Time) (map[string]v1alpha1.MetricTimeSeriesList, error) {
	result := make(map[string]v1alpha1.MetricTimeSeriesList)
	for _, metric := range tsPrediction.Spec.PredictionMetrics {
		predictor := tc.getPredictor(metric.Algorithm.AlgorithmType)
		if predictor == nil {
			return result, fmt.Errorf("not support algorithm type %v for metric %v", metric.Algorithm.AlgorithmType, metric.ResourceIdentifier)
		}
		if metric.Query != nil {
			queryExpr := metric.Query.Expression
			err := predictor.WithQuery(queryExpr)
			if err != nil {
				return result, err
			}
			data, err := predictor.QueryPredictedTimeSeries(queryExpr, start, end)
			if err != nil {
				return result, err
			}
			predictedData := CommonTimeSeries2ApiTimeSeries(data)
			result[metric.ResourceIdentifier] = predictedData
		} else if metric.MetricSelector != nil {
			//todo
		}
	}
	return result, nil
}

func (tc *Controller) UpdateStatus(ctx context.Context, tsPrediction *v1alpha1.TimeSeriesPrediction, newStatus *v1alpha1.TimeSeriesPredictionStatus) error {
	if !equality.Semantic.DeepEqual(&tsPrediction.Status, newStatus) {
		tc.Logger.V(4).Info("status should be updated", "currentStatus", &tsPrediction.Status, "newStatus", newStatus)

		tsPrediction.Status = *newStatus
		err := tc.Status().Update(ctx, tsPrediction)
		if err != nil {
			tc.Recorder.Event(tsPrediction, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			tc.Logger.Error(err, "Failed to update status", "time-series-prediction", klog.KObj(tsPrediction))
			return err
		}

		tc.Logger.Info("Update status successful", "time-series-prediction", klog.KObj(tsPrediction))
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

// UpdateTimeSeriesPredictionCondition updates existing timeseriesprediction condition or creates a new one. Sets LastTransitionTime to now if the
// status has changed.
// Returns true if pod condition has changed or has been added.
func UpdateTimeSeriesPredictionCondition(status *v1alpha1.TimeSeriesPredictionStatus, condition *metav1.Condition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this podgroupprediction condition.
	conditionIndex, oldCondition := GetTimeSeriesPredictionCondition(status, condition.Type)

	if oldCondition == nil {
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

// GetTimeSeriesPredictionCondition return the prediction condition of status
func GetTimeSeriesPredictionCondition(status *v1alpha1.TimeSeriesPredictionStatus, conditionType string) (int, *metav1.Condition) {
	var index int
	var condition *metav1.Condition
	if status == nil {
		return index, condition
	}
	for i, cond := range status.Conditions {
		if cond.Type == conditionType {
			index = i
			condition = &cond
		}
	}
	return index, condition
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
	startTs := start.Truncate(1 * time.Minute).Unix()
	endTs := end.Truncate(1 * time.Minute).Unix()
	if startTs >= samples[0].Timestamp && endTs <= samples[n-1].Timestamp {
		return true
	}
	return false
}
