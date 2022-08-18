package framework

// PreRecommend interface
type PreRecommend interface {
	PreRecommend(ctx *RecommendationContext) error
}

// Recommend interface
type Recommend interface {
	Recommend(ctx *RecommendationContext) error
}

// PostRecommend interface
type PostRecommend interface {
	Policy(ctx *RecommendationContext) error
}
