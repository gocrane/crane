package executor

import (
	"google.golang.org/grpc"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"

	cgrpc "github.com/gocrane/crane/pkg/ensurance/grpc"
	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
)

type ActionExecutor struct {
	nodeName string
	client   clientset.Interface
	noticeCh <-chan AvoidanceExecutor

	podLister  corelisters.PodLister
	nodeLister corelisters.NodeLister
	podSynced  cache.InformerSynced
	nodeSynced cache.InformerSynced

	runtimeClient pb.RuntimeServiceClient
	runtimeConn   *grpc.ClientConn
}

// NewActionExecutor create enforcer manager
func NewActionExecutor(client clientset.Interface, nodeName string, podInformer coreinformers.PodInformer, nodeInformer coreinformers.NodeInformer,
	noticeCh <-chan AvoidanceExecutor, runtimeEndpoint string) *ActionExecutor {

	runtimeClient, runtimeConn, err := cruntime.GetRuntimeClient(runtimeEndpoint, true)
	if err != nil {
		klog.Errorf("GetRuntimeClient failed %s", err.Error())
		return nil
	}

	return &ActionExecutor{
		nodeName:      nodeName,
		client:        client,
		noticeCh:      noticeCh,
		podLister:     podInformer.Lister(),
		podSynced:     podInformer.Informer().HasSynced,
		nodeLister:    nodeInformer.Lister(),
		nodeSynced:    nodeInformer.Informer().HasSynced,
		runtimeClient: runtimeClient,
		runtimeConn:   runtimeConn,
	}
}

func (a *ActionExecutor) Name() string {
	return "ActionExecutor"
}

func (a *ActionExecutor) Run(stop <-chan struct{}) {
	klog.Infof("Starting action executor.")

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("action-executor",
		stop,
		a.podSynced,
		a.nodeSynced,
	) {
		return
	}

	go func() {
		for {
			select {
			case as := <-a.noticeCh:
				if err := a.execute(as, stop); err != nil {
					// TODO: if it failed in action, how to retry
					klog.Errorf("Failed to execute action: %v", err)
				}
			case <-stop:
				{
					klog.Infof("Exiting action executor.")
					if err := cgrpc.CloseGrpcConnection(a.runtimeConn); err != nil {
						klog.Errorf("Failed to close grpc connection: %v", err)
					}
					return
				}
			}
		}
	}()

	return
}

func (a *ActionExecutor) execute(ae AvoidanceExecutor, _ <-chan struct{}) error {
	var ctx = &ExecuteContext{
		NodeName:      a.nodeName,
		Client:        a.client,
		PodLister:     a.podLister,
		NodeLister:    a.nodeLister,
		RuntimeClient: a.runtimeClient,
		RuntimeConn:   a.runtimeConn,
	}

	//step1 do enforcer actions
	if err := avoid(ctx, ae); err != nil {
		return err
	}

	//step2 do restoration actions
	if err := restore(ctx, ae); err != nil {
		return err
	}

	return nil
}

func avoid(ctx *ExecuteContext, ae AvoidanceExecutor) error {

	//step1 do DisableScheduled action
	if err := ae.ScheduleExecutor.Avoid(ctx); err != nil {
		return err
	}

	//step2 do Evict action
	if err := ae.EvictExecutor.Avoid(ctx); err != nil {
		return err
	}

	//step3 do Throttle action
	if err := ae.ThrottleExecutor.Avoid(ctx); err != nil {
		return err
	}

	return nil
}

func restore(ctx *ExecuteContext, ae AvoidanceExecutor) error {
	//step1 do DisableScheduled action
	if err := ae.ScheduleExecutor.Restore(ctx); err != nil {
		return err
	}

	//step2 do Evict action
	if err := ae.EvictExecutor.Restore(ctx); err != nil {
		return err
	}

	//step3 do Throttle action
	if err := ae.ThrottleExecutor.Restore(ctx); err != nil {
		return err
	}

	return nil
}
