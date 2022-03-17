package known

type Module string

const (
	ModuleAnormalyAnalyzer    Module = "AnormalyAnalyzer"
	ModuleStateCollector      Module = "StateCollector"
	ModuleActionExecutor      Module = "ActionExecutor"
	ModuleNodeResourceManager Module = "ModuleNodeResourceManager"
	ModulePodResourceManager  Module = "ModulePodResourceManager"
)
