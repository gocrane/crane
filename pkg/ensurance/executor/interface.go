package executor

import (
	"google.golang.org/grpc"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"github.com/gocrane/crane/pkg/common"
)

type Executor interface {
	Avoid(ctx *ExecuteContext) error
	Restore(ctx *ExecuteContext) error
}

type AvoidanceExecutor struct {
	ScheduleExecutor ScheduleExecutor
	ThrottleExecutor ThrottleExecutor
	EvictExecutor    EvictExecutor
	StateMap         map[string][]common.TimeSeries
}

type ExecuteContext struct {
	NodeName      string
	Client        clientset.Interface
	PodLister     corelisters.PodLister
	NodeLister    corelisters.NodeLister
	RuntimeClient pb.RuntimeServiceClient
	RuntimeConn   *grpc.ClientConn

	// Gap for metrics Evictable/ThrottleAble
	// Key is the metric name, value is (actual used)-(the lowest watermark for NodeQOSEnsurancePolicies which use throttleDown action)
	ThrottoleDownGapToWatermarks GapToWatermarks
	// Key is the metric name, value is (actual used)-(the lowest watermark for NodeQOSEnsurancePolicies which use throttleUp action)
	ThrottoleUpGapToWatermarks GapToWatermarks
	// key is the metric name, value is (actual used)-(the lowest watermark for NodeQOSEnsurancePolicies which use evict action)
	EvictGapToWatermarks GapToWatermarks

	stateMap map[string][]common.TimeSeries

	executeExcessPercent float64
}
