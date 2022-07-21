package replicas

import (
	"github.com/gocrane/crane/pkg/recommendation"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"k8s.io/klog"
)

var _ recommendation.Recommender = &ReplicasRecommender{}

type ReplicasRecommender struct {
}

func (rr *ReplicasRecommender) Name() recommendation.RecommenderName {
	return recommendation.ReplicasRecommender
}

// NewReplicasRecommender create a new replicas recommender.
func NewReplicasRecommender() ReplicasRecommender {
	return ReplicasRecommender{}
}

func (rr *ReplicasRecommender) Run(ctx *framework.RecommendationContext) {
	// If context is canceled, directly return.
	if ctx.Canceled() {
		klog.Infof("Recommender %q has been cancelled...", rr.Name())
		return
	}

	err := rr.Filter(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at filter phase!")
		return
	}

	err = rr.CheckDataProviders(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at prepare check data provider phase!")
		return
	}

	err = rr.CollectData(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at prepare collect data phase!")
		return
	}

	err = rr.PostProcessing(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at prepare data post processing phase!")
		return
	}

	err = rr.Recommend(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at recommend phase!")
		return
	}

	err = rr.Policy(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at recommend policy phase!")
		return
	}

	err = rr.Observe(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at observe phase!")
		return
	}
	return
}
