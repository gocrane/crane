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
	WorkloadMinReplicas int64
	PodMinReadySeconds  int64
	PodAvailableRatio   float64
	CpuPercentile       float64
	DefaultMinReplicas  int64
	TargetUtilization   int64
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

	availbleRatio, exists := recommender.Config["pod-available-ratio"]
	if !exists {
		availbleRatio = "0.5"
	}

	podAvailableRatio, err := strconv.ParseFloat(availbleRatio, 64)
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

	defaultMinReplicas, exists := recommender.Config["default-min-replicas"]
	if !exists {
		defaultMinReplicas = "1"
	}

	defaultMinReplicasInt, err := strconv.ParseInt(defaultMinReplicas, 10, 32)
	if err != nil {
		return nil, err
	}

	targetUtilization, exists := recommender.Config["cpu-target-utilization"]
	if !exists {
		targetUtilization = "50"
	}

	targetUtilizationInt, err := strconv.ParseInt(targetUtilization, 10, 32)
	if err != nil {
		return nil, err
	}

	return &ReplicasRecommender{
		*base.NewBaseRecommender(recommender),
		workloadMinReplicasInt,
		podMinReadySeconds,
		podAvailableRatio,
		cpuPercentileFloat,
		defaultMinReplicasInt,
		targetUtilizationInt,
	}, nil
}
