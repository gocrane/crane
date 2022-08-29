package sort

import (
	"sort"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
)

type RankFunc func(pods []podinfo.PodContext)

var sortFunc = map[string]func(p1, p2 podinfo.PodContext) int32{
	"UseElasticResource": UseElasticCPU,
	"PodQOSClass":        ComparePodQOSClass,
	"ExtCpuUsage":        CompareElasticCPU,
	"CpuUsage":           CompareCPUUsage,
	"RunningTime":        CompareRunningTime,
}

// RankFuncConstruct is a sample for future extends, keep it even it is not called
func RankFuncConstruct(customize []string) RankFunc {
	if len(customize) == 0 {
		klog.Fatal("If customize sort func is defined, it can't be empty.")
	}
	var rankFunc RankFunc
	if len(customize) != 0 {
		cmp := []cmpFunc{}
		for _, f := range customize {
			if f, ok := sortFunc[f]; ok {
				cmp = append(cmp, f)
			}
			rankFunc = orderedBy(cmp...).Sort
		}
	} else {
		rankFunc = CpuUsageSort
	}

	return rankFunc
}

// CompareRunningTime compares pods by pod's start time
func CompareRunningTime(p1, p2 podinfo.PodContext) int32 {
	t1 := p1.StartTime
	t2 := p2.StartTime

	if t1.Before(t2) {
		return 1
	} else if t1.Equal(t2) {
		return 0
	}

	if t2 == nil {
		return 1
	}
	return -1
}

// ComparePodQOSClass compares pods by pod's QOSClass
func ComparePodQOSClass(p1, p2 podinfo.PodContext) int32 {
	return ComparePodQosClass(p1.QOSClass, p2.QOSClass)
}

func ComparePriority(p1, p2 podinfo.PodContext) int32 {
	if p1.Priority == p2.Priority {
		return 0
	} else if p1.Priority < p2.Priority {
		return -1
	}
	return 1
}

// ComparePodQosClass compares Pod QOSClass
// Guaranteed > Burstable > BestEffort
func ComparePodQosClass(a v1.PodQOSClass, b v1.PodQOSClass) int32 {
	switch b {
	case v1.PodQOSGuaranteed:
		if a == v1.PodQOSGuaranteed {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBurstable:
		if a == v1.PodQOSGuaranteed {
			return 1
		} else if a == v1.PodQOSBurstable {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBestEffort:
		if (a == v1.PodQOSGuaranteed) || (a == v1.PodQOSBurstable) {
			return 1
		} else if a == v1.PodQOSBestEffort {
			return 0
		} else {
			return -1
		}
	default:
		if (a == v1.PodQOSGuaranteed) || (a == v1.PodQOSBurstable) || (a == v1.PodQOSBestEffort) {
			return 1
		} else {
			return 0
		}
	}
}

// Cmp compares p1 and p2 and returns:
//
//   -1 if p1 <  p2
//    0 if p1 == p2
//   +1 if p1 >  p2
//
type cmpFunc func(p1, p2 podinfo.PodContext) int32

// podSorter implements the Sort interface, sorting changes within.
type podSorter struct {
	pods []podinfo.PodContext
	cmp  []cmpFunc
}

// Sort sorts the argument slice according to the less functions passed to orderedBy.
func (ms *podSorter) Sort(pods []podinfo.PodContext) {
	ms.pods = pods
	sort.Sort(ms)
}

// orderedBy returns a Sorter that sorts using the cmp functions, in order.
// Call its Sort method to sort the data.
func orderedBy(cmp ...cmpFunc) *podSorter {
	return &podSorter{
		cmp: cmp,
	}
}

// Len is part of sort.Interface.
func (ms *podSorter) Len() int {
	return len(ms.pods)
}

// Swap is part of sort.Interface.
func (ms *podSorter) Swap(i, j int) {
	ms.pods[i], ms.pods[j] = ms.pods[j], ms.pods[i]
}

// Less is part of sort.Interface.
func (ms *podSorter) Less(i, j int) bool {
	p1, p2 := ms.pods[i], ms.pods[j]
	var k int
	for k = 0; k < len(ms.cmp)-1; k++ {
		cmpResult := ms.cmp[k](p1, p2)
		// p1 is less than p2
		if cmpResult < 0 {
			return true
		}
		// p1 is greater than p2
		if cmpResult > 0 {
			return false
		}
		// we don't know yet
	}
	// the last cmp func is the final decider
	return ms.cmp[k](p1, p2) < 0
}
