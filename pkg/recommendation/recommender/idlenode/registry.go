package idlenode

import (
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

const (
	cpuRequestUtilizationKey    = "cpu-request-utilization"
	cpuUsageUtilizationKey      = "cpu-usage-utilization"
	cpuPercentileKey            = "cpu-percentile"
	memoryRequestUtilizationKey = "memory-request-utilization"
	memoryUsageUtilizationKey   = "memory-usage-utilization"
	memoryPercentileKey         = "memory-percentile"
)

var _ recommender.Recommender = &IdleNodeRecommender{}

type IdleNodeRecommender struct {
	base.BaseRecommender
	cpuRequestUtilization    float64
	cpuUsageUtilization      float64
	cpuPercentile            float64
	memoryRequestUtilization float64
	memoryUsageUtilization   float64
	memoryPercentile         float64
}

func init() {
	recommender.RegisterRecommenderProvider(recommender.IdleNodeRecommender, NewIdleNodeRecommender)
}

func (inr *IdleNodeRecommender) Name() string {
	return recommender.IdleNodeRecommender
}

// NewIdleNodeRecommender create a new idle node recommender.
func NewIdleNodeRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (recommender.Recommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)

	cpuRequestUtilization, err := recommender.GetConfigFloat(cpuRequestUtilizationKey, 0)
	if err != nil {
		return nil, err
	}

	cpuUsageUtilization, err := recommender.GetConfigFloat(cpuUsageUtilizationKey, 0)
	if err != nil {
		return nil, err
	}
	cpuPercentile, err := recommender.GetConfigFloat(cpuPercentileKey, 0.99)
	if err != nil {
		return nil, err
	}
	cpuPercentile = cpuPercentile * 100

	memoryRequestUtilization, err := recommender.GetConfigFloat(memoryRequestUtilizationKey, 0)
	if err != nil {
		return nil, err
	}

	memoryUsageUtilization, err := recommender.GetConfigFloat(memoryUsageUtilizationKey, 0)
	if err != nil {
		return nil, err
	}
	memoryPercentile, err := recommender.GetConfigFloat(memoryPercentileKey, 0.99)
	if err != nil {
		return nil, err
	}
	memoryPercentile = memoryPercentile * 100

	return &IdleNodeRecommender{
		*base.NewBaseRecommender(recommender),
		cpuRequestUtilization,
		cpuUsageUtilization,
		cpuPercentile,
		memoryRequestUtilization,
		memoryUsageUtilization,
		memoryPercentile,
	}, err
}
