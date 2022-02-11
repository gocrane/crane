package ehpa

type EhpaControllerConfig struct {
	PropagationConfig EhpaControllerPropagationConfig
}

type EhpaControllerPropagationConfig struct {
	LabelPrefixes      []string
	AnnotationPrefixes []string
	Labels             []string
	Annotations        []string
}
