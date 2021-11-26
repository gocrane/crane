package hpa

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
)

func (p *AdvancedHPAController) ReconcileHPA(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID)}),
	}
	err := p.Client.List(ctx, hpaList, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			return p.CreateHPA(ctx, ahpa)
		} else {
			p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedGetHPA", err.Error())
			p.Log.Error(err, "Failed to get HPA", "advanced-hpa", klog.KObj(ahpa))
			return nil, err
		}
	} else if len(hpaList.Items) == 0 {
		return p.CreateHPA(ctx, ahpa)
	}

	return p.UpdateHPAIfNeed(ctx, ahpa, &hpaList.Items[0])
}

func (p *AdvancedHPAController) GetHPA(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID)}),
	}
	err := p.Client.List(ctx, hpaList, opts...)
	if err != nil {
		return nil, err
	} else if len(hpaList.Items) == 0 {
		return nil, nil
	}

	return &hpaList.Items[0], nil
}

func (p *AdvancedHPAController) CreateHPA(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpa, err := p.NewHPAObject(ctx, ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreateHPAObject", err.Error())
		p.Log.Error(err, "Failed to create object", "HorizontalPodAutoscaler", hpa)
		return nil, err
	}

	err = p.Client.Create(ctx, hpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreateHPA", err.Error())
		p.Log.Error(err, "Failed to create", "HorizontalPodAutoscaler", hpa)
		return nil, err
	}

	p.Log.Info("Create HorizontalPodAutoscaler successfully", "HorizontalPodAutoscaler", hpa)
	p.Recorder.Event(ahpa, v1.EventTypeNormal, "HPACreated", "Create HorizontalPodAutoscaler successfully")

	return hpa, nil
}

func (p *AdvancedHPAController) NewHPAObject(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	metrics, err := p.GetHPAMetrics(ctx, ahpa)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("advancedhpa-%s", ahpa.Name)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ahpa.Namespace, // the same namespace to advancedhpa
			Name:      name,
			Labels: map[string]string{
				"app.kubernetes.io/name":                      name,
				"app.kubernetes.io/part-of":                   ahpa.Name,
				"app.kubernetes.io/managed-by":                known.AdvancedHorizontalPodAutoscalerManagedBy,
				known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID),
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: ahpa.Spec.ScaleTargetRef,
			MinReplicas:    ahpa.Spec.MinReplicas,
			MaxReplicas:    ahpa.Spec.MaxReplicas,
			Metrics:        metrics,
		},
	}

	var behavior *autoscalingv2.HorizontalPodAutoscalerBehavior
	// Behavior works in k8s version > 1.18
	if p.K8SVersion.Minor() >= 18 && ahpa.Spec.Behavior != nil {
		behavior = hpa.Spec.Behavior
	} else {
		behavior = nil
	}
	hpa.Spec.Behavior = behavior

	// AdvancedHPA control the underground hpa so set controller reference for hpa here
	if err := controllerutil.SetControllerReference(ahpa, hpa, p.Scheme); err != nil {
		return nil, err
	}

	return hpa, nil
}

func (p *AdvancedHPAController) UpdateHPAIfNeed(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler, hpaExist *autoscalingv2.HorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpa, err := p.NewHPAObject(ctx, ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreateHPAObject", err.Error())
		p.Log.Error(err, "Failed to create object", "HorizontalPodAutoscaler", hpa)
		return nil, err
	}

	if !equality.Semantic.DeepEqual(&hpaExist.Spec, &hpa.Spec) {
		p.Log.Info("HorizontalPodAutoscaler is unsynced according to AdvancedHorizontalPodAutoscaler, should be updated", "currentHPA", hpaExist.Spec, "expectHPA", hpa.Spec)

		hpaExist.Spec = hpa.Spec
		err := p.Update(ctx, hpa)
		if err != nil {
			p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedUpdateHPA", err.Error())
			p.Log.Error(err, "Failed to update", "HorizontalPodAutoscaler", hpaExist)
			return nil, err
		}

		p.Log.Info("Update HorizontalPodAutoscaler successful", "HorizontalPodAutoscaler", hpa)
	}

	return hpaExist, nil
}

// GetHPAMetrics loop metricSpec in AdvancedHorizontalPodAutoscaler and generate metricSpec for HPA
func (p *AdvancedHPAController) GetHPAMetrics(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) ([]autoscalingv2.MetricSpec, error) {
	var metrics []autoscalingv2.MetricSpec
	for _, metric := range ahpa.Spec.Metrics {
		copyMetric := metric.DeepCopy()
		metrics = append(metrics, *copyMetric)
	}

	if IsPredictionEnabled(ahpa) {
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
						// add known.AdvancedHorizontalPodAutoscalerUidLabel=uid in metric.selector
						// MetricAdapter use label selector to match the matching PodGroupPrediction to return metrics
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID),
							},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type: autoscalingv2.AverageValueMetricType,
					},
				}

				// When use AverageUtilization in AdvancedHorizontalPodAutoscaler's metricSpec, convert to AverageValue
				if metric.Resource.Target.AverageUtilization != nil {
					scale, _, err := p.GetScale(ctx, ahpa)
					if err != nil {
						return nil, err
					}

					pods, err := p.GetPodsFromScale(scale)
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

func (p *AdvancedHPAController) DisableHPA(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) error {
	hpa, err := p.GetHPA(ctx, ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedGetHPA", err.Error())
		p.Log.Error(err, "Failed to get", "HorizontalPodAutoscaler", hpa)
		return err
	}

	if hpa == nil {
		// do nothing if not create hpa before
		return nil
	}

	policyDisable := autoscalingv2.DisabledPolicySelect

	hpa.Spec.Behavior = &autoscalingv2.HorizontalPodAutoscalerBehavior{
		ScaleUp: &autoscalingv2.HPAScalingRules{
			SelectPolicy: &policyDisable,
		},
		ScaleDown: &autoscalingv2.HPAScalingRules{
			SelectPolicy: &policyDisable,
		},
	}

	err = p.Update(ctx, hpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedDisableHPA", err.Error())
		p.Log.Error(err, "Failed to disable", "HorizontalPodAutoscaler", hpa)
		return err
	}

	p.Log.Info("Disable scaling successful", "HorizontalPodAutoscaler", hpa)
	return nil
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

func setHPACondition(status *autoscalingapi.AdvancedHorizontalPodAutoscalerStatus, conditions []autoscalingv2.HorizontalPodAutoscalerCondition) {
	for _, cond := range conditions {
		setCondition(status, autoscalingapi.ConditionType(cond.Type), metav1.ConditionStatus(cond.Status), cond.Reason, cond.Message)
	}
}
