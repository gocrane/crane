package cm

import v1 "k8s.io/api/core/v1"

const CPUSetAnnotation string = "qos.gocrane.io/cpu-manager"

// CPUSetPolicy the type for cpuset
type CPUSetPolicy string

const (
	CPUSetNone      CPUSetPolicy = "none"
	CPUSetExclusive CPUSetPolicy = "exclusive"
	CPUSetShare     CPUSetPolicy = "share"
)

func GetPodCPUSetType(pod *v1.Pod, _ *v1.Container) CPUSetPolicy {
	csp := CPUSetPolicy(pod.GetAnnotations()[CPUSetAnnotation])
	if csp == "" {
		return CPUSetNone
	}
	return csp
}

func IsPodNotRunning(statuses []v1.ContainerStatus) bool {
	for _, status := range statuses {
		if status.State.Terminated == nil && status.State.Waiting == nil {
			return false
		}
	}
	return true
}
