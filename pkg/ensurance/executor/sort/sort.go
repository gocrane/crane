package sort

import (
	"sort"

	"k8s.io/klog/v2"

	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/pod-info"
)

// RankFunc sorts the pods
type RankFunc func(pods []podinfo.PodContext)

var sortFunc = map[string]func(p1, p2 podinfo.PodContext) int32{
	"ExtCpuBeUsed":     extCpuBeUsed,
	"ClassAndPriority": classAndPriority,
	"ExtCpuUsage":      extCpuUsage,
	"CpuUsage":         cpuUsage,
	"RunningTime":      runningTime,
}

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
		rankFunc = CpuUsageSorter
	}

	return rankFunc
}

// runningTime compares pods by pod's start time
func runningTime(p1, p2 podinfo.PodContext) int32 {
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

// classAndPriority compares pods by pod's ClassAndPriority
func classAndPriority(p1, p2 podinfo.PodContext) int32 {
	return podinfo.CompareClassAndPriority(p1.ClassAndPriority, p2.ClassAndPriority)
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
