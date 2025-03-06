package known

import "k8s.io/apimachinery/pkg/api/resource"

type Module string

const (
	ModuleAnomalyAnalyzer     Module = "AnomalyAnalyzer"
	ModuleStateCollector      Module = "StateCollector"
	ModuleActionExecutor      Module = "ActionExecutor"
	ModuleNodeResourceManager Module = "ModuleNodeResourceManager"
	ModulePodResourceManager  Module = "ModulePodResourceManager"
)

type ResourceStatus struct {
	CPUReserved        *resource.Quantity
	CPUUsage           *resource.Quantity
	CPUUsageOffline    *resource.Quantity
	CPUSetIdle         *resource.Quantity
	MemoryReserved     *resource.Quantity
	MemoryUsage        *resource.Quantity
	MemoryUsageOffline *resource.Quantity

	CPUReservedTSP    *resource.Quantity
	MemoryReservedTSP *resource.Quantity
}
