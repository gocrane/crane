package providers

import (
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
)

// Interface is a source of monitoring metric that provides metrics that can be used for
// prediction, such as 'cpu usage', 'memory footprint', 'request per second (qps)', etc.
type Interface interface {
	RealTime
	History
}

// RealTime is a source of monitoring metric that provides data that is streamed data in current time.
type RealTime interface {
	// QueryLatestTimeSeries returns the latest metric values that meet the given metricNamer.
	QueryLatestTimeSeries(metricNamer metricnaming.MetricNamer) ([]*common.TimeSeries, error)
}

// History is a data source can provides history time series data at any time periods.
type History interface {
	// QueryTimeSeries returns the time series that meet thw given metricNamer.
	QueryTimeSeries(metricNamer metricnaming.MetricNamer, startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error)
}
