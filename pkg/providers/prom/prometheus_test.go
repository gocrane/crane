package prom

//import (
//	"testing"
//	"time"
//
//	"github.com/gocrane/crane/pkg/metricnaming"
//	"github.com/gocrane/crane/pkg/metricquery"
//	"github.com/gocrane/crane/pkg/providers"
//)
//
//func TestNewProvider(t *testing.T) {
//	config := &providers.PromConfig{
//		Address:                     "http://120.53.133.232:8080/",
//		QueryConcurrency:            10,
//		BRateLimit:                  false,
//		MaxPointsLimitPerTimeSeries: 11000,
//	}
//
//	dataSource, err := NewProvider(config)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	namer := &metricnaming.GeneralMetricNamer{
//		Metric: &metricquery.Metric{
//			Type: metricquery.PromQLMetricType,
//			Prom: &metricquery.PromNamerInfo{
//				QueryExpr: `sum (irate (container_cpu_usage_seconds_total{container!="",image!="",container!="POD",namespace="default",pod=~"^dep-1-100m-500mib-.*$"}[5m]))`,
//			},
//		},
//	}
//
//	end := time.Now().Truncate(time.Minute)
//	start := end.Add(-15 * 24 * time.Hour)
//	step := time.Minute
//	tsList, err := dataSource.QueryTimeSeries(namer, start, end, step)
//	if err != nil {
//		t.Fatal(err)
//	}
//	PrintTsList(tsList)
//}
