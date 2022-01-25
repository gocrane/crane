package executor

import (
	"google.golang.org/grpc"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Executor interface {
	Avoid(ctx *ExecuteContext) error
	Restore(ctx *ExecuteContext) error
}

type AvoidanceExecutor struct {
	ScheduleExecutor ScheduleExecutor
	ThrottleExecutor ThrottleExecutor
	EvictExecutor    EvictExecutor
}

type ExecuteContext struct {
	NodeName      string
	Client        clientset.Interface
	PodLister     corelisters.PodLister
	NodeLister    corelisters.NodeLister
	RuntimeClient pb.RuntimeServiceClient
	RuntimeConn   *grpc.ClientConn
}
