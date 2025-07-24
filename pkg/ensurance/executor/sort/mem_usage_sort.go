package sort

import (
	"github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
	"github.com/gocrane/crane/pkg/utils"
)

func MemUsageSort(pods []podinfo.PodContext) {
	orderedBy(UseElasticMem, ComparePriority, ComparePodQOSClass, CompareMemUsage, CompareElasticMem, CompareRunningTime).Sort(pods)
}

// UseElasticMem compares pod by using ext resource whether
func UseElasticMem(p1, p2 podinfo.PodContext) int32 {
	use1 := utils.Bool2Uint(p1.ElasticMemLimit != 0)
	use2 := utils.Bool2Uint(p2.ElasticMemLimit != 0)

	return int32(use2 - use1)
}

// CompareMemUsage compares the partition mem usage of mem limit
func CompareMemUsage(p1, p2 podinfo.PodContext) int32 {
	return utils.CmpFloat(p2.PodMemUsage, p1.PodMemUsage)
}

// CompareElasticMem compares the partition of extmem usage to extmem limit
func CompareElasticMem(p1, p2 podinfo.PodContext) int32 {
	// if both pod don't use ext resource, then return
	if p1.ElasticMemLimit == 0 && p2.ElasticMemLimit == 0 {
		return 0
	}

	p1Ratio := p1.PodMemUsage / float64(p1.ElasticMemLimit)
	p2Ratio := p2.PodMemUsage / float64(p2.ElasticMemLimit)

	return utils.CmpFloat(p1Ratio, p2Ratio)
}
