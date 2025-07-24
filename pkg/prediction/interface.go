package prediction

import (
	"context"
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction/config"
)

type Interface interface {
	// Run performs the prediction routine.
	Run(stopCh <-chan struct{})

	// WithQuery registers a PromQL like query expression, so that the prediction will involve the time series that
	// are selected and aggregated through the specified 'queryExpr'.
	WithQuery(metricNamer metricnaming.MetricNamer, caller string, config config.Config) error

	DeleteQuery(metricNamer metricnaming.MetricNamer, caller string) error

	// QueryPredictionStatus return the metricNamer prediction status. it is predictable only when it is ready
	QueryPredictionStatus(ctx context.Context, metricNamer metricnaming.MetricNamer) (Status, error)

	// QueryRealtimePredictedValues returns predicted values based on the specified query expression
	QueryRealtimePredictedValues(ctx context.Context, metricNamer metricnaming.MetricNamer) ([]*common.TimeSeries, error)

	// QueryPredictedTimeSeries returns predicted time series based on the specified query expression
	QueryPredictedTimeSeries(ctx context.Context, metricNamer metricnaming.MetricNamer, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error)

	// A analysis task function
	QueryRealtimePredictedValuesOnce(ctx context.Context, namer metricnaming.MetricNamer, config config.Config) ([]*common.TimeSeries, error)

	Name() string
}
