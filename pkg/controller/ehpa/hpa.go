package ehpa

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/utils"
)

func (c *EffectiveHPAController) ReconcileHPA(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substitute *autoscalingapi.Substitute) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
	}
	err := c.Client.List(ctx, hpaList, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			return c.CreateHPA(ctx, ehpa, substitute)
		} else {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetHPA", err.Error())
			c.Log.Error(err, "Failed to get HPA", "ehpa", klog.KObj(ehpa))
			return nil, err
		}
	} else if len(hpaList.Items) == 0 {
		return c.CreateHPA(ctx, ehpa, substitute)
	}

	return c.UpdateHPAIfNeed(ctx, ehpa, &hpaList.Items[0], substitute)
}

func (c *EffectiveHPAController) GetHPA(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
	}
	err := c.Client.List(ctx, hpaList, opts...)
	if err != nil {
		return nil, err
	} else if len(hpaList.Items) == 0 {
		return nil, nil
	}

	return &hpaList.Items[0], nil
}

func (c *EffectiveHPAController) CreateHPA(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substitute *autoscalingapi.Substitute) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpa, err := c.NewHPAObject(ctx, ehpa, substitute)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateHPAObject", err.Error())
		c.Log.Error(err, "Failed to create object", "HorizontalPodAutoscaler", hpa)
		return nil, err
	}

	err = c.Client.Create(ctx, hpa)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateHPA", err.Error())
		c.Log.Error(err, "Failed to create", "HorizontalPodAutoscaler", hpa)
		return nil, err
	}

	c.Log.Info("Create HorizontalPodAutoscaler successfully", "HorizontalPodAutoscaler", hpa)
	c.Recorder.Event(ehpa, v1.EventTypeNormal, "HPACreated", "Create HorizontalPodAutoscaler successfully")

	return hpa, nil
}

func (c *EffectiveHPAController) NewHPAObject(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substitute *autoscalingapi.Substitute) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	metrics, err := c.GetHPAMetrics(ctx, ehpa)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("ehpa-%s", ehpa.Name)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ehpa.Namespace, // the same namespace to effectivehpa
			Name:      name,
			Labels: map[string]string{
				"app.kubernetes.io/name":                       name,
				"app.kubernetes.io/part-of":                    ehpa.Name,
				"app.kubernetes.io/managed-by":                 known.EffectiveHorizontalPodAutoscalerManagedBy,
				known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID),
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MinReplicas: ehpa.Spec.MinReplicas,
			MaxReplicas: ehpa.Spec.MaxReplicas,
			Metrics:     metrics,
		},
	}

	if ehpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyPreview {
		hpa.Spec.ScaleTargetRef = autoscalingv2.CrossVersionObjectReference{
			Kind:       "Substitute",
			Name:       substitute.Name,
			APIVersion: "autoscaling.crane.io/v1alpha1",
		}
	} else if ehpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyAuto {
		hpa.Spec.ScaleTargetRef = ehpa.Spec.ScaleTargetRef
	}

	var behavior *autoscalingv2.HorizontalPodAutoscalerBehavior
	// Behavior works in k8s version > 1.18
	if c.K8SVersion.Minor() >= 18 && ehpa.Spec.Behavior != nil {
		behavior = hpa.Spec.Behavior
	} else {
		behavior = nil
	}
	hpa.Spec.Behavior = behavior

	// EffectiveHPA control the underground hpa so set controller reference for hpa here
	if err := controllerutil.SetControllerReference(ehpa, hpa, c.Scheme); err != nil {
		return nil, err
	}

	return hpa, nil
}

