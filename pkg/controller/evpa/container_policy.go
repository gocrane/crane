package evpa

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gocrane/crane/pkg/autoscaling/estimator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpatypes "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/utils"
)

type ScaleDirection string

const (
	ScaleUp   ScaleDirection = "ScaleUp"
	ScaleDown ScaleDirection = "ScaleDown"
)

const (
	DefaultStabWindowSeconds = int32(120)
)

func (c *EffectiveVPAController) ReconcileContainerPolicies(ctx context.Context, evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, status autoscalingapi.EffectiveVerticalPodAutoscalerStatus, podTemplate *corev1.PodTemplateSpec, resourceEstimators []estimator.ResourceEstimatorInstance) (currentEstimatorStatus []autoscalingapi.ResourceEstimatorStatus, recommendation *vpatypes.RecommendedPodResources, err error) {
	recommendation = &vpatypes.RecommendedPodResources{}

	rankedEstimators := RankEstimators(resourceEstimators)
	for _, containerPolicy := range evpa.Spec.ResourcePolicy.ContainerPolicies {
		// container scaling is disabled
		if (containerPolicy.ScaleUpPolicy.ScaleMode != nil && *containerPolicy.ScaleUpPolicy.ScaleMode == vpatypes.ContainerScalingModeOff) ||
			(containerPolicy.ScaleDownPolicy.ScaleMode != nil && *containerPolicy.ScaleDownPolicy.ScaleMode == vpatypes.ContainerScalingModeOff) {
			continue
		}

		// get current resource by pod template
		// todo: support "*"
		resourceRequirement := utils.GetResourceByPodTemplate(podTemplate, containerPolicy.ContainerName)
		if resourceRequirement == nil {
			klog.Warningf("ContainerName %s resource requirement not found", containerPolicy.ContainerName)
			continue
		}

		// loop estimator and get final estimated resource for container
		recommendResourceContainer := GetEstimatedResourceForContainer(evpa, containerPolicy, resourceRequirement, rankedEstimators, currentEstimatorStatus)
		if IsResourceListEmpty(recommendResourceContainer) {
			klog.V(4).Infof("Container %s recommend resource is empty, skip scaling. ", containerPolicy.ContainerName)
			continue
		} else {
			klog.V(4).Infof("Container %s recommend resource %v", containerPolicy.ContainerName, recommendResourceContainer)
		}

		shouldScaleUp, msg := c.CheckContainerScalingCondition(evpa, containerPolicy, containerPolicy.ScaleUpPolicy, ScaleUp, resourceRequirement.Requests, recommendResourceContainer)
		if !shouldScaleUp {
			klog.Infof("Should not %s container %s: %s", ScaleUp, containerPolicy.ContainerName, msg)
		} else {
			klog.V(4).Infof("Should %s container %s, resource %v", ScaleUp, containerPolicy.ContainerName, recommendResourceContainer)
			UpdateRecommendStatus(recommendation, containerPolicy.ContainerName, recommendResourceContainer)
			continue
		}

		shouldScaleDown, msg := c.CheckContainerScalingCondition(evpa, containerPolicy, containerPolicy.ScaleUpPolicy, ScaleDown, resourceRequirement.Requests, recommendResourceContainer)
		if !shouldScaleDown {
			klog.Infof("Should not %s container %s: %s", ScaleDown, containerPolicy.ContainerName, msg)
		} else {
			klog.V(4).Infof("Should %s container %s, resource %v", ScaleDown, containerPolicy.ContainerName, recommendResourceContainer)
			UpdateRecommendStatus(recommendation, containerPolicy.ContainerName, recommendResourceContainer)
			continue
		}
	}

	return
}

func UpdateCurrentEstimatorStatus(estimator estimator.ResourceEstimatorInstance, containerName string, resourceList corev1.ResourceList, currentEstimatorStatus []autoscalingapi.ResourceEstimatorStatus) {
	for _, currentStatus := range currentEstimatorStatus {
		if currentStatus.Type == estimator.GetSpec().Type {
			currentStatus.LastUpdateTime = metav1.Now()
			for _, containerRecommend := range currentStatus.Recommendation.ContainerRecommendations {
				if containerRecommend.ContainerName == containerName {
					containerRecommend.Target = resourceList
					// todo: lowBound and UpperBound and uncappedTarget
					return
				}
			}

			return
		}
	}

	currentStatus := autoscalingapi.ResourceEstimatorStatus{
		Type:           estimator.GetSpec().Type,
		LastUpdateTime: metav1.Now(),
		Recommendation: &vpatypes.RecommendedPodResources{
			ContainerRecommendations: []vpatypes.RecommendedContainerResources{
				{
					ContainerName: containerName,
					Target:        resourceList,
				},
			},
		},
	}

	currentEstimatorStatus = append(currentEstimatorStatus, currentStatus) // nolint:ineffassign
}

