package framework

// PrePrepare interface
type PrePrepare interface {
	CheckDataProviders(ctx *RecommendationContext) error
}

// Prepare interface
type Prepare interface {
	CollectData(ctx *RecommendationContext) error
}

type PostPrepare interface {
	PostProcessing(ctx *RecommendationContext) error
}
