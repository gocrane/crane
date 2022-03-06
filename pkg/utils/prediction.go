package utils

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
)

func QueryPredictedTimeSeriesOnce(predictor prediction.Interface, caller string, pConfig *config.Config, queryExpr string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	err := predictor.WithQuery(queryExpr, caller, *pConfig)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := predictor.DeleteQuery(queryExpr, caller)
		if err != nil {
			klog.ErrorS(err, "Failed to delete query.", "queryExpr", queryExpr, "caller", caller)
		}
	}()

	return predictor.QueryPredictedTimeSeries(context.TODO(), queryExpr, startTime, endTime)
}

func QueryPredictedValuesOnce(recommendation *v1alpha1.Recommendation, predictor prediction.Interface, caller string, pConfig *config.Config, queryExpr string) ([]*common.TimeSeries, error) {
	err := predictor.WithQuery(queryExpr, caller, *pConfig)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := predictor.DeleteQuery(queryExpr, caller)
		if err != nil {
			klog.ErrorS(err, "Failed to delete query.", "queryExpr", queryExpr, "caller", caller)
		}
	}()

	return predictor.QueryRealtimePredictedValues(context.TODO(), queryExpr)
}
