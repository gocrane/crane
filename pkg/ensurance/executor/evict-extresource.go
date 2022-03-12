package executor

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gocrane/crane/pkg/utils"
	"k8s.io/klog/v2"
)

type EvictExtResourceExecutor struct {
	EvictPods EvictPods
}

func (e *EvictExtResourceExecutor) Avoid(ctx *ExecuteContext) error {
	klog.V(6).Infof("EvictExtResourceExecutor avoid, %v", *e)

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

			err = utils.EvictPodForExtResource(ctx.Client, pod)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, "evict-extresource failed ", evictPod.PodKey.String())
				klog.Warningf("Failed to evict pod %s for extresource: %v", evictPod.PodKey.String(), err)
				return
			}

			klog.V(4).Infof("Pod %s is evicted for extresource", klog.KObj(pod))
		}(e.EvictPods[i])
	}

	wg.Wait()

	if !bSucceed {
		return fmt.Errorf("some pod evict failed for extresource,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}

func (e *EvictExtResourceExecutor) Restore(ctx *ExecuteContext) error {
	return nil
}
