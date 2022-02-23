package utils

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
)

func PredictionQueryTimeSeriesOnce(predictor prediction.Interface, caller string, metricConfig *config.Config, query string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	metricContext := &config.MetricContext{}
	err := predictor.WithQuery(query, caller)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := predictor.DeleteQuery(query, caller)
		if err != nil {
			klog.Errorf("Delete query failed: %v", err)
		}
	}()

	metricContext.WithConfig(metricConfig)
	return predictor.QueryPredictedTimeSeries(query, startTime, endTime)
}

func PredictionQueryTimeSeriesValuesOnce(predictor prediction.Interface, caller string, metricConfig *config.Config, query string) ([]*common.TimeSeries, error) {
	metricContext := &config.MetricContext{}
	err := predictor.WithQuery(query, caller)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := predictor.DeleteQuery(query, caller)
		if err != nil {
			klog.Errorf("Delete query failed: %v", err)
		}
	}()

	metricContext.WithConfig(metricConfig)
	return predictor.QueryRealtimePredictedValues(query)
}
