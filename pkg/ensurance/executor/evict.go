package executor

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	einformer "github.com/gocrane/crane/pkg/ensurance/informer"
	"github.com/gocrane/crane/pkg/utils/log"
)

const (
	DefaultDeletionGracePeriodSeconds = 30
)

type EvictExecutor struct {
	Executors EvictPods
}

type EvictPod struct {
	DeletionGracePeriodSeconds int32
	PodTypes                   types.NamespacedName
	PodQOSPriority             ScheduledQOSPriority
}

type EvictPods []EvictPod

func (e EvictPods) Len() int      { return len(e) }
func (e EvictPods) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e EvictPods) Less(i, j int) bool {
	return e[i].PodQOSPriority.Less(e[j].PodQOSPriority)
}

func (e EvictPods) Find(podTypes types.NamespacedName) int {
	for i, v := range e {
		if v.PodTypes == podTypes {
			return i
		}
	}

	return -1
}

func (e *EvictExecutor) Avoid(ctx *ExecuteContext) error {
	log.Logger().V(4).Info("Avoid", "EvictExecutor", *e)

	var bSucceed = true
	var errPodKeys []string

	wg := sync.WaitGroup{}

	for i := range e.Executors {
		wg.Add(1)

		go func(v EvictPod) {
			defer wg.Done()

			pod, err := einformer.GetPodFromInformer(ctx.PodInformer, v.PodTypes.String())
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, "not found ", v.PodTypes.String())
				return
			}

			log.Logger().V(4).Info(fmt.Sprintf("Pod %s", log.GenerateObj(pod)))
			err = einformer.EvictPodWithGracePeriod(ctx.Client, pod, v.DeletionGracePeriodSeconds)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, "evict failed ", v.PodTypes.String())
				log.Logger().V(4).Info(fmt.Sprintf("Warning: evict failed %s, err %s", v.PodTypes.String(), err.Error()))
				return
			}
		}(e.Executors[i])
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
