package executor

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/utils"
)

const (
	DefaultDeletionGracePeriodSeconds = 30
)

type EvictExecutor struct {
	EvictPods EvictPods
}

type EvictPod struct {
	DeletionGracePeriodSeconds int32
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
	klog.V(6).Infof("EvictExecutor avoid, %v", *e)

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
