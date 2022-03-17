package cm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/containermap"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
)

const (
	// cpuManagerStateFileName is the file name where cpu manager stores its state
	cpuManagerStateFileName = "cpu_manager_state"
	craneCpusetPolicyName   = "crane"
	// stateFilePath holds the directory where the state file for checkpoints is held.
	stateFilePath        = "/rootvar/run/crane"
	kubeletStateFilePath = "/kubelet/"
	// cpusetReconcilePeriod  is the duration between calls to reconcileState.
	cpusetReconcilePeriod = 5 * time.Second
	// intervalRetryAddContainer is the interval between try add container
	intervalRetryAddContainer = 200 * time.Millisecond
	// timeoutRetryAddContainer is the timeout for adding container
	timeoutRetryAddContainer = 2 * time.Second
)

type AdvancedCpuManager struct {
	isStarted bool

	sync.RWMutex
	policy Policy

	podLister corelisters.PodLister
	podSynced cache.InformerSynced

	runtimeClient pb.RuntimeServiceClient
	runtimeConn   *grpc.ClientConn

	// reconcilePeriod is the duration between calls to reconcileState.
	reconcilePeriod time.Duration

	// state allows pluggable CPU assignment policies while sharing a common
	// representation of state for the system to inspect and reconcile.
	state state.State

	// stateFileDirectory holds the directory where the state file for checkpoints is held.
	stateFileDirectory string

	cadvisor.Manager
}

