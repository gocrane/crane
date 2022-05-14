package influxdb

import (
	gocontext "context"
	"github.com/gocrane/crane/pkg/common"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/query"
	"k8s.io/klog/v2"
	"time"
)

const (
	InfluxDBClientOrg = "crane"
)

type context struct {
	api             api.QueryAPI
}

// NewContext creates a new InfluxDB querying context from the given client.
func NewContext(client influxdb2.Client) *context {
	return &context{
		api:        client.QueryAPI(InfluxDBClientOrg),
	}
}

// QueryRangeSync range query influxDB in sync way
func (c *context) QueryRangeSync(ctx gocontext.Context, query string, start, end time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	// TODO
}

// QuerySync query influxDB in sync way
func (c *context) QuerySync(ctx gocontext.Context, query string) ([]*common.TimeSeries, error) {
	// TODO
	var ts []*common.TimeSeries
	results, err := c.api.Query(ctx, ``)
	if err != nil {
		return ts, err
	}
	klog.V(8).InfoS("InfluxDB query result", "result", results.Record().String())
	return c.convertInfluxDBResultsToTimeSeriesMap(results)
}

func (c *context) convertInfluxDBResultsToTimeSeriesMap(record query.FluxRecord) ([]*common.TimeSeries, error) {
	// TODO
	var results []*common.TimeSeries
	record.Start()
}