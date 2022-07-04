package sort

import (
	v1 "k8s.io/api/core/v1"

	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/pod-info"
	"github.com/gocrane/crane/pkg/utils"
)

func CpuUsageSorter(pods []podinfo.PodContext) {
	orderedBy(classAndPriority, cpuUsage, extCpuUsage, runningTime).Sort(pods)
}

// extCpuUsage compares the partition of extcpu usage to extcpu limit
func extCpuUsage(p1, p2 podinfo.PodContext) int32 {
	// if both pod don't use ext resource, then return
	if p1.ExtCpuBeUsed == false && p2.ExtCpuBeUsed == false {
		return 0
	}

	p1Ratio := p1.PodCPUUsage / float64(p1.ExtCpuLimit)
	p2Ratio := p2.PodCPUUsage / float64(p2.ExtCpuLimit)

	return utils.CmpFloat(p1Ratio, p2Ratio)
}

// cpuUsage compares the partition extcpu usage of extcpu limit
func cpuUsage(p1, p2 podinfo.PodContext) int32 {
	var p1usage, p2usage float64
	// if both pod is PodQOSBestEffort, then compare the absolute usage;otherwise, cmpare the ratio compared with PodCPUQuota
	if p1.ClassAndPriority.PodQOSClass == v1.PodQOSBestEffort && p2.ClassAndPriority.PodQOSClass == v1.PodQOSBestEffort {
		p1usage = p1.PodCPUUsage
		p2usage = p2.PodCPUUsage
	} else {
		p1usage = p1.PodCPUUsage * p1.PodCPUPeriod / p1.PodCPUQuota
		p2usage = p2.PodCPUUsage * p2.PodCPUPeriod / p2.PodCPUQuota
	}
	return utils.CmpFloat(p1usage, p2usage)
}

// extCpuBeUsed compares pod by using ext resource whether
func extCpuBeUsed(p1, p2 podinfo.PodContext) int32 {
	use1 := utils.Bool2Uint(p1.ExtCpuBeUsed)
	use2 := utils.Bool2Uint(p2.ExtCpuBeUsed)

	return int32(use1 - use2)
}
