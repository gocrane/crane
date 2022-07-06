package utils

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
)

func QueryPredictedTimeSeriesOnce(predictor prediction.Interface, caller string, pConfig *config.Config, namer metricnaming.MetricNamer, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	err := predictor.WithQuery(namer, caller, *pConfig)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := predictor.DeleteQuery(namer, caller)
		if err != nil {
			klog.ErrorS(err, "Failed to delete query.", "queryExpr", namer, "caller", caller)
		}
	}()

	return predictor.QueryPredictedTimeSeries(context.TODO(), namer, startTime, endTime)
}

func QueryPredictedValues(predictor prediction.Interface, caller string, pConfig *config.Config, namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	err := predictor.WithQuery(namer, caller, *pConfig)
	if err != nil {
		return nil, err
	}

	return predictor.QueryRealtimePredictedValues(context.TODO(), namer)
}

func QueryPredictedValuesOnce(recommendation *v1alpha1.Recommendation, predictor prediction.Interface, caller string, pConfig *config.Config, namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	return predictor.QueryRealtimePredictedValuesOnce(context.TODO(), namer, *pConfig)
}

func GetReadyPredictionMetric(metric string, prediction *predictionapi.TimeSeriesPrediction) (*predictionapi.MetricTimeSeries, error) {
	var targetMetricStatus *predictionapi.PredictionMetricStatus
	for _, metricStatus := range prediction.Status.PredictionMetrics {
		if metricStatus.ResourceIdentifier == metric && len(metricStatus.Prediction) == 1 {
			targetMetricStatus = &metricStatus
		}
	}

	if targetMetricStatus == nil {
		return nil, fmt.Errorf("TimeSeries is empty, metric name %s", metric)
	}

	if !targetMetricStatus.Ready {
		return nil, fmt.Errorf("TimeSeries is not ready, metric name %s", metric)
	}

	if len(targetMetricStatus.Prediction) != 1 {
		return nil, fmt.Errorf("TimeSeries data length is unexpected: %d", len(targetMetricStatus.Prediction))
	}

	return targetMetricStatus.Prediction[0], nil
}
