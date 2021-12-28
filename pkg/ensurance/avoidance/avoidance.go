package avoidance

import (
	"google.golang.org/grpc"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/ensurance/executor"
	cgrpc "github.com/gocrane/crane/pkg/ensurance/grpc"
	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
)

type AvoidanceManager struct {
	nodeName string
	client   clientset.Interface
	noticeCh <-chan executor.AvoidanceExecutor

	podLister corelisters.PodLister
	podSynced cache.InformerSynced

	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced

	runtimeClient pb.RuntimeServiceClient
	runtimeConn   *grpc.ClientConn
}

// NewAvoidanceManager create avoidance manager
func NewAvoidanceManager(client clientset.Interface, nodeName string, podInformer coreinformers.PodInformer, nodeInformer coreinformers.NodeInformer,
	noticeCh <-chan executor.AvoidanceExecutor, runtimeEndpoint string) *AvoidanceManager {

	runtimeClient, runtimeConn, err := cruntime.GetRuntimeClient(runtimeEndpoint, true)
	if err != nil {
		klog.Errorf("GetRuntimeClient failed %s", err.Error())
		return nil
	}

	return &AvoidanceManager{
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

func (a *AvoidanceManager) Name() string {
	return "AvoidanceManager"
}

// Run does nothing
func (a *AvoidanceManager) Run(stop <-chan struct{}) {
	klog.Infof("Starting avoid manager.")

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("avoidance-manager",
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
				if err := a.doAction(as, stop); err != nil {
					// TODO: if it failed in action, how to retry
					klog.Errorf("Failed to doAction: %v", err)
				}
			case <-stop:
				{
					klog.Infof("Avoidance exit")
					cgrpc.CloseGrpcConnection(a.runtimeConn)
					return
				}
			}
		}
	}()

	return
}

func (a *AvoidanceManager) doAction(ae executor.AvoidanceExecutor, _ <-chan struct{}) error {
	var ctx = &executor.ExecuteContext{
		NodeName:      a.nodeName,
		Client:        a.client,
		PodLister:     a.podLister,
		NodeLister:    a.nodeLister,
		RuntimeClient: a.runtimeClient,
		RuntimeConn:   a.runtimeConn,
	}

	//step1 do avoidance actions
	if err := doAvoidance(ctx, ae); err != nil {
		return err
	}

	//step2 do restoration actions
	if err := doRestoration(ctx, ae); err != nil {
		return err
	}

	return nil
}

func doAvoidance(ctx *executor.ExecuteContext, ae executor.AvoidanceExecutor) error {

	//step1 do DisableScheduled action
	if err := ae.ScheduledExecutor.Avoid(ctx); err != nil {
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

func doRestoration(ctx *executor.ExecuteContext, ae executor.AvoidanceExecutor) error {
	//step1 do DisableScheduled action
	if err := ae.ScheduledExecutor.Restore(ctx); err != nil {
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
