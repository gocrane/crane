package executor

import (
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type Executor interface {
	Avoid(ctx *ExecuteContext) error
	Restore(ctx *ExecuteContext) error
}

type AvoidanceExecutor struct {
	ScheduledExecutor ScheduledExecutor
	ThrottleExecutor  ThrottleExecutor
	EvictExecutor     EvictExecutor
}

type ExecuteContext struct {
	NodeName   string
	Client     clientset.Interface
	PodLister  corelisters.PodLister
	NodeLister corelisters.NodeLister
}
