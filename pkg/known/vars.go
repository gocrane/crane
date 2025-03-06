package known

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

var (
	CraneSystemNamespace = "crane-system"
)

func init() {
	if namespace, ok := os.LookupEnv("CRANE_SYSTEM_NAMESPACE"); ok {
		CraneSystemNamespace = namespace
	}
}

const (
	// ElasticResourcePrefix is crane resource namespace prefix.
	ElasticResourcePrefix = "gocrane.io/"
)

var (
	ElasticCPU    = ElasticResourcePrefix + corev1.ResourceCPU
	ElasticMemory = ElasticResourcePrefix + corev1.ResourceMemory
)
