package cpumanager

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	cpumanagerstate "k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
	cputopo "k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	topologyapi "github.com/gocrane/api/topology/v1alpha1"
)

const (
	PolicyNameStatic = "static"
)

// Policy implements logic for pod container to CPU assignment.
type Policy interface {
	Name() string
	Start(s cpumanagerstate.State, nodeTopology TopologyResult) error
	// Allocate call is idempotent
	Allocate(s cpumanagerstate.State, pod *corev1.Pod, container *corev1.Container, mode string, tr TopologyResult) error
	// RemoveContainer call is idempotent
	RemoveContainer(s cpumanagerstate.State, podUID string, containerName string)
	// GetAllocatableCPUs returns the assignable (not allocated) CPUs
	GetAllocatableCPUs(m cpumanagerstate.State) (cpuset.CPUSet, error)
	// GetSharedCPUs returns the set of unassigned CPUs minus the reserved set.
	GetSharedCPUs(s cpumanagerstate.State) cpuset.CPUSet
	// GetExclusiveCPUSet returns the set of all CPUs minus the default set.
	GetExclusiveCPUSet(s cpumanagerstate.State) cpuset.CPUSet
	// GetReservedCPUSet returns the set of reserved CPUs
	GetReservedCPUSet() cpuset.CPUSet
}

type staticPolicy struct {
	topology *cputopo.CPUTopology
	reserved cpuset.CPUSet
	// set of CPUs to reuse across allocations in a pod
	cpusToReuse map[string]cpuset.CPUSet
	// getPodFunc returns all pods which belong to specified cpu policy.
	getPodFunc ActivePodsByPolicyFunc
}

// NewStaticPolicy builds a new staticPolicy.
func NewStaticPolicy(topology *cputopo.CPUTopology, getPodFunc ActivePodsByPolicyFunc) Policy {
	return &staticPolicy{
		topology:    topology,
		cpusToReuse: make(map[string]cpuset.CPUSet),
		getPodFunc:  getPodFunc,
	}
}

func (p *staticPolicy) Start(s cpumanagerstate.State, nodeTopology TopologyResult) (err error) {
	if p.reserved, err = buildReservedCPUSet(p.topology, nodeTopology); err != nil {
		klog.ErrorS(err, "Static policy failed to build reserved CPUSet")
		return
	}

	if err = p.validateState(s); err != nil {
		klog.ErrorS(err, "Static policy invalid state, try to remove policy state file")
		s.ClearState()
		if err = p.validateState(s); err != nil {
			klog.ErrorS(err, "Static policy invalid state again, there may be something wrong with topology collection")
			return
		}
	}
	return
}

func (p *staticPolicy) Name() string {
	return PolicyNameStatic
}

func (p *staticPolicy) GetReservedCPUSet() cpuset.CPUSet {
	return p.reserved
}

func (p *staticPolicy) Allocate(
	s cpumanagerstate.State,
	pod *corev1.Pod,
	ctr *corev1.Container,
	mode string,
	tr TopologyResult,
) error {
	klog.V(3).InfoS("Static policy: Allocate", "pod", klog.KObj(pod), "containerName", ctr.Name)
	if cset, ok := s.GetCPUSet(string(pod.UID), ctr.Name); ok {
		p.updateCPUsToReuse(pod, ctr, cset)
		klog.V(4).InfoS("Container already present in state, skipping",
			"pod", klog.KObj(pod), "containerName", ctr.Name)
		return nil
	}
	cset, err := p.allocateCPUs(s, tr, p.cpusToReuse[string(pod.UID)], mode)
	if err != nil {
		return err
	}
	klog.V(3).InfoS("AllocateCPUs", "result", cset)
	s.SetCPUSet(string(pod.UID), ctr.Name, cset)
	p.updateCPUsToReuse(pod, ctr, cset)
	return nil
}

