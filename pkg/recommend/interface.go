package recommend

type Inspector interface {
	// Inspect valid for Context to ensure the target is available for recommendation
	Inspect() error
}

type Advisor interface {
	// Advise analysis and give advice in ProposedRecommendation
	Advise(proposed *ProposedRecommendation) error
}
