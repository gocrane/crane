package replicas

import (
	"strconv"

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
func NewReplicasRecommender(recommender apis.Recommender) (*ReplicasRecommender, error) {
	workloadMinReplicas, exists := recommender.Config["workload-min-replicas"]
	if !exists {
		workloadMinReplicas = "1"
	}

	workloadMinReplicasInt, err := strconv.ParseInt(workloadMinReplicas, 10, 32)
	if err != nil {
		return nil, err
	}

	minReadySeconds, exists := recommender.Config["pod-min-ready-seconds"]
	if !exists {
		minReadySeconds = "30"
	}

	podMinReadySeconds, err := strconv.ParseInt(minReadySeconds, 10, 32)
	if err != nil {
		return nil, err
	}

	availableRatio, exists := recommender.Config["pod-available-ratio"]
	if !exists {
		availableRatio = "0.5"
	}

	podAvailableRatio, err := strconv.ParseFloat(availableRatio, 64)
	if err != nil {
		return nil, err
	}

	cpuPercentile, exists := recommender.Config["cpu-percentile"]
	if !exists {
		cpuPercentile = "0.95"
	}

	cpuPercentileFloat, err := strconv.ParseFloat(cpuPercentile, 64)
	if err != nil {
		return nil, err
	}
	cpuPercentileFloat = cpuPercentileFloat * 100

	memPercentile, exists := recommender.Config["mem-percentile"]
	if !exists {
		memPercentile = "0.95"
	}

	memPercentileFloat, err := strconv.ParseFloat(memPercentile, 64)
	if err != nil {
		return nil, err
	}
	memPercentileFloat = memPercentileFloat * 100

	defaultMinReplicas, exists := recommender.Config["default-min-replicas"]
	if !exists {
		defaultMinReplicas = "1"
	}

	defaultMinReplicasInt, err := strconv.ParseInt(defaultMinReplicas, 10, 32)
	if err != nil {
		return nil, err
	}

	cpuTargetUtilization, exists := recommender.Config["cpu-target-utilization"]
	if !exists {
		cpuTargetUtilization = "0.5"
	}

	cpuTargetUtilizationFloat, err := strconv.ParseFloat(cpuTargetUtilization, 64)
	if err != nil {
		return nil, err
	}

	memTargetUtilization, exists := recommender.Config["mem-target-utilization"]
	if !exists {
		memTargetUtilization = "0.5"
	}

	memTargetUtilizationFloat, err := strconv.ParseFloat(memTargetUtilization, 64)
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
