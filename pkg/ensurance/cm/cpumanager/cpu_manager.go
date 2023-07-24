package cpumanager

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	criapis "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/containermap"
	cpumanagerstate "k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"

	topologyinformer "github.com/gocrane/api/pkg/generated/informers/externalversions/topology/v1alpha1"
	topologylisters "github.com/gocrane/api/pkg/generated/listers/topology/v1alpha1"
	topologyapi "github.com/gocrane/api/topology/v1alpha1"

	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/utils"
)

// ActivePodsFunc is a function that returns a list of pods to reconcile.
type ActivePodsFunc func() ([]*corev1.Pod, error)

// ActivePodsByPolicyFunc is a function that returns a list of pods which belong to specified policy.
type ActivePodsByPolicyFunc func(policy string) ([]*corev1.Pod, error)

const (
	cpuManagerName          = "CPUManager"
	cpuManagerStateFileName = "crane_cpu_manager_state"
	cpuPolicyKeyIndex       = "cpuPolicy"
)

var DefaultExclusiveCPUSet = func() cpuset.CPUSet {
	return cpuset.NewCPUSet()
}

type CPUManager interface {
	manager.Manager

	GetExclusiveCPUSet() cpuset.CPUSet

	GetSharedCPUs() cpuset.CPUSet
}

type cpuManager struct {
	sync.Mutex
	nodeName  string
	policy    Policy
	workqueue workqueue.RateLimitingInterface
	podLister corelisters.PodLister
	nrtLister topologylisters.NodeResourceTopologyLister
	podSync   cache.InformerSynced
	nrtSync   cache.InformerSynced

	// defaultCPUPolicy is the default cpu policy for a pod if policy is not specified.
	defaultCPUPolicy string

	// reconcilePeriod is the duration between calls to reconcileState.
	reconcilePeriod time.Duration

	// state allows pluggable CPU assignment policies while sharing a common
	// representation of state for the system to inspect and reconcile.
	state cpumanagerstate.State

	// lastUpdatedstate holds state for each container from the last time it was updated.
	lastUpdateState cpumanagerstate.State

	// containerRuntime is the container runtime service interface needed
	// to make UpdateContainerResources() calls against the containers.
	containerRuntime criapis.RuntimeService

	// activePods is a method for listing active pods on the node
	// so all the containers can be updated in the reconciliation loop.
	activePods ActivePodsFunc

	// containerMap provides a mapping from (pod, container) -> containerID
	// for all containers whose cpuset is updated in reconcileState.
	containerMap containermap.ContainerMap
}

