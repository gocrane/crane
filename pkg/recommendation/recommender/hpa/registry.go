package hpa

import (
	"strconv"

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

	predictable, exists := recommender.Config["predictable"]
	if !exists {
		predictable = "false"
	}
	predictableEnabled, err := strconv.ParseBool(predictable)
	if err != nil {
		return nil, err
	}

	referenceHPA, exists := recommender.Config["reference-hpa"]
	if !exists {
		referenceHPA = "true"
	}
	referenceHpaEnabled, err := strconv.ParseBool(referenceHPA)
	if err != nil {
		return nil, err
	}

	minCpuUsageThreshold, exists := recommender.Config["min-cpu-usage-threshold"]
	if !exists {
		minCpuUsageThreshold = "1"
	}
	minCpuUsageThresholdFloat, err := strconv.ParseFloat(minCpuUsageThreshold, 64)
	if err != nil {
		return nil, err
	}

	fluctuationThreshold, exists := recommender.Config["fluctuation-threshold"]
	if !exists {
		fluctuationThreshold = "1.5"
	}
	fluctuationThresholdFloat, err := strconv.ParseFloat(fluctuationThreshold, 64)
	if err != nil {
		return nil, err
	}

	minCpuTargetUtilization, exists := recommender.Config["min-cpu-target-utilization"]
	if !exists {
		minCpuTargetUtilization = "30"
	}
	minCpuTargetUtilizationInt, err := strconv.ParseInt(minCpuTargetUtilization, 10, 32)
	if err != nil {
		return nil, err
	}

	maxCpuTargetUtilization, exists := recommender.Config["max-cpu-target-utilization"]
	if !exists {
		maxCpuTargetUtilization = "75"
	}
	maxCpuTargetUtilizationInt, err := strconv.ParseInt(maxCpuTargetUtilization, 10, 32)
	if err != nil {
		return nil, err
	}

	maxReplicasFactor, exists := recommender.Config["max-replicas-factor"]
	if !exists {
		maxReplicasFactor = "3"
	}
	maxReplicasFactorFloat, err := strconv.ParseFloat(maxReplicasFactor, 64)
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
