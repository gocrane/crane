package sort

import (
	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
	"github.com/gocrane/crane/pkg/utils"
)

func CpuUsageSort(pods []podinfo.PodContext) {
	// todo, need ut to make sure all cases
	orderedBy(UseElasticCPU, ComparePriority, ComparePodQOSClass, CompareCPUUsage, CompareElasticCPU, CompareRunningTime).Sort(pods)
}

// CompareElasticCPU compares the partition of extcpu usage to extcpu limit
func CompareElasticCPU(p1, p2 podinfo.PodContext) int32 {
	// if both pod don't use ext resource, then return
	if p1.ElasticCPU == 0 && p2.ElasticCPU == 0 {
		return 0
	}

	p1Ratio := p1.PodCPUUsage / float64(p1.ElasticCPU)
	p2Ratio := p2.PodCPUUsage / float64(p2.ElasticCPU)

	return utils.CmpFloat(p1Ratio, p2Ratio)
}

// CompareCPUUsage compares the partition cpu usage of cpu limit
func CompareCPUUsage(p1, p2 podinfo.PodContext) int32 {
	return utils.CmpFloat(p2.PodCPUUsage, p1.PodCPUUsage)
}

// UseElasticCPU compares pod by using ext resource whether
func UseElasticCPU(p1, p2 podinfo.PodContext) int32 {
	use1 := utils.Bool2Uint(p1.ElasticCPU != 0)
	use2 := utils.Bool2Uint(p2.ElasticCPU != 0)

	return int32(use2 - use1)
}
