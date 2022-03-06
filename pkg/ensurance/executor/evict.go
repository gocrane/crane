package executor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

type EvictExecutor struct {
	EvictPods EvictPods
}

type EvictPod struct {
	DeletionGracePeriodSeconds *int32
	PodKey                     types.NamespacedName
	ClassAndPriority           ClassAndPriority
}

type EvictPods []EvictPod

func (e EvictPods) Len() int      { return len(e) }
func (e EvictPods) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e EvictPods) Less(i, j int) bool {
	return e[i].ClassAndPriority.Less(e[j].ClassAndPriority)
}

func (e EvictPods) Find(key types.NamespacedName) int {
	for i, v := range e {
		if v.PodKey == key {
			return i
		}
	}

	return -1
}

func (e *EvictExecutor) Avoid(ctx *ExecuteContext) error {
	var start = time.Now()
	metrics.UpdateLastTimeWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentEvict), metrics.StepAvoid, start)
	defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentEvict), metrics.StepAvoid, start)

	klog.V(6).Infof("EvictExecutor avoid, %v", *e)

	if len(e.EvictPods) == 0 {
		metrics.UpdateExecutorStatus(metrics.SubComponentEvict, metrics.StepAvoid, 0.0)
		return nil
	}

	metrics.UpdateExecutorStatus(metrics.SubComponentEvict, metrics.StepAvoid, 1.0)
	metrics.ExecutorStatusCounterInc(metrics.SubComponentEvict, metrics.StepAvoid)

	var bSucceed = true
	var errPodKeys []string

	wg := sync.WaitGroup{}

	for i := range e.EvictPods {
		wg.Add(1)

		go func(evictPod EvictPod) {
			defer wg.Done()

			pod, err := ctx.PodLister.Pods(evictPod.PodKey.Namespace).Get(evictPod.PodKey.Name)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, "not found ", evictPod.PodKey.String())
				return
			}

			err = utils.EvictPodWithGracePeriod(ctx.Client, pod, evictPod.DeletionGracePeriodSeconds)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, "evict failed ", evictPod.PodKey.String())
				klog.Warningf("Failed to evict pod %s: %v", evictPod.PodKey.String(), err)
				return
			}

			metrics.ExecutorEvictCountsInc()

			klog.V(4).Infof("Pod %s is evicted", klog.KObj(pod))
		}(e.EvictPods[i])
	}

	wg.Wait()

	if !bSucceed {
		return fmt.Errorf("some pod evict failed,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}

func (e *EvictExecutor) Restore(ctx *ExecuteContext) error {
	return nil
}