func NewCPUManager(
	nodeName string,
	defaultCPUPolicy string,
	reconcilePeriod time.Duration,
	cadvisorManager cadvisor.Manager,
	containerRuntime criapis.RuntimeService,
	stateFileDirectory string,
	podInformer coreinformers.PodInformer,
	nrtInformer topologyinformer.NodeResourceTopologyInformer,
) (CPUManager, error) {
	machineInfo, err := cadvisorManager.GetMachineInfo()
	if err != nil {
		return nil, err
	}
	topo, err := topology.Discover(machineInfo)
	if err != nil {
		return nil, err
	}
	klog.InfoS("Detected CPU topology", "topology", topo)

	initialContainers := buildContainerMapFromRuntime(containerRuntime)
	if err != nil {
		return nil, fmt.Errorf("failed to build map of initial containers from runtime: %v", err)
	}

	cm := &cpuManager{
		nodeName:         nodeName,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "cpumanager"),
		podLister:        podInformer.Lister(),
		nrtLister:        nrtInformer.Lister(),
		podSync:          podInformer.Informer().HasSynced,
		nrtSync:          nrtInformer.Informer().HasSynced,
		defaultCPUPolicy: defaultCPUPolicy,
		reconcilePeriod:  reconcilePeriod,
		lastUpdateState:  cpumanagerstate.NewMemoryState(),
		containerRuntime: containerRuntime,
		containerMap:     containermap.NewContainerMap(),
	}

	_ = podInformer.Informer().AddIndexers(cache.Indexers{
		cpuPolicyKeyIndex: func(obj interface{}) ([]string, error) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return []string{}, nil
			}
			policyName := cm.getPodCPUPolicyOrDefault(pod)
			return []string{policyName}, nil
		},
	})

	podIndexer := podInformer.Informer().GetIndexer()
	getPodFunc := func(cpuPolicy string) ([]*corev1.Pod, error) {
		objs, err := podIndexer.ByIndex(cpuPolicyKeyIndex, cpuPolicy)
		if err != nil {
			return nil, err
		}
		pods := make([]*corev1.Pod, 0, len(objs))
		for _, obj := range objs {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				continue
			}
			// Succeeded and failed pods are not considered because they don't occupy any resource.
			// See https://github.com/kubernetes/kubernetes/blob/f61ed439882e34d9dad28b602afdc852feb2337a/pkg/scheduler/scheduler.go#L756-L763
			if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
				pods = append(pods, pod)
			}
		}
		return pods, nil
	}

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: cm.enqueuePod,
	})

	cm.activePods = func() ([]*corev1.Pod, error) {
		allPods, err := cm.podLister.List(labels.Everything())
		if err != nil {
			return nil, err
		}
		activePods := make([]*corev1.Pod, 0, len(allPods))
		for _, pod := range allPods {
			if !utils.IsPodTerminated(pod) {
				activePods = append(activePods, pod)
			}
		}
		return activePods, nil
	}

	cm.policy = NewStaticPolicy(topo, getPodFunc)

	cm.state, err = cpumanagerstate.NewCheckpointState(
		stateFileDirectory,
		cpuManagerStateFileName,
		cm.policy.Name(),
		initialContainers,
	)
	if err != nil {
		klog.ErrorS(err, "Could not initialize checkpoint state, please remove crane cpu policy state file and restart crane agent")
		return nil, err
	}

	return cm, nil
}

func (cm *cpuManager) Name() string {
	return cpuManagerName
}

func (cm *cpuManager) Run(stopCh <-chan struct{}) {
	klog.InfoS("Starting CPU manager", "policy", cm.policy.Name())
	klog.InfoS("Reconciling", "reconcilePeriod", cm.reconcilePeriod)

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if !cache.WaitForCacheSync(stopCh, cm.podSync, cm.nrtSync) {
		return
	}

	go wait.Until(cm.runWorker, time.Second, stopCh)

	nodeTopologyResult, err := cm.getNodeTopologyResult()
	if err != nil {
		klog.Fatalf("Failed to get reserved cpus: %v", err)
	}

	if err := cm.policy.Start(cm.state, nodeTopologyResult); err != nil {
		klog.Fatalf("Failed to start cpumanager policy: %v", err)
	}

	// Periodically call m.reconcileState() to continue to keep the CPU sets of
	// all pods in sync with and guaranteed CPUs handed out among them.
	go wait.Until(func() {
		nrt, err := cm.nrtLister.Get(cm.nodeName)
		if err != nil || nrt.CraneManagerPolicy.CPUManagerPolicy != topologyapi.CPUManagerPolicyStatic {
			return
		}
		cm.reconcileState()
	}, cm.reconcilePeriod, stopCh)
}

func (cm *cpuManager) Allocate(pod *corev1.Pod, container *corev1.Container, mode string, tr TopologyResult) error {
	// Get the list of active pods.
	activePods, err := cm.activePods()
	if err != nil {
		return err
	}

	cm.renewState(activePods)

	cm.Lock()
	defer cm.Unlock()

	// Call down into the policy to assign this container CPUs if required.
	if err = cm.policy.Allocate(cm.state, pod, container, mode, tr); err != nil {
		klog.ErrorS(err, "Allocate error", "pod", klog.KObj(pod), "container", container.Name)
		return err
	}

	return nil
}

func (cm *cpuManager) State() cpumanagerstate.Reader {
	return cm.state
}

func (cm *cpuManager) GetExclusiveCPUSet() cpuset.CPUSet {
	return cm.policy.GetExclusiveCPUSet(cm.state)
}

func (cm *cpuManager) GetSharedCPUs() cpuset.CPUSet {
	return cm.policy.GetSharedCPUs(cm.state)
}

