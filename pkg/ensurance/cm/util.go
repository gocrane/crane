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

func podCPUSetType(pod *v1.Pod, _ *v1.Container) CPUSetPolicy {
	csp := CPUSetPolicy(pod.GetAnnotations()[CPUSetAnnotation])
	if csp == "" {
		return CPUSetNone
	}
	return csp
}
