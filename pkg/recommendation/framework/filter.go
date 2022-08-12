package framework

// Filter interface
type Filter interface {
	// The Filter will filter resource can`t be recommended via target recommender.
	Filter(ctx *RecommendationContext) error
}
