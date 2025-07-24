package known

import "os"

var (
	CraneSystemNamespace = "crane-system"
)

func init() {
	if namespace, ok := os.LookupEnv("CRANE_SYSTEM_NAMESPACE"); ok {
		CraneSystemNamespace = namespace
	}
}