func (cm *cpuManager) updateContainerCPUSet(containerID string, cpus cpuset.CPUSet) error {
	return cm.containerRuntime.UpdateContainerResources(
		containerID,
		&runtimeapi.LinuxContainerResources{
			CpusetCpus: cpus.String(),
		})
}

func (cm *cpuManager) renewState(activePods []*corev1.Pod) {
	// We grab the lock to ensure that no new containers will grab CPUs while
	// executing the code below. Without this lock, its possible that we end up
	// removing state that is newly added by an asynchronous call to
	// AddContainer() during the execution of this code.
	cm.Lock()
	defer cm.Unlock()

	// Build a list of (podUID, containerName) pairs for all containers in all active Pods.
	activeContainers := make(map[string]sets.String)
	for _, pod := range activePods {
		activeContainers[string(pod.UID)] = sets.NewString()
		for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
			activeContainers[string(pod.UID)].Insert(container.Name)
		}
	}

	// Loop through the CPUManager state. Remove any state for containers not
	// in the `activeContainers` list built above.
	var (
		assignments                  = cm.state.GetCPUAssignments()
		defaultCPUSet                = cm.state.GetDefaultCPUSet()
		hasExclusiveContainerRemoved = false
	)

	for podUID := range assignments {
		for containerName := range assignments[podUID] {
			if _, ok := activeContainers[podUID][containerName]; !ok {
				klog.InfoS("Remove stale container state", "podUID", podUID, "containerName", containerName)
				// exclusive
				if defaultCPUSet.Intersection(assignments[podUID][containerName]).IsEmpty() {
					hasExclusiveContainerRemoved = true
				}
				cm.policyRemoveContainerByRef(podUID, containerName)
			}
		}
	}

	/* TODO(Garrybest): remove inactive container from containerMap to avoid memory leak
	   Need to update dependency to 1.24, https://github.com/kubernetes/kubernetes/pull/109103
	*/

	if hasExclusiveContainerRemoved {
		for _, pod := range activePods {
			// If shared, remove result and let cpumanager bind cpus again.
			if cm.getPodCPUPolicyOrDefault(pod) == topologyapi.AnnotationPodCPUPolicyNUMA {
				klog.InfoS("Remove stale container state due to exclusive container destroyed", "podUID", pod.UID)
				for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
					cm.policyRemoveContainerByRef(string(pod.UID), container.Name)
				}
				// After removing the cpuset for numa policy pod, enqueue this pod again to reallocate the cpuset.
				cm.enqueuePod(pod)
			}
		}
	}
}

func (cm *cpuManager) policyRemoveContainerByRef(podUID string, containerName string) {
	cm.policy.RemoveContainer(cm.state, podUID, containerName)
	cm.lastUpdateState.Delete(podUID, containerName)
	cm.containerMap.RemoveByContainerRef(podUID, containerName)
}

func (cm *cpuManager) runWorker() {
	for cm.processNextWorkItem() {
	}
}

func (cm *cpuManager) processNextWorkItem() bool {
	obj, shutdown := cm.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer cm.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			cm.workqueue.Forget(obj)
			return nil
		}
		if err := cm.syncHandler(key); err != nil {
			cm.workqueue.AddRateLimited(key)
			return fmt.Errorf("failed to sync '%s': %v, requeuing", key, err)
		}
		cm.workqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (cm *cpuManager) enqueuePod(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	cm.workqueue.Add(key)
}

func (cm *cpuManager) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	pod, err := cm.podLister.Pods(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if utils.IsPodTerminated(pod) {
		klog.V(2).Infof("Found pod(%s) terminated, ignore to allocate cpuset", key)
		return nil
	}

	// Get cpu policy.
	mode := cm.getPodCPUPolicyOrDefault(pod)
	if mode == topologyapi.AnnotationPodCPUPolicyNone {
		return nil
	}

	tr, err := fromZoneListToTopologyResult(GetPodNUMANodeResult(pod), nil)
	if err != nil {
		klog.ErrorS(err, "Failed to decode pod zoneList to topology result", "pod", key)
		return nil
	}
	// container belongs in the shared pool (nothing to do; use default cpuset)
	if len(tr) == 0 {
		return nil
	}

	for _, idx := range GetPodTargetContainerIndices(pod) {
		container := &pod.Spec.Containers[idx]
		if err = cm.Allocate(pod, container, mode, tr); err != nil {
			klog.ErrorS(err, "Failed to allocate cpu", "pod", key, "container", container.Name)
			return err
		}
	}

	return nil
}

