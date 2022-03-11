package cm

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/topology"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
)

// AdvancedPolicyStatic is the name of the advanced static policy
const AdvancedPolicyStatic policyName = "advanced-static"

type advancedStaticPolicy struct {
	// cpu socket topology
	topology *topology.CPUTopology
	// set of CPUs that is not available for exclusive assignment
	reserved cpuset.CPUSet
}

func NewAdvancedStaticPolicy(topology *topology.CPUTopology) (Policy, error) {
	return &advancedStaticPolicy{
		topology: topology,
		reserved: cpuset.MustParse("0"),
	}, nil
}

func (p *advancedStaticPolicy) Name() string {
	return string(AdvancedPolicyStatic)
}

func (p *advancedStaticPolicy) Start(s state.State) error {
	if err := p.validateState(s); err != nil {
		klog.Errorf("[cpumanager] advanced static policy invalid state: %v, please drain node and remove policy state file", err)
		return err
	}
	return nil
}

func (p *advancedStaticPolicy) validateState(s state.State) error {
	tmpAssignments := s.GetCPUAssignments()
	tmpDefaultCPUset := s.GetDefaultCPUSet()

	// Default cpuset cannot be empty when assignments exist
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

	// 2. Check if state for static policy is consistent
	for pod := range tmpAssignments {
		for container, cset := range tmpAssignments[pod] {
			// None of the cpu in DEFAULT cset should be in s.assignments
			if !tmpDefaultCPUset.Intersection(cset).IsEmpty() {
				return fmt.Errorf("pod: %s, container: %s cpuset: \"%s\" overlaps with default cpuset \"%s\"",
					pod, container, cset.String(), tmpDefaultCPUset.String())
			}
		}
	}

	// 3. It's possible that the set of available CPUs has changed since
	// the state was written. This can be due to for example
	// offlining a CPU when kubelet is not running. If this happens,
	// CPU manager will run into trouble when later it tries to
	// assign non-existent CPUs to containers. Validate that the
	// topology that was received during CPU manager startup matches with
	// the set of CPUs stored in the state.
	totalKnownCPUs := tmpDefaultCPUset.Clone()
	tmpCPUSets := []cpuset.CPUSet{}
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

func (p *advancedStaticPolicy) Allocate(s state.State, pod *v1.Pod, container *v1.Container) error {
	if numCPUs := p.guaranteedCPUs(pod, container); numCPUs != 0 {
		klog.Infof("[cpumanager] advanced static policy: Allocate (pod: %s, container: %s)", format.Pod(pod), container.Name)
		if _, ok := s.GetCPUSet(string(pod.UID), container.Name); ok {
			klog.Infof("[cpumanager] advanced static policy: container already present in state, skipping (pod: %s, container: %s)", format.Pod(pod), container.Name)
			return nil
		}

		// Allocate CPUs according to the NUMA affinity contained in the hint.
		cpuset, err := p.allocateCPUs(s, numCPUs)
		if err != nil {
			klog.Errorf("[cpumanager] unable to allocate %d CPUs (pod: %s, container: %s, error: %v)", numCPUs, format.Pod(pod), container.Name, err)
			return err
		}
		s.SetCPUSet(string(pod.UID), container.Name, cpuset)
	}
	// container belongs in the shared pool (nothing to do; use default cpuset)
	return nil
}

func (p *advancedStaticPolicy) allocateCPUs(s state.State, numCPUs int) (cpuset.CPUSet, error) {
	klog.Infof("[cpumanager] allocateCpus: (numCPUs: %d)", numCPUs)

	// If there are aligned CPUs in numaAffinity, attempt to take those first.
	result := cpuset.NewCPUSet()
	klog.Infof("[cpumanager] allocateCpu %+v", p.assignableCPUs(s))
	// Get any remaining CPUs from what's leftover after attempting to grab aligned ones.
	remainingCPUs, err := takeByTopology(p.topology, p.assignableCPUs(s), numCPUs)
	if err != nil {
		return cpuset.NewCPUSet(), err
	}
	result = result.Union(remainingCPUs)

	// Remove allocated CPUs from the shared CPUSet.
	s.SetDefaultCPUSet(s.GetDefaultCPUSet().Difference(result))

	klog.Infof("[cpumanager] allocateCPUs: returning \"%v\"", result)
	return result, nil
}

// assignableCPUs returns the set of unassigned CPUs minus the reserved set.
func (p *advancedStaticPolicy) assignableCPUs(s state.State) cpuset.CPUSet {
	return s.GetDefaultCPUSet().Difference(p.reserved)
}

func (p *advancedStaticPolicy) RemoveContainer(s state.State, podUID string, containerName string) error {
	klog.Infof("[cpumanager] static policy: RemoveContainer (pod: %s, container: %s)", podUID, containerName)
	if toRelease, ok := s.GetCPUSet(podUID, containerName); ok {
		s.Delete(podUID, containerName)
		// Mutate the shared pool, adding released cpus.
		s.SetDefaultCPUSet(s.GetDefaultCPUSet().Union(toRelease))
	}
	return nil
}
func (p *advancedStaticPolicy) NeedAssigned(pod *v1.Pod, container *v1.Container) bool {
	return p.guaranteedCPUs(pod, container) > 0
}

func (p *advancedStaticPolicy) guaranteedCPUs(pod *v1.Pod, container *v1.Container) int {
	cpuQuantity := container.Resources.Requests[v1.ResourceCPU]
	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return 0
	}
	if container.Resources.Requests[v1.ResourceCPU] != container.Resources.Limits[v1.ResourceCPU] {
		return 0
	}
	if csp := podCPUSetType(pod, container); csp == CPUSetNone {
		return 0
	}
	return int(cpuQuantity.Value())
}
