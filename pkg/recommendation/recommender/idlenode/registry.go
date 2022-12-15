package idlenode

import (
	"strconv"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

var _ recommender.Recommender = &IdleNodeRecommender{}

type IdleNodeRecommender struct {
	base.BaseRecommender
	cpuRequestUtilization float64
	memRequestUtilization float64
}

func (inr *IdleNodeRecommender) Name() string {
	return recommender.IdleNodeRecommender
}

// NewIdleNodeRecommender create a new idle node recommender.
func NewIdleNodeRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (*IdleNodeRecommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)

	cpuRequestUtilization, exists := recommender.Config["cpu-request-utilization"]
	if !exists {
		cpuRequestUtilization = "0"
	}

	cpuRequestUtilizationFloat, err := strconv.ParseFloat(cpuRequestUtilization, 64)
	if err != nil {
		return nil, err
	}

	memRequestUtilization, exists := recommender.Config["mem-request-utilization"]
	if !exists {
		memRequestUtilization = "0"
	}

	memRequestUtilizationFloat, err := strconv.ParseFloat(memRequestUtilization, 64)
	if err != nil {
		return nil, err
	}

	return &IdleNodeRecommender{
		*base.NewBaseRecommender(recommender),
		cpuRequestUtilizationFloat,
		memRequestUtilizationFloat,
	}, err
}
