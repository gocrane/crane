package recommendation

type RecommenderManager interface {
}

func NewRecommenderManager(recommenders []Recommender) RecommenderManager {
	return &manager{
		recommenders: recommenders,
	}
}

type manager struct {
	recommenders []Recommender
}