func (cm *cpuManager) getPodCPUPolicyOrDefault(pod *corev1.Pod) string {
	mode := GetPodCPUPolicy(pod.Annotations)
	if len(mode) == 0 {
		mode = cm.defaultCPUPolicy
	}
	return mode
}

type reconciledContainer struct {
	podName       string
	containerName string
	containerID   string
}

func (cm *cpuManager) reconcileState() (success []reconciledContainer, failure []reconciledContainer) {
	success = []reconciledContainer{}
	failure = []reconciledContainer{}

	// Get the list of active pods.
	activePods, err := cm.activePods()
	if err != nil {
		klog.ErrorS(err, "Failed to get active pods when reconcileState in cpuManager")
		return
	}

	cm.renewState(activePods)

	cm.Lock()
	defer cm.Unlock()

	for _, pod := range activePods {
		allContainers := pod.Spec.InitContainers
		allContainers = append(allContainers, pod.Spec.Containers...)
		for _, container := range allContainers {
			containerID, _, err := findRunningContainerStatus(&pod.Status, container.Name)
			if err != nil {
				klog.V(4).InfoS("ReconcileState: skipping container", "pod", klog.KObj(pod), "containerName", container.Name, "err", err)
				failure = append(failure, reconciledContainer{pod.Name, container.Name, ""})
				continue
			}

			excludeReservedCPUs := utils.PodExcludeReservedCPUs(pod)

			cset := cm.state.GetCPUSetOrDefault(string(pod.UID), container.Name)
			if excludeReservedCPUs {
				cset = cset.Difference(cm.policy.GetReservedCPUSet())
			}

			if cset.IsEmpty() {
				// NOTE: This should not happen outside of tests.
				klog.V(4).InfoS("ReconcileState: skipping container; assigned cpuset is empty", "pod", klog.KObj(pod), "containerName", container.Name)
				failure = append(failure, reconciledContainer{pod.Name, container.Name, containerID})
				continue
			}

			podUID, containerName, err := cm.containerMap.GetContainerRef(containerID)
			updated := err == nil && podUID == string(pod.UID) && containerName == container.Name

			lcset := cm.lastUpdateState.GetCPUSetOrDefault(string(pod.UID), container.Name)
			if excludeReservedCPUs {
				lcset = lcset.Difference(cm.policy.GetReservedCPUSet())
			}
			if !cset.Equals(lcset) || !updated {
				klog.V(4).InfoS("ReconcileState: updating container", "pod", klog.KObj(pod), "containerName", container.Name, "containerID", containerID, "cpuSet", cset)
				err = cm.updateContainerCPUSet(containerID, cset)
				if err != nil {
					klog.ErrorS(err, "ReconcileState: failed to update container", "pod", klog.KObj(pod), "containerName", container.Name, "containerID", containerID, "cpuSet", cset)
					failure = append(failure, reconciledContainer{pod.Name, container.Name, containerID})
					continue
				}
				cm.lastUpdateState.SetCPUSet(string(pod.UID), container.Name, cset)
				cm.containerMap.Add(string(pod.UID), container.Name, containerID)
			}
			success = append(success, reconciledContainer{pod.Name, container.Name, containerID})
		}
	}
	return success, failure
}

func (cm *cpuManager) getNodeTopologyResult() (TopologyResult, error) {
	nrt, err := cm.nrtLister.Get(cm.nodeName)
	if err != nil {
		return nil, err
	}
	return fromZoneListToTopologyResult(nrt.Zones, nrt.Attributes)
}

func findContainerIDByName(status *corev1.PodStatus, name string) (string, error) {
	allStatuses := status.InitContainerStatuses
	allStatuses = append(allStatuses, status.ContainerStatuses...)
	for _, container := range allStatuses {
		if container.Name == name && container.ContainerID != "" {
			cid := &kubecontainer.ContainerID{}
			err := cid.ParseString(container.ContainerID)
			if err != nil {
				return "", err
			}
			return cid.ID, nil
		}
	}
	return "", fmt.Errorf("failed to find ID for container with name %s in pod status (it may not be running)", name)
}

