package idlenode

import (
	"fmt"

	"github.com/gocrane/crane/pkg/common"
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
		ctx.Recommendation.Status.Description = "Node is owned by DaemonSet"
		return nil
	}

	if inr.cpuUsageUtilization == 0 && inr.memoryUsageUtilization == 0 && inr.cpuRequestUtilization == 0 && inr.memoryRequestUtilization == 0 {
		return fmt.Errorf("Node %s is not a idle node ", ctx.Object.GetName())
	}

	// check if cpu usage utilization lt config value
	if cpuUsageUtilization := inr.getMaxValue(inr.cpuUsageUtilization, ctx.InputValue(cpuUsageUtilizationKey)); cpuUsageUtilization > inr.cpuUsageUtilization {
		return fmt.Errorf("Node %s is not a idle node, because the config value is %f, but the node max cpu usage utilization is %f ", ctx.Object.GetName(), inr.cpuUsageUtilization, cpuUsageUtilization)
	}

	// check if memory usage utilization lt config value
	if memoryUsageUtilization := inr.getMaxValue(inr.memoryUsageUtilization, ctx.InputValue(memoryUsageUtilizationKey)); memoryUsageUtilization > inr.memoryUsageUtilization {
		return fmt.Errorf("Node %s is not a idle node, because the config value is %f, but the node max memory usage utilization is %f ", ctx.Object.GetName(), inr.memoryUsageUtilization, memoryUsageUtilization)
	}

	// check if cpu request utilization lt config value
	if cpuRequestUtilization := inr.getMaxValue(inr.cpuRequestUtilization, ctx.InputValue(cpuRequestUtilizationKey)); cpuRequestUtilization > inr.cpuRequestUtilization {
		return fmt.Errorf("Node %s is not a idle node, because the config value is %f, but the node max cpu request utilization is %f ", ctx.Object.GetName(), inr.cpuRequestUtilization, cpuRequestUtilization)
	}

	// check if memory request utilization lt config value
	if memoryRequestUtilization := inr.getMaxValue(inr.memoryRequestUtilization, ctx.InputValue(memoryRequestUtilizationKey)); memoryRequestUtilization > inr.memoryRequestUtilization {
		return fmt.Errorf("Node %s is not a idle node, because the config value is %f, but the node max memory request utilization is %f ", ctx.Object.GetName(), inr.memoryRequestUtilization, memoryRequestUtilization)
	}

	ctx.Recommendation.Status.Action = "Delete"
	ctx.Recommendation.Status.Description = "Node resource utilization is low"
	return nil
}

// Policy add some logic for result of recommend phase.
func (inr *IdleNodeRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}

func (inr *IdleNodeRecommender) getMaxValue(configValue float64, ts []*common.TimeSeries) float64 {
	if configValue == 0 {
		return configValue
	}
	var maxValue float64
	for _, s := range ts[0].Samples {
		if s.Value > maxValue {
			maxValue = s.Value
		}
	}
	return maxValue
}
