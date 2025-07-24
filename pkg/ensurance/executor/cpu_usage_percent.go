package executor

import (
	"sync"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
	"github.com/gocrane/crane/pkg/ensurance/executor/sort"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

func init() {
	registerMetricMap(cpuUsagePercent)
}

var cpuUsagePercent = metric{
	Name:           CpuUsagePercent,
	ActionPriority: 5,
	Sortable:       true,
	SortFunc:       sort.CpuUsageSort,

	Throttleable:       false,
	ThrottleQuantified: false,
	ThrottleFunc:       nil,
	RestoreFunc:        nil,

	Evictable:       true,
	EvictQuantified: true,
	EvictFunc:       cpuUsagePercentEvictPod,
}

func cpuUsagePercentEvictPod(wg *sync.WaitGroup, ctx *ExecuteContext, index int, totalReleasedResource *ReleaseResource, EvictPods EvictPods) (errPodKeys []string, released ReleaseResource) {
	wg.Add(1)

	// Calculate release resources
	released = releaseCPUUsagePercent(EvictPods[index])
	totalReleasedResource.Add(released)

	go func(evictPod podinfo.PodContext) {
		defer wg.Done()

		pod, err := ctx.PodLister.Pods(evictPod.Key.Namespace).Get(evictPod.Key.Name)
		if err != nil {
			errPodKeys = append(errPodKeys, "not found ", evictPod.Key.String())
			return
		}
		klog.Warningf("Evicting pod %v", evictPod.Key)
		err = utils.EvictPodWithGracePeriod(ctx.Client, pod, evictPod.DeletionGracePeriodSeconds)
		if err != nil {
			errPodKeys = append(errPodKeys, "evict failed ", evictPod.Key.String())
			klog.Warningf("Failed to evict pod %s: %v", evictPod.Key.String(), err)
			return
		}
		metrics.ExecutorEvictCountsInc()

		klog.Warningf("Pod %s is evicted", klog.KObj(pod))
	}(EvictPods[index])
	return
}

func releaseCPUUsagePercent(pod podinfo.PodContext) ReleaseResource {
	if pod.ActionType == podinfo.Evict {
		return ReleaseResource{
			CpuUsagePercent: pod.PodCPUUsage * CpuQuotaCoefficient,
		}
	}
	return ReleaseResource{}
}
