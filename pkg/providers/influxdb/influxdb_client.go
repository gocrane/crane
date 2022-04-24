package influxdb

import (
	"github.com/gocrane/crane/pkg/providers"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// NewPrometheusClient returns a prometheus.Client
func NewInfluxDBClient(config *providers.InfluxDBConfig) (influxdb2.Client, error) {
	client := influxdb2.NewClient(config.Url, config.Token)

	return client, nil
}