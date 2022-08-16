package known

type Module string

const (
	ModuleAnomalyAnalyzer     Module = "AnomalyAnalyzer"
	ModuleStateCollector      Module = "StateCollector"
	ModuleActionExecutor      Module = "ActionExecutor"
	ModuleNodeResourceManager Module = "ModuleNodeResourceManager"
	ModulePodResourceManager  Module = "ModulePodResourceManager"
)
