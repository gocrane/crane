package resource

import (
	"fmt"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
	"sort"
	"strings"
)

var _ recommender.Recommender = &ResourceRecommender{}

type ResourceSpec struct {
	CPU    string
	Memory string
}
type ResourceRecommender struct {
	base.BaseRecommender
	CpuSampleInterval        string
	CpuRequestPercentile     string
	CpuRequestMarginFraction string
	CpuTargetUtilization     string
	CpuModelHistoryLength    string
	MemSampleInterval        string
	MemPercentile            string
	MemMarginFraction        string
	MemTargetUtilization     string
	MemHistoryLength         string
	ResourceSpecs            []ResourceSpec
}

func (rr *ResourceRecommender) Name() string {
	return recommender.ResourceRecommender
}

// NewResourceRecommender create a new resource recommender.
func NewResourceRecommender(recommender apis.Recommender) (*ResourceRecommender, error) {
	if recommender.Config == nil {
		recommender.Config = map[string]string{}
	}

	cpuSampleInterval, exists := recommender.Config["cpu-sample-interval"]
	if !exists {
		cpuSampleInterval = "1m"
	}
	cpuPercentile, exists := recommender.Config["cpu-request-percentile"]
	if !exists {
		cpuPercentile = "0.99"
	}
	cpuMarginFraction, exists := recommender.Config["cpu-request-margin-fraction"]
	if !exists {
		cpuMarginFraction = "0.15"
	}
	cpuTargetUtilization, exists := recommender.Config["cpu-target-utilization"]
	if !exists {
		cpuTargetUtilization = "1.0"
	}
	cpuHistoryLength, exists := recommender.Config["cpu-model-history-length"]
	if !exists {
		cpuHistoryLength = "168h"
	}

	memSampleInterval, exists := recommender.Config["mem-sample-interval"]
	if !exists {
		memSampleInterval = "1m"
	}
	memPercentile, exists := recommender.Config["mem-request-percentile"]
	if !exists {
		memPercentile = "0.99"
	}
	memMarginFraction, exists := recommender.Config["mem-request-margin-fraction"]
	if !exists {
		memMarginFraction = "0.15"
	}
	memTargetUtilization, exists := recommender.Config["mem-target-utilization"]
	if !exists {
		memTargetUtilization = "1.0"
	}
	memHistoryLength, exists := recommender.Config["mem-model-history-length"]
	if !exists {
		memHistoryLength = "168h"
	}
	//
	resourceSpecification, exists := recommender.Config["resource-specification"]
	if !exists {
		resourceSpecification = ""
	}
	// format specs
	specs, err := t(resourceSpecification)
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
		specs,
	}, nil
}

func t(sc string) ([]ResourceSpec, error) {
	var ResourceSpecs []ResourceSpec
	//s := "5c11g,4c8g,4c5g"
	////先把2c4g, 2c8g,4c4g,4c8g 切割
	arr := strings.Split(sc, ",")
	sort.Strings(arr)
	fmt.Println(arr)
	for i := 0; i < len(arr); i++ {
		//需要用正则
		arr1 := strings.Split(arr[i], "")
		//fmt.Println(arr1[0:])
		ResourceSpecs1 := &ResourceSpec{
			CPU:    arr1[0],
			Memory: arr1[2],
		}

		ResourceSpecs = append(ResourceSpecs, *ResourceSpecs1)
		//fmt.Println(ResourceSpecs)
		//fmt.Println(arr1)
	}
	return ResourceSpecs, nil
}