func (p *staticPolicy) allocateCPUs(
	s cpumanagerstate.State,
	tr TopologyResult,
	reusableCPUs cpuset.CPUSet,
	mode string,
) (cpuset.CPUSet, error) {
	klog.V(3).InfoS("AllocateCPUs", "topologyResult", tr)

	// If had reusable cpus, just return them.
	if tr.CPUs() == reusableCPUs.Size() {
		return reusableCPUs, nil
	}

	result := cpuset.NewCPUSet()
	if len(tr) == 0 {
		return result, nil
	}

	// The `shared` refers to default cpuset minus reserved ones.
	// The `allocatable` refers to default cpuset minus reserved and immovable ones.
	shared := p.GetSharedCPUs(s)
	allocatable, err := p.GetAllocatableCPUs(s)
	if err != nil {
		return result, err
	}

	switch mode {
	case topologyapi.AnnotationPodCPUPolicyExclusive, topologyapi.AnnotationPodCPUPolicyImmovable:
		// 1. Get the allocatable cpuset of every NUMA Node.
		// 2. Take by topology.
		// 3. Union result.
		// 4. For exclusive, remove allocated cpuset from the shared ones: default and numa-aware ones.
		for id, info := range tr {
			total := p.topology.CPUDetails.CPUsInNUMANodes(id)
			available := allocatable.Intersection(total)
			aligned, err := p.takeByTopology(available, info.CPUs)
			if err != nil {
				return cpuset.CPUSet{}, err
			}
			result = result.Union(aligned)
		}

		if mode == topologyapi.AnnotationPodCPUPolicyExclusive {
			// Remove allocated CPUs from the shared CPUSet: default and numa-aware.
			s.SetDefaultCPUSet(s.GetDefaultCPUSet().Difference(result))
			assignments := s.GetCPUAssignments()
			for pod := range assignments {
				for container, cset := range assignments[pod] {
					// Remove the result from numa-aware container assignments.
					if !result.Intersection(cset).IsEmpty() {
						s.SetCPUSet(pod, container, cset.Difference(result))
					}
				}
			}
		}

	case topologyapi.AnnotationPodCPUPolicyNUMA:
		// 1. Get the allocatable cpuset of every NUMA Node.
		// 2. Union all result.
		for id := range tr {
			total := p.topology.CPUDetails.CPUsInNUMANodes(id)
			result = result.Union(shared.Intersection(total))
		}
	}
	return result, nil
}

func (p *staticPolicy) RemoveContainer(s cpumanagerstate.State, podUID string, containerName string) {
	klog.InfoS("Static policy: RemoveContainer", "podUID", podUID, "containerName", containerName)
	cpusInUse := getAssignedCPUsOfSiblings(s, podUID, containerName)
	if toRelease, ok := s.GetCPUSet(podUID, containerName); ok {
		s.Delete(podUID, containerName)
		// Mutate the shared pool, adding released cpus.
		toRelease = toRelease.Difference(cpusInUse)
		s.SetDefaultCPUSet(s.GetDefaultCPUSet().Union(toRelease))
	}
}

func (p *staticPolicy) validateState(s cpumanagerstate.State) error {
	tmpAssignments := s.GetCPUAssignments()
	tmpDefaultCPUset := s.GetDefaultCPUSet()

	// 1. Default cpuset cannot be empty when assignments exist
	if tmpDefaultCPUset.IsEmpty() {
		if len(tmpAssignments) != 0 {
			return fmt.Errorf("default cpuset cannot be empty")
		}
		// state is empty initialize
		allCPUs := p.topology.CPUDetails.CPUs()
		s.SetDefaultCPUSet(allCPUs)
		return nil
	}

	// State has already been initialized from file (is not empty)
	// 1. Check if the reserved cpuset is not part of default cpuset because:
	// - kube/system reserved have changed (increased) - may lead to some containers not being able to start
	// - user tampered with file
	if !p.reserved.Intersection(tmpDefaultCPUset).Equals(p.reserved) {
		return fmt.Errorf("not all reserved cpus: \"%s\" are present in defaultCpuSet: \"%s\"",
			p.reserved.String(), tmpDefaultCPUset.String())
	}

	// 2. It's possible that the set of available CPUs has changed since
	// the state was written. This can be due to for example
	// offlining a CPU when kubelet is not running. If this happens,
	// CPU manager will run into trouble when later it tries to
	// assign non-existent CPUs to containers. Validate that the
	// topology that was received during CPU manager startup matches with
	// the set of CPUs stored in the state.
	totalKnownCPUs := tmpDefaultCPUset.Clone()
	var tmpCPUSets []cpuset.CPUSet
	for pod := range tmpAssignments {
		for _, cset := range tmpAssignments[pod] {
			tmpCPUSets = append(tmpCPUSets, cset)
		}
	}
	totalKnownCPUs = totalKnownCPUs.UnionAll(tmpCPUSets)
	if !totalKnownCPUs.Equals(p.topology.CPUDetails.CPUs()) {
		return fmt.Errorf("current set of available CPUs \"%s\" doesn't match with CPUs in state \"%s\"",
			p.topology.CPUDetails.CPUs().String(), totalKnownCPUs.String())
	}

	return nil
}

