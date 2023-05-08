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
	memoryRequestUtilizationKey = "memory-request-utilization"
	memoryUsageUtilizationKey   = "memory-usage-utilization"
)

var _ recommender.Recommender = &IdleNodeRecommender{}

type IdleNodeRecommender struct {
	base.BaseRecommender
	cpuRequestUtilization    float64
	cpuUsageUtilization      float64
	memoryRequestUtilization float64
	memoryUsageUtilization   float64
}

func (inr *IdleNodeRecommender) Name() string {
	return recommender.IdleNodeRecommender
}

// NewIdleNodeRecommender create a new idle node recommender.
func NewIdleNodeRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (*IdleNodeRecommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)

	cpuRequestUtilization, err := recommender.GetConfigFloat(cpuRequestUtilizationKey, 0)
	if err != nil {
		return nil, err
	}

	cpuUsageUtilization, err := recommender.GetConfigFloat(cpuUsageUtilizationKey, 0)
	if err != nil {
		return nil, err
	}

	memoryRequestUtilization, err := recommender.GetConfigFloat(memoryRequestUtilizationKey, 0)
	if err != nil {
		return nil, err
	}

	memoryUsageUtilization, err := recommender.GetConfigFloat(memoryUsageUtilizationKey, 0)
	if err != nil {
		return nil, err
	}

	return &IdleNodeRecommender{
		*base.NewBaseRecommender(recommender),
		cpuRequestUtilization,
		cpuUsageUtilization,
		memoryRequestUtilization,
		memoryUsageUtilization,
	}, err
}
