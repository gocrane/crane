package sort

import "github.com/gocrane/crane/pkg/ensurance/executor/podinfo"

func GeneralSorter(pods []podinfo.PodContext) {
	orderedBy(ComparePriority, ComparePodQOSClass, CompareRunningTime).Sort(pods)
}
