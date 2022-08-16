package replicas

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

// Filter out k8s resources that are not supported by the recommender.
func (rr *ReplicasRecommender) Filter(ctx *framework.RecommendationContext) error {
	var err error

	// filter resource that not match objectIdentity
	if err = rr.BaseRecommender.Filter(ctx); err != nil {
		return err
	}

	if err = framework.RetrievePodTemplate(ctx); err != nil {
		return err
	}

	if err = framework.RetrieveScale(ctx); err != nil {
		return err
	}

	if err = framework.RetrievePods(ctx); err != nil {
		return err
	}

	if ctx.Scale != nil && ctx.Scale.Spec.Replicas < int32(rr.WorkloadMinReplicas) {
		return fmt.Errorf("workload replicas %d should be larger than %d ", ctx.Scale.Spec.Replicas, int32(rr.WorkloadMinReplicas))
	}

	if len(ctx.Pods) == 0 {
		return fmt.Errorf("existing pods should be larger than 0 ")
	}

	readyPods := 0
	for _, pod := range ctx.Pods {
		if utils.IsPodAvailable(&pod, int32(rr.PodMinReadySeconds), metav1.Now()) {
			readyPods++
		}
	}

	if readyPods == 0 {
		return fmt.Errorf("pod available number must larger than zero. ")
	}

	availableRatio := float64(readyPods) / float64(len(ctx.Pods))
	if availableRatio < rr.PodAvailableRatio {
		return fmt.Errorf("pod available ratio is %.3f less than %.3f ", availableRatio, rr.PodAvailableRatio)
	}

	return nil
}
