package replicas

import (
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

var _ recommender.Recommender = &ReplicasRecommender{}

type ReplicasRecommender struct {
	base.BaseRecommender
	WorkloadMinReplicas  int64
	PodMinReadySeconds   int64
	PodAvailableRatio    float64
	CpuPercentile        float64
	MemPercentile        float64
	DefaultMinReplicas   int64
	CPUTargetUtilization float64
	MemTargetUtilization float64
}

func (rr *ReplicasRecommender) Name() string {
	return recommender.ReplicasRecommender
}

// NewReplicasRecommender create a new replicas recommender.
func NewReplicasRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (*ReplicasRecommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)

	workloadMinReplicasInt, err := recommender.GetConfigInt("workload-min-replicas", 1)
	if err != nil {
		return nil, err
	}

	podMinReadySeconds, err := recommender.GetConfigInt("pod-min-ready-seconds", 30)
	if err != nil {
		return nil, err
	}

	podAvailableRatio, err := recommender.GetConfigFloat("pod-available-ratio", 0.5)
	if err != nil {
		return nil, err
	}

	cpuPercentileFloat, err := recommender.GetConfigFloat("cpu-percentile", 0.95)
	if err != nil {
		return nil, err
	}
	cpuPercentileFloat = cpuPercentileFloat * 100

	memPercentileFloat, err := recommender.GetConfigFloat("mem-percentile", 0.95)
	if err != nil {
		return nil, err
	}
	memPercentileFloat = memPercentileFloat * 100

	defaultMinReplicasInt, err := recommender.GetConfigInt("default-min-replicas", 1)
	if err != nil {
		return nil, err
	}

	cpuTargetUtilizationFloat, err := recommender.GetConfigFloat("cpu-target-utilization", 0.5)
	if err != nil {
		return nil, err
	}

	memTargetUtilizationFloat, err := recommender.GetConfigFloat("mem-target-utilization", 0.5)
	if err != nil {
		return nil, err
	}

	return &ReplicasRecommender{
		*base.NewBaseRecommender(recommender),
		workloadMinReplicasInt,
		podMinReadySeconds,
		podAvailableRatio,
		cpuPercentileFloat,
		memPercentileFloat,
		defaultMinReplicasInt,
		cpuTargetUtilizationFloat,
		memTargetUtilizationFloat,
	}, nil
}
