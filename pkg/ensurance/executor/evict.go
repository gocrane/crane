package executor

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	einformer "github.com/gocrane/crane/pkg/ensurance/informer"
	"github.com/gocrane/crane/pkg/utils/clogs"
)

type EvictExecutor struct {
	Executors []Evict
}

type Evict struct {
	DeletionGracePeriodSeconds *int32
	EvictPods                  []types.NamespacedName
}

func (e *EvictExecutor) Avoid(ctx *ExecuteContext) error {
	var bSucceed bool

	for _, v := range e.Executors {
		for _, podNamespace := range v.EvictPods {
			pod, err := einformer.GetPodFromInformer(ctx.PodInformer, podNamespace.String())
			if err != nil {
				bSucceed = false
				continue
			}
			clogs.Log().V(5).Info("Pod %+v", pod)
			//go einformer.EvictPodWithGracePeriod(a.client,pod,einformer.GetGracePeriodSeconds(e.DeletionGracePeriodSeconds))
		}
	}

	if !bSucceed {
		return fmt.Errorf("some pod evict failed")
	}

	return nil
}

func (e *EvictExecutor) Restore(ctx *ExecuteContext) error {
	return nil
}
