package resource

import (
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

var _ recommender.Recommender = &ResourceRecommender{}

type ResourceRecommender struct {
	base.BaseRecommender
	CpuSampleInterval        string
	CpuRequestPercentile     string
	CpuRequestMarginFraction string
	CpuTargetUtilization     string
	CpuModelHistoryLength    string
	MemSampleInterval        string
	MemPercentile            string
	MemMarginFraction        string
	MemTargetUtilization     string
	MemHistoryLength         string
}

func (rr *ResourceRecommender) Name() string {
	return recommender.ResourceRecommender
}

// NewResourceRecommender create a new resource recommender.
func NewResourceRecommender(recommender apis.Recommender) *ResourceRecommender {
	if recommender.Config == nil {
		recommender.Config = map[string]string{}
	}

	cpuSampleInterval, exists := recommender.Config["cpu-sample-interval"]
	if !exists {
		cpuSampleInterval = "1m"
	}
	cpuPercentile, exists := recommender.Config["cpu-request-percentile"]
	if !exists {
		cpuPercentile = "0.99"
	}
	cpuMarginFraction, exists := recommender.Config["cpu-request-margin-fraction"]
	if !exists {
		cpuMarginFraction = "0.15"
	}
	cpuTargetUtilization, exists := recommender.Config["cpu-target-utilization"]
	if !exists {
		cpuTargetUtilization = "1.0"
	}
	cpuHistoryLength, exists := recommender.Config["cpu-model-history-length"]
	if !exists {
		cpuHistoryLength = "168h"
	}

	memSampleInterval, exists := recommender.Config["mem-sample-interval"]
	if !exists {
		memSampleInterval = "1m"
	}
	memPercentile, exists := recommender.Config["mem-request-percentile"]
	if !exists {
		memPercentile = "0.99"
	}
	memMarginFraction, exists := recommender.Config["mem-request-margin-fraction"]
	if !exists {
		memMarginFraction = "0.15"
	}
	memTargetUtilization, exists := recommender.Config["mem-target-utilization"]
	if !exists {
		memTargetUtilization = "1.0"
	}
	memHistoryLength, exists := recommender.Config["mem-model-history-length"]
	if !exists {
		memHistoryLength = "168h"
	}

	return &ResourceRecommender{
		*base.NewBaseRecommender(recommender),
		cpuSampleInterval,
		cpuPercentile,
		cpuMarginFraction,
		cpuTargetUtilization,
		cpuHistoryLength,
		memSampleInterval,
		memPercentile,
		memMarginFraction,
		memTargetUtilization,
		memHistoryLength,
	}
}