func UpdateRecommendStatus(recommendation *vpatypes.RecommendedPodResources, containerName string, recommendResource corev1.ResourceList) {
	containerRecommendation := vpatypes.RecommendedContainerResources{
		ContainerName: containerName,
		Target:        recommendResource,
	}

	recommendation.ContainerRecommendations = append(recommendation.ContainerRecommendations, containerRecommendation)
	// todo: handle minAllow and maxAllow, for example uncapping and sharping
	return
}

// CheckContainerScalingCondition check the conditions for container with scale direction
func (c *EffectiveVPAController) CheckContainerScalingCondition(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, containerPolicy autoscalingapi.ContainerResourcePolicy, scalingPolicy *autoscalingapi.ContainerScalingPolicy, direction ScaleDirection, containerResource corev1.ResourceList, recommendContainerResource corev1.ResourceList) (bool, string) {
	if scalingPolicy == nil {
		return true, ""
	}

	if utils.IsResourceEqual(containerResource, recommendContainerResource) {
		return false, "Container resource has not changed"
	}

	if scalingPolicy.ScaleMode != nil && *scalingPolicy.ScaleMode == vpatypes.ContainerScalingModeOff {
		return false, "Scaling disabled"
	}

	lastScaleTime := c.GetLastScaleTime(evpa.Namespace, evpa.Spec.TargetRef.Name, containerPolicy.ContainerName, string(direction))
	stabilizationWindowSeconds := DefaultStabWindowSeconds
	if scalingPolicy.StabilizationWindowSeconds != nil {
		stabilizationWindowSeconds = *scalingPolicy.StabilizationWindowSeconds
	}

	if time.Since(lastScaleTime.Time) <= time.Duration(stabilizationWindowSeconds)*time.Second {
		return false, "In stabilization window"
	}

	if scalingPolicy.MetricThresholds != nil {
		for resourceName := range *scalingPolicy.MetricThresholds {
			usage, err := GetResourceUsedRatio(containerResource, recommendContainerResource, corev1.ResourceName(resourceName))
			if err != nil {
				return false, err.Error()
			}

			metricThresholds := *(scalingPolicy.MetricThresholds)
			if metricThresholds[resourceName].Utilization == nil {
				continue
			}

			klog.V(4).Infof("Resource %s Thresholds %d usage %d", resourceName, *metricThresholds[resourceName].Utilization, usage)
			if direction == ScaleUp && usage <= *metricThresholds[resourceName].Utilization {
				return false, fmt.Sprintf("Resource %s not reach Thresholds %d, usage %d", resourceName, *metricThresholds[resourceName].Utilization, usage)
			}

			if direction == ScaleDown && usage >= *metricThresholds[resourceName].Utilization {
				return false, fmt.Sprintf("Resource %s not reach Thresholds %d, usage %d", resourceName, *metricThresholds[resourceName].Utilization, usage)
			}
		}
	}

	return true, ""
}

// GetScaleEventKey concat information for scale event key
func GetScaleEventKey(namespace string, workload string, container string, direction string) string {
	return namespace + "-" + workload + "-" + container + "-" + direction
}

func (c *EffectiveVPAController) GetLastScaleTime(namespace string, workload string, container string, direction string) metav1.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.lastScaleTime[GetScaleEventKey(namespace, workload, container, direction)]
}

