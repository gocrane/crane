package cm

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
)

type policyName string

// Policy implements logic for pod container to CPU assignment.
type Policy interface {
	Name() string
	// Start is only called once to start policy
	Start(s state.State) error
	// Allocate call is idempotent to allocate cpus for containers
	Allocate(s state.State, pod *v1.Pod, container *v1.Container) error
	// RemoveContainer call is idempotent to reclaim cpus
	RemoveContainer(s state.State, podUID string, containerName string) error
	// NeedAllocated is called to judge if container needs to allocate cpu
	NeedAllocated(pod *v1.Pod, container *v1.Container) bool
}
