package influxdb

import (
	gocontext "context"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/providers"
	"k8s.io/klog/v2"
	"time"
)

type influxDB struct {
	ctx    *context
	config *providers.InfluxDBConfig
}

// NewProvider return a prometheus data provider
func NewProvider(config *providers.InfluxDBConfig) (providers.Interface, error) {
	client, err := NewInfluxDBClient(config)
	if err != nil {
		return nil, err
	}

	ctx := NewContext(client)

	return &influxDB{ctx: ctx, config: config}, nil
}


func (i *influxDB) QueryTimeSeries(namer metricnaming.MetricNamer, startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	influxdbBuilder := namer.QueryBuilder().Builder(metricquery.PrometheusMetricSource)
	influxdbQuery, err := influxdbBuilder.BuildQuery()
	if err != nil {
		klog.Errorf("Failed to BuildQuery: %v", err)
		return nil, err
	}
	klog.V(6).Infof("QueryTimeSeries metricNamer %v, timeout: %v", namer.BuildUniqueKey(), i.config.Timeout)
	timeoutCtx, cancelFunc := gocontext.WithTimeout(gocontext.Background(), i.config.Timeout)
	defer cancelFunc()
	timeSeries, err := i.ctx.QueryRangeSync(timeoutCtx, influxdbQuery.Prometheus.Query, startTime, endTime, step)
	if err != nil {
		klog.Errorf("Failed to QueryTimeSeries: %v, metricNamer: %v, query: %v", err, namer.BuildUniqueKey(), influxdbQuery.Prometheus.Query)
		return nil, err
	}
	return timeSeries, nil
}

func (i *influxDB) QueryLatestTimeSeries(namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	influxdbBuilder := namer.QueryBuilder().Builder(metricquery.InfluxDBMetricSource)
	influxdbQuery, err := influxdbBuilder.BuildQuery()
	if err != nil {
		klog.Errorf("Failed to QueryLatestTimeSeries metricNamer %v, err: %v", namer.BuildUniqueKey(), err)
		return nil, err
	}
	klog.V(6).Infof("QueryLatestTimeSeries metricNamer %v", namer.BuildUniqueKey())
	// use range query for latest too. because the queryExpr is an range in crd spec
	//end := time.Now()
	// avoid no data latest. multiply 2
	//start := end.Add(-step * 2)
	klog.V(6).Infof("QueryLatestTimeSeries metricNamer %v, timeout: %v", namer.BuildUniqueKey(), i.config.Timeout)
	timeoutCtx, cancelFunc := gocontext.WithTimeout(gocontext.Background(), i.config.Timeout)
	defer cancelFunc()
	timeSeries, err := i.ctx.QuerySync(timeoutCtx, influxdbQuery.Prometheus.Query)
	if err != nil {
		klog.Errorf("Failed to QueryLatestTimeSeries: %v, metricNamer: %v, query: %v", err, namer.BuildUniqueKey(), influxdbQuery.Prometheus.Query)
		return nil, err
	}
	return timeSeries, nil
}