func NewAdvancedCpuManager(podInformer coreinformers.PodInformer, runtimeEndpoint string, cadvisorManager cadvisor.Manager) *AdvancedCpuManager {
	runtimeClient, runtimeConn, err := cruntime.GetRuntimeClient(runtimeEndpoint, true)
	if err != nil {
		klog.Errorf("GetRuntimeClient failed %s", err.Error())
		return nil
	}

	m := &AdvancedCpuManager{
		podLister:          podInformer.Lister(),
		podSynced:          podInformer.Informer().HasSynced,
		runtimeClient:      runtimeClient,
		runtimeConn:        runtimeConn,
		reconcilePeriod:    cpusetReconcilePeriod,
		stateFileDirectory: stateFilePath,
		Manager:            cadvisorManager,
	}
	//pod add actions need to handle quickly, delete/update can handle in loop laterly
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			pod := new.(*v1.Pod)
			for _, container := range pod.Spec.Containers {
				err := m.Allocate(pod, &container)
				if err == nil {
					_ = wait.PollImmediate(intervalRetryAddContainer, timeoutRetryAddContainer,
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

func (m *AdvancedCpuManager) Run(stop <-chan struct{}) {
	klog.Infof("Starting advanced cpu manager")
	machineInfo, err := m.Manager.GetMachineInfo()
	if err != nil {
		klog.Errorf("GetMachineInfo failed %s", err.Error())
		return
	}
	topo, err := topology.Discover(machineInfo)
	if err != nil {
		klog.Errorf("Topology Discover failed %s", err.Error())
		return
	}
	klog.Infof("Node topology: %+v", topo)
	m.policy, err = NewAdvancedStaticPolicy(topo)
	if err != nil {
		klog.Errorf("New static policy error: %v", err)
		return
	}
	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("advanced-cpuset-manager",
		stop,
		m.podSynced,
	) {
		return
	}

	containerMap, err := buildContainerMapFromRuntime(m.runtimeClient)
	if err != nil {
		klog.Errorf("Failed to build ContainerMapFromRuntime: %v", err)
		return
	}
	if p := m.loadKubeletPolicy(kubeletStateFilePath + cpuManagerStateFileName); p != "none" {
		klog.Errorf("Can not read kubelet policy or is not none, %s", p)
		return
	}
	stateImpl, err := state.NewCheckpointState(m.stateFileDirectory, cpuManagerStateFileName, craneCpusetPolicyName, containerMap)
	if err != nil {
		klog.Errorf("Could not initialize checkpoint manager: %v, please remove policy state file", err)
		return
	}
	m.state = stateImpl

	err = m.policy.Start(m.state)
	if err != nil {
		klog.Errorf("[Advancedcpumanager] policy start error: %v", err)
		return
	}

	go wait.Until(func() { m.reconcileState() }, m.reconcilePeriod, stop)
	m.isStarted = true
}

func (m *AdvancedCpuManager) Allocate(p *v1.Pod, c *v1.Container) error {
	// wait cpu manger start
	_ = wait.PollImmediateUntil(100*time.Millisecond, func() (bool, error) { return m.isStarted, nil }, wait.NeverStop)
	// Garbage collect any stranded resources before allocating CPUs, do not need to allocate
	m.syncState(false)

	m.Lock()
	defer m.Unlock()

	// Call down into the policy to assign this container CPUs if required.
	err := m.policy.Allocate(m.state, p, c)
	if err != nil {
		klog.Errorf("[Advancedcpumanager] Allocate error: %v", err)
		return err
	}
	return nil
}

func (m *AdvancedCpuManager) AddContainer(p *v1.Pod, c *v1.Container) error {
	containerID := GetContainerIdFromPod(p, c.Name)
	cset, ok := m.state.GetCPUSet(string(p.UID), c.Name)
	if !ok {
		cset = m.getSharedCpu().Union(m.state.GetDefaultCPUSet())
	}
	err := m.updateContainerCPUSet(containerID, cset)
	if err != nil {
		klog.Errorf("[Advancedcpumanager] AddContainer error: error updating CPUSet for container (pod: %s, container: %s, container id: %s, err: %v)", p.Name, c.Name, containerID, err)
		return err
	}
	klog.V(5).Infof("[Advancedcpumanager] update container resources is skipped due to cpu set is empty")
	return nil
}

func (m *AdvancedCpuManager) reconcileState() {
	m.syncState(true)
	sharedCPUSet := m.getSharedCpu().Union(m.state.GetDefaultCPUSet())
	for _, pod := range m.activepods() {
		for _, container := range pod.Spec.Containers {
			containerID := GetContainerIdFromPod(pod, container.Name)
			cset, ok := m.state.GetCPUSet(string(pod.UID), container.Name)
			if !ok {
				cset = sharedCPUSet
			}
			klog.Infof("[Advancedcpumanager] reconcileState: updating container (pod: %s, container: %s, container id: %s, cpuset: \"%v\")",
				pod.Name, container.Name, containerID, cset)
			err := m.updateContainerCPUSet(containerID, cset)
			if err != nil {
				klog.Errorf("[Advancedcpumanager] reconcileState: failed to update container (pod: %s, container: %s, container id: %s, cpuset: \"%v\", error: %v)",
					pod.Name, container.Name, containerID, cset, err)
				continue
			}
		}
	}
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
			if m.policy.NeedAllocated(pod, &container) {
				toBeAssignedContainers[string(pod.UID)][container.Name] = struct{}{}
				if _, ok := assignments[string(pod.UID)][container.Name]; !ok && doAllocate {
					err := m.policy.Allocate(m.state, pod, &container)
					if err != nil {
						klog.Errorf("[Advancedcpumanager] Allocate error: %v", err)
						continue
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
				klog.Errorf("[Advancedcpumanager] removeStaleState: removing (pod %s, container: %s)", podUID, containerName)
				err := m.policyRemoveContainerByRef(podUID, containerName)
				if err != nil {
					klog.Errorf("[Advancedcpumanager] removeStaleState: failed to remove (pod %s, container %s), error: %v)", podUID, containerName, err)
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

func (m *AdvancedCpuManager) loadKubeletPolicy(fileName string) string {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		klog.Errorf("[Advancedcpumanager] loadKubeletPolicy: %v", err)
		return ""
	}
	var cpumangerCheckpoint state.CPUManagerCheckpoint
	err = json.Unmarshal(data, &cpumangerCheckpoint)
	if err != nil {
		klog.Errorf("[Advancedcpumanager] unmarshal KubeletPolicy: %v", err)
		return ""
	}
	return cpumangerCheckpoint.PolicyName
}

func (m *AdvancedCpuManager) getSharedCpu() cpuset.CPUSet {
	sharedCPUSet := cpuset.NewCPUSet()
	for _, pod := range m.activepods() {
		for _, container := range pod.Spec.Containers {
			if cset, ok := m.state.GetCPUSet(string(pod.UID), container.Name); ok {
				if csp := GetPodCPUSetType(pod, &container); csp == CPUSetShare {
					sharedCPUSet = sharedCPUSet.Union(cset)
				}
			}
		}
	}
	return sharedCPUSet
}

func (m *AdvancedCpuManager) activepods() []*v1.Pod {
	allPods, _ := m.podLister.List(labels.Everything())
	activePods := make([]*v1.Pod, 0, len(allPods))
	for _, pod := range allPods {
		//todo judge terminating status
		if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded ||
			(pod.DeletionTimestamp != nil && IsPodNotRunning(pod.Status.ContainerStatuses)) {
			continue
		}
		activePods = append(activePods, pod)
	}
	return activePods
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
