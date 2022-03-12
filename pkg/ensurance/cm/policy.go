package cm

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state"
)

type policyName string

// Policy implements logic for pod container to CPU assignment.
type Policy interface {
	Name() string
	Start(s state.State) error
	// Allocate call is idempotent
	Allocate(s state.State, pod *v1.Pod, container *v1.Container) error
	//RemoveContainer call is idempotent
	RemoveContainer(s state.State, podUID string, containerName string) error
	NeedAssigned(pod *v1.Pod, container *v1.Container) bool
}
