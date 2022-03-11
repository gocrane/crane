package cm

import (
	"fmt"
	"strings"
	"sync"
	"time"

	cmanager "github.com/google/cadvisor/manager"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cm/containermap"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
)

const (
	// cpuManagerStateFileName is the file name where cpu manager stores its state
	cpuManagerStateFileName = "cpu_manager_state"
	stateFilePath           = "/rootvar/run/crane"
)

type AdvancedCpuManager struct {
	isStarted bool
	sync.RWMutex
	policy Policy

	podLister     corelisters.PodLister
	runtimeClient pb.RuntimeServiceClient
	runtimeConn   *grpc.ClientConn
	// reconcilePeriod is the duration between calls to reconcileState.
	reconcilePeriod time.Duration
	// state allows pluggable CPU assignment policies while sharing a common
	// representation of state for the system to inspect and reconcile.
	state state.State
	// stateFileDirectory holds the directory where the state file for checkpoints is held.
	stateFileDirectory string
}

func NewAdvancedCpuManager(client clientset.Interface, nodeName string, podInformer coreinformers.PodInformer,
	nodeInformer coreinformers.NodeInformer, runtimeEndpoint string, cadvisorManager cmanager.Manager) *AdvancedCpuManager {
	runtimeClient, runtimeConn, err := cruntime.GetRuntimeClient(runtimeEndpoint, true)
	if err != nil {
		klog.Errorf("GetRuntimeClient failed %s", err.Error())
		return nil
	}
	machineInfo, err := cadvisorManager.GetMachineInfo()
	if err != nil {
		klog.Errorf("cadvisorManager GetMachineInfo failed %s", err.Error())
		return nil
	}
	topo, err := topology.Discover(machineInfo)
	if err != nil {
		klog.Errorf("topology Discover failed %s", err.Error())
		return nil
	}
	klog.Infof("node topology: %+v", topo)
	policy, err := NewAdvancedStaticPolicy(topo)
	if err != nil {
		klog.Errorf("new static policy error: %v", err)
		return nil
	}
	m := &AdvancedCpuManager{
		policy:             policy,
		podLister:          podInformer.Lister(),
		runtimeClient:      runtimeClient,
		runtimeConn:        runtimeConn,
		reconcilePeriod:    5 * time.Second,
		stateFileDirectory: stateFilePath,
	}
	//pod add need to handle quickly,add/update can use loop
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			pod := new.(*v1.Pod)
			for _, container := range pod.Spec.Containers {
				err := m.Allocate(pod, &container)
				if err == nil {
					wait.PollImmediate(100*time.Millisecond, 2*time.Second,
						func() (bool, error) {
							return m.AddContainer(pod, &container) == nil, nil
						})
				}
			}
		},
	})
	return m
}

func (m *AdvancedCpuManager) Name() string {
	return "AdvancedCpuManager"
}

// TODO If kubelet use static Cpu Manager Policy, then do not run
func (m *AdvancedCpuManager) Run(stop <-chan struct{}) {
	klog.Infof("Starting advanced cpu manager")
	containerMap, err := buildContainerMapFromRuntime(m.runtimeClient)
	if err != nil {
		return
	}
	stateImpl, err := state.NewCheckpointState(m.stateFileDirectory, cpuManagerStateFileName, "crane", containerMap)
	if err != nil {
		klog.Errorf("could not initialize checkpoint manager: %v, please remove policy state file", err)
		return
	}
	m.state = stateImpl

	err = m.policy.Start(m.state)
	if err != nil {
		klog.Errorf("[advancedcpumanager] policy start error: %v", err)
		return
	}

	go wait.Until(func() { m.reconcileState() }, m.reconcilePeriod, stop)
	m.isStarted = true
}

func (m *AdvancedCpuManager) Allocate(p *v1.Pod, c *v1.Container) error {
	wait.PollImmediateUntil(100*time.Millisecond, func() (bool, error) { return m.isStarted, nil }, wait.NeverStop)
	// Garbage collect any stranded resources before allocating CPUs.
	m.syncState(false)
	m.Lock()
	defer m.Unlock()
	// Call down into the policy to assign this container CPUs if required.
	err := m.policy.Allocate(m.state, p, c)
	if err != nil {
		klog.Errorf("[advancedcpumanager] Allocate error: %v", err)
		return err
	}
	return nil
}

func (m *AdvancedCpuManager) reconcileState() {
	m.syncState(true)
	sharedCPUSet := m.getSharedCpu().Union(m.state.GetDefaultCPUSet())
	for _, pod := range m.activepods() {
		// filter terminating
		for _, container := range pod.Spec.Containers {
			containerID := GetContainerIdFromPod(pod, container.Name)
			cset, ok := m.state.GetCPUSet(string(pod.UID), container.Name)
			if !ok {
				cset = sharedCPUSet
			}
			klog.Infof("[cpumanager] reconcileState: updating container (pod: %s, container: %s, container id: %s, cpuset: \"%v\")",
				pod.Name, container.Name, containerID, cset)
			err := m.updateContainerCPUSet(containerID, cset)
			if err != nil {
				klog.Errorf("[cpumanager] reconcileState: failed to update container (pod: %s, container: %s, container id: %s, cpuset: \"%v\", error: %v)",
					pod.Name, container.Name, containerID, cset, err)
				continue
			}
		}
	}
	return
}

