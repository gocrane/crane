package sort

import "github.com/gocrane/crane/pkg/ensurance/executor/podinfo"

// Todo: Memory metrics related sort func need to be filled

func MemMetricsSorter(pods []podinfo.PodContext) {
	orderedBy(ComparePriority, ComparePodQOSClass, CompareRunningTime).Sort(pods)
}
