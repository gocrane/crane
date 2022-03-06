package prediction

import (
	"context"
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/providers"
)

type Interface interface {
	// Run performs the prediction routine.
	Run(stopCh <-chan struct{})

	// WithProviders registers providers from which metrics data come from
	// todo move to constructors
	WithProviders(map[string]providers.Interface)

	// WithQuery registers a PromQL like query expression, so that the prediction will involve the time series that
	// are selected and aggregated through the specified 'queryExpr'.
	WithQuery(queryExpr string, caller string, config config.Config) error

	DeleteQuery(queryExpr string, caller string) error

	// QueryRealtimePredictedValues returns predicted values based on the specified query expression
	QueryRealtimePredictedValues(ctx context.Context, queryExpr string) ([]*common.TimeSeries, error)

	// QueryPredictedTimeSeries returns predicted time series based on the specified query expression
	QueryPredictedTimeSeries(ctx context.Context, queryExpr string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error)

	Name() string
}
