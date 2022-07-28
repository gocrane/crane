package framework

// Observe interface
type Observe interface {
	Observe(ctx *RecommendationContext) error
}