// GetAllocatableCPUs returns the set of unassigned CPUs minus the reserved set and immovable ones.
func (p *staticPolicy) GetAllocatableCPUs(s cpumanagerstate.State) (cpuset.CPUSet, error) {
	pods, err := p.getPodFunc(topologyapi.AnnotationPodCPUPolicyImmovable)
	if err != nil {
		return cpuset.CPUSet{}, fmt.Errorf("failed to get immovable pods, %v", err)
	}
	immovableCPUSet := cpuset.NewCPUSet()
	for _, pod := range pods {
		for _, container := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
			if cset, ok := s.GetCPUSet(string(pod.UID), container.Name); ok {
				immovableCPUSet = immovableCPUSet.Union(cset)
			}
		}
	}
	return s.GetDefaultCPUSet().Difference(p.reserved).Difference(immovableCPUSet), nil
}

// GetSharedCPUs returns the set of unassigned CPUs minus the reserved set.
func (p *staticPolicy) GetSharedCPUs(s cpumanagerstate.State) cpuset.CPUSet {
	return s.GetDefaultCPUSet().Difference(p.reserved)
}

// GetExclusiveCPUSet returns the set of all CPUs minus the default set.
func (p *staticPolicy) GetExclusiveCPUSet(s cpumanagerstate.State) cpuset.CPUSet {
	return p.topology.CPUDetails.CPUs().Difference(s.GetDefaultCPUSet())
}

func (p *staticPolicy) updateCPUsToReuse(pod *corev1.Pod, container *corev1.Container, cset cpuset.CPUSet) {
	// If pod entries to m.cpusToReuse other than the current pod exist, delete them.
	for podUID := range p.cpusToReuse {
		if podUID != string(pod.UID) {
			delete(p.cpusToReuse, podUID)
		}
	}
	// If no cpuset exists for cpusToReuse by this pod yet, create one.
	if _, ok := p.cpusToReuse[string(pod.UID)]; !ok {
		p.cpusToReuse[string(pod.UID)] = cpuset.NewCPUSet()
	}
	// Add its cpuset to the cpuset of reusable CPUs for any new allocations.
	for _, c := range pod.Spec.Containers {
		if container.Name == c.Name {
			p.cpusToReuse[string(pod.UID)] = p.cpusToReuse[string(pod.UID)].Union(cset)
			return
		}
	}
}

func (p *staticPolicy) takeByTopology(availableCPUs cpuset.CPUSet, numCPUs int) (cpuset.CPUSet, error) {
	return takeByTopologyNUMAPacked(p.topology, availableCPUs, numCPUs)
}

func buildReservedCPUSet(topology *cputopo.CPUTopology, tr TopologyResult) (cpuset.CPUSet, error) {
	res := cpuset.NewCPUSet()
	for idx, info := range tr {
		// if reserved system cpus is specified, use it directly
		if info.ReservedSystemCPUs.Size() != 0 {
			res = res.Union(info.ReservedSystemCPUs)
			continue
		}
		reserved, err := takeByTopologyNUMAPacked(topology, topology.CPUDetails.CPUsInNUMANodes(idx), info.NumReservedCPUs)
		if err != nil {
			return cpuset.CPUSet{}, err
		}
		res = res.Union(reserved)
	}
	return res, nil
}

// getAssignedCPUsOfSiblings returns assigned cpus of given container's siblings(all containers other than the given container) in the given pod `podUID`.
func getAssignedCPUsOfSiblings(s cpumanagerstate.State, podUID string, containerName string) cpuset.CPUSet {
	assignments := s.GetCPUAssignments()
	cset := cpuset.NewCPUSet()
	for name, cpus := range assignments[podUID] {
		if containerName == name {
			continue
		}
		cset = cset.Union(cpus)
	}
	return cset
}
