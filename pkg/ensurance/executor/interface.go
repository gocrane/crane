package executor

import (
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Executor interface {
	Avoid(ctx *ExecuteContext) error
	Restore(ctx *ExecuteContext) error
}

type AvoidanceExecutor struct {
	BlockScheduledExecutor BlockScheduledExecutor
	ThrottleExecutor       ThrottleExecutor
	EvictExecutor          EvictExecutor
}

type ExecuteContext struct {
	NodeName     string
	Client       clientset.Interface
	NodeInformer cache.SharedIndexInformer
	PodInformer  cache.SharedIndexInformer
}
