package avoidance

import (
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/utils/log"
)

type AvoidanceManager struct {
	nodeName          string
	client            clientset.Interface
	noticeCh          <-chan executor.AvoidanceExecutor
	podInformer       cache.SharedIndexInformer
	nodeInformer      cache.SharedIndexInformer
	avoidanceInformer cache.SharedIndexInformer
}

// AvoidanceManager create avoidance manager
func NewAvoidanceManager(client clientset.Interface, nodeName string, podInformer cache.SharedIndexInformer, nodeInformer cache.SharedIndexInformer,
	avoidanceInformer cache.SharedIndexInformer, noticeCh <-chan executor.AvoidanceExecutor) *AvoidanceManager {
	return &AvoidanceManager{
		nodeName:          nodeName,
		client:            client,
		noticeCh:          noticeCh,
		podInformer:       podInformer,
		nodeInformer:      nodeInformer,
		avoidanceInformer: avoidanceInformer,
	}
}

func (a *AvoidanceManager) Name() string {
	return "AvoidanceManager"
}

// Run does nothing
func (a *AvoidanceManager) Run(stop <-chan struct{}) {
	log.Logger().V(2).Info("Avoidance manager starts running")

	go func() {
		for {
			select {
			case as := <-a.noticeCh:
				if err := a.doAction(as, stop); err != nil {
					// TODO: if it failed in action, how to retry
					log.Logger().Error(err, "doAction failed")
				}
			case <-stop:
				{
					log.Logger().V(2).Info("Avoidance exit")
					return
				}
			}
		}
	}()

	return
}

func (a *AvoidanceManager) doAction(ae executor.AvoidanceExecutor, _ <-chan struct{}) error {

	var ctx = &executor.ExecuteContext{
		NodeName:     a.nodeName,
		Client:       a.client,
		PodInformer:  a.podInformer,
		NodeInformer: a.nodeInformer,
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
