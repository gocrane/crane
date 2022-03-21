package utils

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/api/analysis/v1alpha1"

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

func QueryPredictedValuesOnce(recommendation *v1alpha1.Recommendation, predictor prediction.Interface, caller string, pConfig *config.Config, namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	return predictor.QueryRealtimePredictedValuesOnce(context.TODO(), namer, *pConfig)
}
