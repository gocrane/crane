package resource

import (
	"time"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

var _ recommender.Recommender = &ResourceRecommender{}

type ResourceRecommender struct {
	base.BaseRecommender
	CpuSampleInterval                string
	CpuRequestPercentile             string
	CpuRequestMarginFraction         string
	CpuTargetUtilization             string
	CpuModelHistoryLength            string
	MemSampleInterval                string
	MemPercentile                    string
	MemMarginFraction                string
	MemTargetUtilization             string
	MemHistoryLength                 string
	OOMProtection                    bool
	OOMHistoryLength                 time.Duration
	OOMBumpRatio                     float64
	Specification                    bool
	SpecificationConfigs             []Specification
	CpuHistogramBucketSize           string
	CpuHistogramMaxValue             string
	MemHistogramBucketSize           string
	MemHistogramMaxValue             string
	HistoryCompletionCheck           bool
	ResourceComplianceRecommendation bool
}

func init() {
	recommender.RegisterRecommenderProvider(recommender.ResourceRecommender, NewResourceRecommender)
}

func (rr *ResourceRecommender) Name() string {
	return recommender.ResourceRecommender
}

// NewResourceRecommender create a new resource recommender.
func NewResourceRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (recommender.Recommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)

	cpuSampleInterval := recommender.GetConfigString("cpu-sample-interval", "1m")
	cpuPercentile := recommender.GetConfigString("cpu-request-percentile", "0.99")
	cpuMarginFraction := recommender.GetConfigString("cpu-request-margin-fraction", "0.15")
	cpuTargetUtilization := recommender.GetConfigString("cpu-target-utilization", "1.0")
	cpuHistoryLength := recommender.GetConfigString("cpu-model-history-length", "168h")
	memSampleInterval := recommender.GetConfigString("mem-sample-interval", "1m")
	memPercentile := recommender.GetConfigString("mem-request-percentile", "0.99")
	memMarginFraction := recommender.GetConfigString("mem-request-margin-fraction", "0.15")
	memTargetUtilization := recommender.GetConfigString("mem-target-utilization", "1.0")
	memHistoryLength := recommender.GetConfigString("mem-model-history-length", "168h")

	specificationBool, err := recommender.GetConfigBool("specification", false)
	if err != nil {
		return nil, err
	}

	specificationConfig := recommender.GetConfigString("specification-config", DefaultSpecs)
	specifications, err := GetResourceSpecifications(specificationConfig)
	if err != nil {
		return nil, err
	}

	oomProtectionBool, err := recommender.GetConfigBool("oom-protection", true)
	if err != nil {
		return nil, err
	}

	oomHistoryLengthDuration, err := recommender.GetConfigDuration("oom-history-length", 168*time.Hour)
	if err != nil {
		return nil, err
	}

	OOMBumpRatioFloat, err := recommender.GetConfigFloat("oom-bump-ratio", 1.2)
	if err != nil {
		return nil, err
	}

	cpuHistogramBucketSize := recommender.GetConfigString("cpu-histogram-bucket-size", "0.1")
	cpuHistogramMaxValue := recommender.GetConfigString("cpu-histogram-max-value", "100")
	memHistogramBucketSize := recommender.GetConfigString("mem-histogram-bucket-size", "104857600")
	memHistogramMaxValue := recommender.GetConfigString("mem-histogram-max-value", "104857600000")

	historyCompletion, err := recommender.GetConfigBool("history-completion-check", false)
	if err != nil {
		return nil, err
	}
	resourceComplianceRecommendation, err := recommender.GetConfigBool("resource-compliance-recommendation-check", false)
	if err != nil {
		return nil, err
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
		oomProtectionBool,
		oomHistoryLengthDuration,
		OOMBumpRatioFloat,
		specificationBool,
		specifications,
		cpuHistogramBucketSize,
		cpuHistogramMaxValue,
		memHistogramBucketSize,
		memHistogramMaxValue,
		historyCompletion,
		resourceComplianceRecommendation,
	}, nil
}