func findContainerStatusByName(status *corev1.PodStatus, name string) (*corev1.ContainerStatus, error) {
	for _, containerStatus := range append(status.InitContainerStatuses, status.ContainerStatuses...) {
		if containerStatus.Name == name {
			return &containerStatus, nil
		}
	}
	return nil, fmt.Errorf("unable to find status for container with name %s in pod status (it may not be running)", name)
}

func findRunningContainerStatus(status *corev1.PodStatus, container string) (string, *corev1.ContainerStatus, error) {
	containerID, err := findContainerIDByName(status, container)
	if err != nil {
		return "", nil, err
	}

	cstatus, err := findContainerStatusByName(status, container)
	if err != nil {
		return "", nil, err
	}

	if cstatus.State.Waiting != nil ||
		(cstatus.State.Waiting == nil && cstatus.State.Running == nil && cstatus.State.Terminated == nil) {
		return "", nil, fmt.Errorf("container still in the waiting state")
	}

	if cstatus.State.Terminated != nil {
		return "", nil, fmt.Errorf("container in the terminated state but pod is still running, may be in the process of being restarted")
	}
	return containerID, cstatus, nil
}

func buildContainerMapFromRuntime(runtimeService criapis.RuntimeService) containermap.ContainerMap {
	podSandboxMap := make(map[string]string)
	podSandboxList, _ := runtimeService.ListPodSandbox(nil)
	for _, p := range podSandboxList {
		podSandboxMap[p.Id] = p.Metadata.Uid
	}

	containerMap := containermap.NewContainerMap()
	containerList, _ := runtimeService.ListContainers(nil)
	for _, c := range containerList {
		if _, exists := podSandboxMap[c.PodSandboxId]; !exists {
			klog.InfoS("no PodSandBox found for the container", "podSandboxId", c.PodSandboxId, "containerName", c.Metadata.Name, "containerId", c.Id)
			continue
		}
		containerMap.Add(podSandboxMap[c.PodSandboxId], c.Metadata.Name, c.Id)
	}

	return containerMap
}

type TopologyResult map[int]NodeInfo

type NodeInfo struct {
	CPUs               int
	NumReservedCPUs    int
	ReservedSystemCPUs cpuset.CPUSet
}

func NewTopologyResult() TopologyResult {
	return make(TopologyResult)
}

func (tr TopologyResult) CPUs() int {
	var res int
	for _, num := range tr {
		res += num.CPUs
	}
	return res
}

func (tr TopologyResult) NumReservedCPUs() int {
	var res int
	for _, num := range tr {
		res += num.NumReservedCPUs
	}
	return res
}

func (tr TopologyResult) CPUsInNUMANodes(ids ...int) int {
	var res int
	for _, id := range ids {
		res += tr[id].CPUs
	}
	return res
}

func (tr TopologyResult) NumReservedCPUsInNUMANodes(ids ...int) int {
	var res int
	for _, id := range ids {
		res += tr[id].NumReservedCPUs
	}
	return res
}

func fromZoneListToTopologyResult(zones topologyapi.ZoneList, attributes map[string]string) (TopologyResult, error) {
	tr := NewTopologyResult()
	reservedCPUs, err := utils.GetReservedCPUs(attributes[topologyapi.ReservedSystemCPUsAttributes])
	if err != nil {
		return tr, fmt.Errorf("get reserved cpus: %v", err)
	}
	for i := range zones {
		if zones[i].Type == topologyapi.ZoneTypeNode && zones[i].Resources != nil {
			// the zone should have a prefix 'node*'
			nodeID, err := strconv.Atoi(zones[i].Name[4:])
			if err != nil {
				return nil, fmt.Errorf("failed to determine nodeID from zones, expected integer after 4th char")
			}
			if numCPUs := zones[i].Resources.Capacity.Cpu().Value(); numCPUs > 0 {
				tr[nodeID] = NodeInfo{
					CPUs:               int(numCPUs),
					NumReservedCPUs:    int(zones[i].Resources.ReservedCPUNums),
					ReservedSystemCPUs: reservedCPUs,
				}
			}
		}
	}
	return tr, nil
}
