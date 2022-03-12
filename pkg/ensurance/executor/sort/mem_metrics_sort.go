package sort

import podinfo "github.com/gocrane/crane/pkg/ensurance/executor/pod-info"

// Todo: Memory metrics related sort func need to be filled

func MemMetricsSorter(pods []podinfo.PodContext) {
	orderedBy(classAndPriority, runningTime).Sort(pods)
}
