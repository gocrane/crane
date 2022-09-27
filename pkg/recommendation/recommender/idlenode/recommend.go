package idlenode

import (
	"fmt"

	"github.com/gocrane/crane/pkg/recommendation/framework"
)

func (inr *IdleNodeRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (inr *IdleNodeRecommender) Recommend(ctx *framework.RecommendationContext) error {

	// check if all pods in Node are owned by DaemonSet
	allDaemonSetPod := true
	for _, pod := range ctx.Pods {
		for _, ownRef := range pod.OwnerReferences {
			if ownRef.Kind != "DaemonSet" {
				allDaemonSetPod = false
			}
		}
	}

	if allDaemonSetPod {
		ctx.Recommendation.Status.Action = "Delete"
		return nil
	} else {
		return fmt.Errorf("Node %s is not a idle node ", ctx.Object.GetName())
	}
}

// Policy add some logic for result of recommend phase.
func (inr *IdleNodeRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
