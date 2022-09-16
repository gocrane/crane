package resource

import (
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
	"strconv"
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
func NewIdleNodeRecommender(recommender apis.Recommender) (*IdleNodeRecommender, error) {
	if recommender.Config == nil {
		recommender.Config = map[string]string{}
	}

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