// GetEstimatedResourceForContainer iterate resources based on the result from estimator
// If priority is equal, use the larger resource value
// If priority is larger, use the larger estimator's value if value is not Zero
func GetEstimatedResourceForContainer(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, containerPolicy autoscalingapi.ContainerResourcePolicy, containerResource *corev1.ResourceRequirements, rankedEstimators []ResourceEstimatorInstanceRanked, currentEstimatorStatus []autoscalingapi.ResourceEstimatorStatus) corev1.ResourceList {
	var resourcePrePriorityList []corev1.ResourceList
	for _, estimatorList := range rankedEstimators {
		resourcePrePriority := corev1.ResourceList{}
		for _, estimator := range estimatorList.Estimators {
			resourcesEstimated, err := estimator.GetResourceEstimation(evpa, estimator.GetSpec().Config, containerPolicy.ContainerName, containerResource)
			if err != nil {
				klog.Warningf("Get resource estimator failed, type %s config %v container %s error %v", estimator.GetSpec().Type, estimator.GetSpec().Config, containerPolicy.ContainerName, err)
				continue
			}

			if IsResourceListEmpty(resourcesEstimated) {
				klog.V(4).Infof("Get recommended resource is empty from estimator %s", estimator.GetSpec().Type)
				continue
			}

			klog.V(4).Infof("Get recommended resource %v from estimator %s", resourcesEstimated, estimator.GetSpec().Type)
			UpdateCurrentEstimatorStatus(estimator, containerPolicy.ContainerName, resourcesEstimated, currentEstimatorStatus)

			// Use larger resources if priority is the same
			CalculateResourceByValue(resourcePrePriority, resourcesEstimated)
		}

		if !IsResourceListEmpty(resourcePrePriority) {
			resourcePrePriorityList = append(resourcePrePriorityList, resourcePrePriority)
		}
	}

	// Use the highest priority value
	return CalculateResourceByPriority(resourcePrePriorityList)
}

func CalculateResourceByValue(resourceByValue corev1.ResourceList, resourcesEstimated corev1.ResourceList) {
	for resource := range resourcesEstimated {
		if value, exist := resourceByValue[resource]; exist {
			quantityEstimated := resourcesEstimated[resource]
			if quantityEstimated.Cmp(value) > 0 {
				resourceByValue[resource] = resourcesEstimated[resource]
			}
		} else {
			resourceByValue[resource] = resourcesEstimated[resource]
		}
	}
}

func CalculateResourceByPriority(resourceLists []corev1.ResourceList) corev1.ResourceList {
	resourceByPriority := corev1.ResourceList{}
	for _, resourcePrePriority := range resourceLists {
		for resource := range resourcePrePriority {
			quantity := resourcePrePriority[resource]
			if !quantity.IsZero() {
				resourceByPriority[resource] = quantity
			}
		}
	}

	return resourceByPriority
}

// IsResourceListEmpty loop all resource quantities in resourceList, if all resources' quantity are zero, return true, otherwise return false.
func IsResourceListEmpty(resourceList corev1.ResourceList) bool {
	if resourceList == nil {
		return true
	}

	for resourceName := range resourceList {
		quantity := resourceList[resourceName]
		if !quantity.IsZero() {
			return false
		}
	}

	return true
}

// GetResourceUsedRatio get Resource used ratio from oldResource and newResource
func GetResourceUsedRatio(oldResource, newResource corev1.ResourceList, resourceName corev1.ResourceName) (int32, error) {
	var usedRatio int32
	oldQuantity := oldResource[resourceName]
	newQuantity := newResource[resourceName]
	if newQuantity.IsZero() || oldQuantity.IsZero() {
		return 0, fmt.Errorf("%s resource is zero", resourceName)
	}

	usedRatio = int32(newQuantity.MilliValue() * 100 / oldQuantity.MilliValue())
	return usedRatio, nil
}

type ResourceEstimatorInstanceRanked struct {
	Estimators []estimator.ResourceEstimatorInstance
	Priority   int
}

// RankEstimators return ranked estimator list
func RankEstimators(resourceEstimators []estimator.ResourceEstimatorInstance) []ResourceEstimatorInstanceRanked {
	// sort by priority first
	sort.SliceStable(resourceEstimators, func(i, j int) bool {
		return resourceEstimators[i].GetSpec().Priority < resourceEstimators[j].GetSpec().Priority
	})

	var rankedList []ResourceEstimatorInstanceRanked
	for i := range resourceEstimators {
		isFound := false
		for j := range rankedList {
			if resourceEstimators[i].GetSpec().Priority == rankedList[j].Priority {
				// append estimators which have the same priority
				isFound = true
				rankedList[j].Estimators = append(rankedList[j].Estimators, resourceEstimators[i])
			}
		}

		if !isFound {
			ranked := ResourceEstimatorInstanceRanked{
				Priority:   resourceEstimators[i].GetSpec().Priority,
				Estimators: []estimator.ResourceEstimatorInstance{resourceEstimators[i]},
			}
			rankedList = append(rankedList, ranked)
		}
	}

	return rankedList
}