func (c *EffectiveHPAController) UpdateHPAIfNeed(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, hpaExist *autoscalingv2.HorizontalPodAutoscaler, substitute *autoscalingapi.Substitute) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpa, err := c.NewHPAObject(ctx, ehpa, substitute)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateHPAObject", err.Error())
		c.Log.Error(err, "Failed to create object", "HorizontalPodAutoscaler", hpa)
		return nil, err
	}

	if !equality.Semantic.DeepEqual(&hpaExist.Spec, &hpa.Spec) {
		c.Log.V(4).Info("HorizontalPodAutoscaler is unsynced according to EffectiveHorizontalPodAutoscaler, should be updated", "currentHPA", hpaExist.Spec, "expectHPA", hpa.Spec)

		hpaExist.Spec = hpa.Spec
		err := c.Update(ctx, hpaExist)
		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedUpdateHPA", err.Error())
			c.Log.Error(err, "Failed to update", "HorizontalPodAutoscaler", hpaExist)
			return nil, err
		}

		c.Log.Info("Update HorizontalPodAutoscaler successful", "HorizontalPodAutoscaler", hpaExist)
	}

	return hpaExist, nil
}

// GetHPAMetrics loop metricSpec in EffectiveHorizontalPodAutoscaler and generate metricSpec for HPA
func (c *EffectiveHPAController) GetHPAMetrics(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) ([]autoscalingv2.MetricSpec, error) {
	var metrics []autoscalingv2.MetricSpec
	for _, metric := range ehpa.Spec.Metrics {
		copyMetric := metric.DeepCopy()
		metrics = append(metrics, *copyMetric)
	}

	if IsPredictionEnabled(ehpa) {
		var customMetricsForPrediction []autoscalingv2.MetricSpec

		for _, metric := range metrics {
			// generate a custom metric for resource metric
			if metric.Type == autoscalingv2.ResourceMetricSourceType {
				name, err := GetPredictionMetricName(metric.Resource.Name)
				if err != nil {
					return nil, err
				}

				customMetric := &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: name,
						// add known.EffectiveHorizontalPodAutoscalerUidLabel=uid in metric.selector
						// MetricAdapter use label selector to match the matching PodGroupPrediction to return metrics
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID),
							},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				}

				// When use AverageUtilization in EffectiveHorizontalPodAutoscaler's metricSpec, convert to AverageValue
				if metric.Resource.Target.AverageUtilization != nil {
					scale, _, err := utils.GetScale(ctx, c.RestMapper, c.ScaleClient, ehpa.Namespace, ehpa.Spec.ScaleTargetRef)
					if err != nil {
						return nil, err
					}

					pods, err := utils.GetPodsFromScale(c.Client, scale)
					if err != nil {
						return nil, err
					}

					requests, err := calculatePodRequests(pods, metric.Resource.Name)
					if err != nil {
						return nil, err
					}

					averageValue := int64((float64(requests) * float64(*metric.Resource.Target.AverageUtilization) / 100) / float64(len(pods)))
					customMetric.Target.AverageValue = resource.NewMilliQuantity(averageValue, resource.DecimalSI)
				} else {
					customMetric.Target.AverageValue = metric.Resource.Target.AverageValue
				}

				metricSpec := autoscalingv2.MetricSpec{Pods: customMetric, Type: autoscalingv2.PodsMetricSourceType}
				customMetricsForPrediction = append(customMetricsForPrediction, metricSpec)
			}
		}

		metrics = append(metrics, customMetricsForPrediction...)
	}

	return metrics, nil
}

// GetPredictionMetricName return metric name used by prediction
func GetPredictionMetricName(Name v1.ResourceName) (string, error) {
	switch Name {
	case v1.ResourceCPU:
		return known.MetricNamePodCpuUsage, nil
	case v1.ResourceMemory:
		return known.MetricNamePodMemoryUsage, nil
	default:
		return "", fmt.Errorf("resource name not predictable")
	}
}

// calculatePodRequests sum request total from pods
func calculatePodRequests(pods []v1.Pod, resource v1.ResourceName) (int64, error) {
	var requests int64
	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			if containerRequest, ok := c.Resources.Requests[resource]; ok {
				requests += containerRequest.MilliValue()
			} else {
				return 0, fmt.Errorf("missing request for %s", resource)
			}
		}
	}
	return requests, nil
}

func setHPACondition(status *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus, conditions []autoscalingv2.HorizontalPodAutoscalerCondition) {
	for _, cond := range conditions {
		setCondition(status, autoscalingapi.ConditionType(cond.Type), metav1.ConditionStatus(cond.Status), cond.Reason, cond.Message)
	}
}
