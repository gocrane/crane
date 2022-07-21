package recommendation

type RecommenderManager struct {
	recommenders []Recommender
}

func NewRecommenderManager(recommenders []Recommender) *RecommenderManager {
	return &RecommenderManager{
		recommenders: recommenders,
	}
}

func LoadRecommendationConfiguration(file string) []Recommender {
	recommenders := make([]Recommender)
}
