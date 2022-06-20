package sort

import podinfo "github.com/gocrane/crane/pkg/ensurance/executor/pod-info"

func GeneralSorter(pods []podinfo.PodContext) {
	orderedBy(classAndPriority, runningTime).Sort(pods)
}