func (m *AdvancedCpuManager) syncState(doAllocate bool) {
	// We grab the lock to ensure that no new containers will grab CPUs while
	// executing the code below. Without this lock, its possible that we end up
	// removing state that is newly added by an asynchronous call to
	// AddContainer() during the execution of this code.
	m.Lock()
	defer m.Unlock()
	assignments := m.state.GetCPUAssignments()

	// Build a list of (podUID, containerName) pairs for all need to be assigned containers in all active Pods.
	toBeAssignedContainers := make(map[string]map[string]struct{})
	for _, pod := range m.activepods() {
		toBeAssignedContainers[string(pod.UID)] = make(map[string]struct{})
		for _, container := range pod.Spec.Containers {
			if m.policy.NeedAssigned(pod, &container) {
				toBeAssignedContainers[string(pod.UID)][container.Name] = struct{}{}
				if _, ok := assignments[string(pod.UID)][container.Name]; !ok && doAllocate {
					err := m.policy.Allocate(m.state, pod, &container)
					if err != nil {
						klog.Errorf("[advancedcpumanager] Allocate error: %v", err)
					}
				}
			}
		}
	}

	// Loop through the CPUManager state. Remove any state for containers not
	// in the `toBeAssignedContainers` list built above.
	for podUID := range assignments {
		for containerName := range assignments[podUID] {
			if _, ok := toBeAssignedContainers[podUID][containerName]; !ok {
				klog.Errorf("[advancedcpumanager] removeStaleState: removing (pod %s, container: %s)", podUID, containerName)
				err := m.policyRemoveContainerByRef(podUID, containerName)
				if err != nil {
					klog.Errorf("[advancedcpumanager] removeStaleState: failed to remove (pod %s, container %s), error: %v)", podUID, containerName, err)
				}
			}
		}
	}
}

func (m *AdvancedCpuManager) policyRemoveContainerByRef(podUID string, containerName string) error {
	err := m.policy.RemoveContainer(m.state, podUID, containerName)
	return err
}

func (m *AdvancedCpuManager) updateContainerCPUSet(containerID string, cpus cpuset.CPUSet) error {
	return cruntime.UpdateContainerResources(
		m.runtimeClient,
		containerID,
		cruntime.UpdateOptions{CpusetCpus: cpus.String()},
	)
}

func buildContainerMapFromRuntime(runtimeClient pb.RuntimeServiceClient) (containermap.ContainerMap, error) {
	podSandboxMap := make(map[string]string)
	podSandboxList, _ := cruntime.ListPodSandboxes(runtimeClient, cruntime.ListOptions{})

	for _, p := range podSandboxList {
		podSandboxMap[p.Id] = p.Metadata.Uid
	}

	containerMap := containermap.NewContainerMap()
	containerList, _ := cruntime.ListContainers(runtimeClient, cruntime.ListOptions{})
	for _, c := range containerList {
		if _, exists := podSandboxMap[c.PodSandboxId]; !exists {
			return nil, fmt.Errorf("no PodsandBox found with Id '%s'", c.PodSandboxId)
		}
		containerMap.Add(podSandboxMap[c.PodSandboxId], c.Metadata.Name, c.Id)
	}

	return containerMap, nil
}

func GetContainerIdFromPod(pod *v1.Pod, name string) string {
	if name == "" {
		return ""
	}

	for _, v := range pod.Status.ContainerStatuses {
		if v.Name == name {
			strList := strings.Split(v.ContainerID, "//")
			if len(strList) > 0 {
				return strList[len(strList)-1]
			}
		}
	}

	return ""
}

func (m *AdvancedCpuManager) AddContainer(p *v1.Pod, c *v1.Container) error {
	containerID := GetContainerIdFromPod(p, c.Name)
	cset, ok := m.state.GetCPUSet(string(p.UID), c.Name)
	if !ok {
		cset = m.getSharedCpu().Union(m.state.GetDefaultCPUSet())
	}
	err := m.updateContainerCPUSet(containerID, cset)
	if err != nil {
		klog.Errorf("[advancedcpumanager] AddContainer error: error updating CPUSet for container (pod: %s, container: %s, container id: %s, err: %v)", p.Name, c.Name, containerID, err)
		return err
	}
	klog.V(5).Infof("[advancedcpumanager] update container resources is skipped due to cpu set is empty")
	return nil
}

func (m *AdvancedCpuManager) getSharedCpu() cpuset.CPUSet {
	sharedCPUSet := cpuset.NewCPUSet()
	for _, pod := range m.activepods() {
		for _, container := range pod.Spec.Containers {
			if cset, ok := m.state.GetCPUSet(string(pod.UID), container.Name); ok {
				if csp := podCPUSetType(pod, &container); csp == CPUSetShare {
					sharedCPUSet = sharedCPUSet.Union(cset)
				}
			}
		}
	}
	return sharedCPUSet
}

func (m *AdvancedCpuManager) GetExclusiveCpu() cpuset.CPUSet {
	exclusiveCPUSet := cpuset.NewCPUSet()
	for _, pod := range m.activepods() {
		for _, container := range pod.Spec.Containers {
			if cset, ok := m.state.GetCPUSet(string(pod.UID), container.Name); ok {
				if csp := podCPUSetType(pod, &container); csp == CPUSetExclusive {
					exclusiveCPUSet = exclusiveCPUSet.Union(cset)
				}
			}
		}
	}
	return exclusiveCPUSet
}

func (m *AdvancedCpuManager) activepods() []*v1.Pod {
	allPods, _ := m.podLister.List(labels.Everything())
	activePods := make([]*v1.Pod, 0, len(allPods))
	for _, pod := range allPods {
		//todo judge terminating status
		if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded ||
			(pod.DeletionTimestamp != nil && podNotRunning(pod.Status.ContainerStatuses)) {
			continue
		}
		activePods = append(activePods, pod)
	}
	return activePods
}

func podNotRunning(statuses []v1.ContainerStatus) bool {
	for _, status := range statuses {
		if status.State.Terminated == nil && status.State.Waiting == nil {
			return false
		}
	}
	return true
}
