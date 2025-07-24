package resource

import (
	"fmt"
	"strings"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	stypes "github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
	cgrpc "github.com/gocrane/crane/pkg/ensurance/grpc"
	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

type PodResourceManager struct {
	nodeName string
	client   clientset.Interface

	podLister corelisters.PodLister
	podSynced cache.InformerSynced

	runtimeClient pb.RuntimeServiceClient
	runtimeConn   *grpc.ClientConn
	stateChann    chan map[string][]common.TimeSeries

	// A copy of data from stateChann
	state map[string][]common.TimeSeries
	// Updated when get new data from stateChann, used to determine whether state has expired
	lastStateTime time.Time

	cadvisor.Manager
}

func NewPodResourceManager(client clientset.Interface, nodeName string, podInformer coreinformers.PodInformer,
	runtimeEndpoint string, stateChann chan map[string][]common.TimeSeries, cadvisorManager cadvisor.Manager) *PodResourceManager {
	runtimeClient, runtimeConn, err := cruntime.GetRuntimeClient(runtimeEndpoint)
	if err != nil {
		klog.Errorf("GetRuntimeClient failed %s", err.Error())
		return nil
	}

	o := &PodResourceManager{
		nodeName:      nodeName,
		client:        client,
		podLister:     podInformer.Lister(),
		podSynced:     podInformer.Informer().HasSynced,
		runtimeClient: runtimeClient,
		runtimeConn:   runtimeConn,
		stateChann:    stateChann,
		Manager:       cadvisorManager,
	}
	podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		// Focused on pod belonged to this node
		FilterFunc: func(obj interface{}) bool {
			switch t := obj.(type) {
			case *v1.Pod:
				return ownedPod(t, o.nodeName)
			default:
				utilruntime.HandleError(fmt.Errorf("unable to handle object %T", obj))
				return false
			}
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: o.reconcilePod,
			UpdateFunc: func(old, cur interface{}) {
				o.reconcilePod(cur)
			},
			DeleteFunc: o.reconcilePod,
		},
	})
	return o
}

func (o *PodResourceManager) Name() string {
	return "PodResourceManager"
}

func (o *PodResourceManager) Run(stop <-chan struct{}) {
	klog.Infof("Starting pod resource manager.")

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("pod-resource-manager",
		stop,
		o.podSynced,
	) {
		return
	}

	go func() {
		for {
			select {
			case state := <-o.stateChann:
				o.state = state
				o.lastStateTime = time.Now()
			case <-stop:
				klog.Infof("Pod resource manager exit")
				if err := cgrpc.CloseGrpcConnection(o.runtimeConn); err != nil {
					klog.Errorf("Failed to close grpc connection: %v", err)
				}
				return
			}
		}
	}()

	return
}

func (o *PodResourceManager) reconcilePod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
			return
		}
		pod, ok = tombstone.Obj.(*v1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a pod %#v", obj))
			return
		}
	}

	o.updatePodExtResToCgroup(pod)
}

func ownedPod(pod *v1.Pod, nodeName string) bool {
	return pod.Spec.NodeName == nodeName && pod.Status.Phase != v1.PodSucceeded && pod.Status.Phase != v1.PodFailed
}

// Get pod's gocrane.io resource and update it to Cgroup
func (o *PodResourceManager) updatePodExtResToCgroup(pod *v1.Pod) {
	start := time.Now()
	metrics.UpdateLastTime(string(known.ModulePodResourceManager), metrics.StepUpdatePodResource, start)

	_, containerCPUQuotas := podinfo.GetPodUsage(string(stypes.MetricNameContainerCpuQuota), o.state, pod)

	for _, c := range pod.Spec.Containers {
		if state := utils.GetContainerStatus(pod, c); state.Running == nil {
			klog.V(4).Infof("container %s is not running, skip it", c.Name)
			return
		}

		for res, val := range c.Resources.Limits {
			if strings.HasPrefix(res.String(), fmt.Sprintf(utils.ExtResourcePrefixFormat, v1.ResourceCPU)) {
				containerId := utils.GetContainerIdFromPod(pod, c.Name)
				if containerId == "" {
					continue
				}

				// If container's quota is -1, pod resource manager will convert limit to quota
				containerCPUQuota, err := podinfo.GetUsageById(containerCPUQuotas, containerId)
				if err != nil {
					klog.Error(err)
				}
				if !utils.AlmostEqual(containerCPUQuota.Value, -1.0) && !utils.AlmostEqual(containerCPUQuota.Value, 0) {
					continue
				}

				containerPeriod := o.getCPUPeriod(pod, containerId)
				if containerPeriod == 0 {
					continue
				}

				// Update cpu quota by CRI
				err = cruntime.UpdateContainerResources(o.runtimeClient, containerId, cruntime.UpdateOptions{CPUQuota: int64(float64(val.MilliValue()) / executor.CpuQuotaCoefficient * containerPeriod)})
				if err != nil {
					metrics.PodResourceUpdateErrorCounterInc(metrics.SubComponentPodResource, metrics.StepUpdateQuota)
					klog.Errorf("Failed to update pod %s container %s Resource, err %s", pod.Name, containerId, err.Error())
					continue
				}
			}
		}
	}
	metrics.UpdateDurationFromStart(string(known.ModulePodResourceManager), metrics.StepUpdatePodResource, start)
}

// Get cpu period from local state is not expired;
// Otherwise, get value from CRI
func (o *PodResourceManager) getCPUPeriod(pod *v1.Pod, containerId string) float64 {
	now := time.Now()

	if o.state != nil && !now.After(o.lastStateTime.Add(StateExpiration)) {
		_, containerCPUPeriods := podinfo.GetPodUsage(string(stypes.MetricNameContainerCpuPeriod), o.state, pod)
		for _, period := range containerCPUPeriods {
			if period.ContainerId == containerId {
				return period.Value
			}
		}
	}

	// Use CRI to get cpu period directly
	var query = info.ContainerInfoRequest{}
	containerInfoV1, err := o.Manager.GetContainerInfo(containerId, &query)
	if err != nil {
		metrics.PodResourceUpdateErrorCounterInc(metrics.SubComponentPodResource, metrics.StepGetPeriod)
		klog.Errorf("ContainerInfoRequest failed for container %s: %v ", containerId, err)
		return 0.0
	}
	return float64(containerInfoV1.Spec.Cpu.Period)
}
