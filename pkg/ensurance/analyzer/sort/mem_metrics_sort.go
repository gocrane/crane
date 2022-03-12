package sort

import "github.com/gocrane/crane/pkg/ensurance/executor"

// Todo: Memory metrics related sort func need to be filled

func MemMetricsSorter(pods []executor.PodContext) {
	orderedBy(classAndPriority, runningTime).Sort(pods)
}
