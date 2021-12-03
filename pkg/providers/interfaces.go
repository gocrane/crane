package providers

import (
	"time"

	"github.com/gocrane/crane/pkg/common"
)

// Interface is a source of monitoring metric that provides metrics that can be used for
// prediction, such as 'cpu usage', 'memory footprint', 'request per second (qps)', etc.
type Interface interface {
	// GetTimeSeries returns the metric time series that meet the given
	// conditions from the specified time range.
	GetTimeSeries(metricName string, Conditions []common.QueryCondition,
		startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error)

	// GetLatestTimeSeries returns the latest metric values that meet the given conditions.
	GetLatestTimeSeries(metricName string, Conditions []common.QueryCondition) ([]*common.TimeSeries, error)

	// QueryTimeSeries returns the time series based on a promql like query string.
	QueryTimeSeries(queryExpr string, startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error)

	// QueryLatestTimeSeries returns the latest metric values that meet the given query.
	QueryLatestTimeSeries(queryExpr string) ([]*common.TimeSeries, error)
}
