package avoidance

import (
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/ensurance/executor"
)

type AvoidanceManager struct {
	nodeName string
	client   clientset.Interface
	noticeCh <-chan executor.AvoidanceExecutor

	podLister corelisters.PodLister
	podSynced cache.InformerSynced

	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced
}

// NewAvoidanceManager create avoidance manager
func NewAvoidanceManager(client clientset.Interface, nodeName string, podInformer coreinformers.PodInformer, nodeInformer coreinformers.NodeInformer,
	noticeCh <-chan executor.AvoidanceExecutor) *AvoidanceManager {
	return &AvoidanceManager{
		nodeName:   nodeName,
		client:     client,
		noticeCh:   noticeCh,
		podLister:  podInformer.Lister(),
		podSynced:  podInformer.Informer().HasSynced,
		nodeLister: nodeInformer.Lister(),
		nodeSynced: nodeInformer.Informer().HasSynced,
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
					return
				}
			}
		}
	}()

	return
}

func (a *AvoidanceManager) doAction(ae executor.AvoidanceExecutor, _ <-chan struct{}) error {
	var ctx = &executor.ExecuteContext{
		NodeName:   a.nodeName,
		Client:     a.client,
		PodLister:  a.podLister,
		NodeLister: a.nodeLister,
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
