package prediction

import (
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/providers"
)

type Interface interface {
	// Run performs the prediction routine.
	Run(stopCh <-chan struct{})

	// WithProviders registers providers from which metrics data come from
	WithProviders(map[string]providers.Interface)

	// WithMetric registers a metric whose values should be predicted in realtime.
	//WithMetric(metricName string, conditions []common.QueryCondition) error

	// WithQuery registers a PromQL like query expression, so that the prediction will involve the time series that
	// are selected and aggregated through the specified 'queryExpr'.
	WithQuery(queryExpr string, caller string) error

	// DeleteQuery unregisters a query expression, so that its prediction routine will be stopped.
	DeleteQuery(queryExpr string, caller string) error

	// GetRealtimePredictedValues returns the predicted values
	//GetRealtimePredictedValues(metricName string, Conditions []common.QueryCondition) ([]*common.TimeSeries, error)

	// QueryRealtimePredictedValues returns predicted values based on the specified query expression
	QueryRealtimePredictedValues(queryExpr string) ([]*common.TimeSeries, error)

	// GetPredictedTimeSeries returns predicted metric time series in the given time range.
	//GetPredictedTimeSeries(
	//	metricName string,
	//	conditions []common.QueryCondition,
	//	startTime time.Time,
	//	endTime time.Time) ([]*common.TimeSeries, error)

	// QueryPredictedTimeSeries returns predicted time series based on the specified query expression
	QueryPredictedTimeSeries(queryExpr string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error)

	Name() string
}
