package prom

//import (
//	"testing"
//	"time"
//
//	"github.com/gocrane/crane/pkg/providers"
//
//)
//
//func TestNewProvider(t *testing.T) {
//	config := &providers.PromConfig{
//		Address: "http://81.70.126.254",
//		QueryConcurrency: 10,
//		BRateLimit: false,
//		MaxPointsLimitPerTimeSeries: 11000,
//	}
//
//	dataSource, err := NewProvider(config)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	end := time.Now().Truncate(time.Minute)
//	start := end.Add(-15 * 24 * time.Hour)
//	step := time.Minute
//	tsList, err := dataSource.QueryTimeSeries(`sum (irate (container_cpu_usage_seconds_total{container!="",image!="",container!="POD",namespace="default",pod=~"^dep-1-100m-500mib-.*$"}[5m]))`, start, end, step)
//	if err != nil {
//		t.Fatal(err)
//	}
//	PrintTsList(tsList)
//}
