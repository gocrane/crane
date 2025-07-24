package prom

import (
	"testing"

	"github.com/gocrane/crane/pkg/providers"
)

func TestNewPrometheusClient(t *testing.T) {
	config := &providers.PromConfig{
		Address:                     "",
		QueryConcurrency:            10,
		BRateLimit:                  true,
		MaxPointsLimitPerTimeSeries: 11000,
	}
	_, err := NewPrometheusClient(config)
	if err != nil {
		t.Fatal(err)
	}

}
