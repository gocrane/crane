package hpa

import (
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/replicas"
)

var _ recommender.Recommender = &HPARecommender{}

type HPARecommender struct {
	replicas.ReplicasRecommender
	PredictableEnabled      bool
	ReferenceHpaEnabled     bool
	MinCpuUsageThreshold    float64
	FluctuationThreshold    float64
	MinCpuTargetUtilization int64
	MaxCpuTargetUtilization int64
	MaxReplicasFactor       float64
}

func (rr *HPARecommender) Name() string {
	return recommender.HPARecommender
}

// NewHPARecommender create a new hpa recommender.
func NewHPARecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (*HPARecommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)

	predictableEnabled, err := recommender.GetConfigBool("predictable", false)
	if err != nil {
		return nil, err
	}

	referenceHpaEnabled, err := recommender.GetConfigBool("reference-hpa", true)
	if err != nil {
		return nil, err
	}

	minCpuUsageThresholdFloat, err := recommender.GetConfigFloat("min-cpu-usage-threshold", 1)
	if err != nil {
		return nil, err
	}

	fluctuationThresholdFloat, err := recommender.GetConfigFloat("fluctuation-threshold", 1.5)
	if err != nil {
		return nil, err
	}

	minCpuTargetUtilizationInt, err := recommender.GetConfigInt("min-cpu-target-utilization", 30)
	if err != nil {
		return nil, err
	}

	maxCpuTargetUtilizationInt, err := recommender.GetConfigInt("max-cpu-target-utilization", 75)
	if err != nil {
		return nil, err
	}

	maxReplicasFactorFloat, err := recommender.GetConfigFloat("max-replicas-factor", 3)
	if err != nil {
		return nil, err
	}

	replicasRecommender, err := replicas.NewReplicasRecommender(recommender, recommendationRule)
	if err != nil {
		return nil, err
	}

	return &HPARecommender{
		*replicasRecommender,
		predictableEnabled,
		referenceHpaEnabled,
		minCpuUsageThresholdFloat,
		fluctuationThresholdFloat,
		minCpuTargetUtilizationInt,
		maxCpuTargetUtilizationInt,
		maxReplicasFactorFloat,
	}, nil
}
