package ehpa

import (
	"context"
	"fmt"
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metricprovider"
	"github.com/gocrane/crane/pkg/utils"
)

func (c *EffectiveHPAController) ReconcileHPA(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substitute *autoscalingapi.Substitute, tsp *predictionapi.TimeSeriesPrediction) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
	}
	err := c.Client.List(ctx, hpaList, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			return c.CreateHPA(ctx, ehpa, substitute, tsp)
		} else {
			c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedGetHPA", err.Error())
			klog.Errorf("Failed to get HPA, ehpa %s error %v", klog.KObj(ehpa), err)
			return nil, err
		}
	} else if len(hpaList.Items) == 0 {
		return c.CreateHPA(ctx, ehpa, substitute, tsp)
	}

	return c.UpdateHPAIfNeed(ctx, ehpa, &hpaList.Items[0], substitute, tsp)
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

func (c *EffectiveHPAController) CreateHPA(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substitute *autoscalingapi.Substitute, tsp *predictionapi.TimeSeriesPrediction) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpa, err := c.NewHPAObject(ctx, ehpa, substitute, tsp)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedCreateHPAObject", err.Error())
		klog.Errorf("Failed to create object, HorizontalPodAutoscaler %s error %v", klog.KObj(hpa), err)
		return nil, err
	}

	err = c.Client.Create(ctx, hpa)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedCreateHPA", err.Error())
		klog.Errorf("Failed to create HorizontalPodAutoscaler %s, error %v", klog.KObj(hpa), err)
		return nil, err
	}

	klog.Infof("Create HorizontalPodAutoscaler successfully, HorizontalPodAutoscaler %s", klog.KObj(hpa))
	c.Recorder.Event(ehpa, v1.EventTypeNormal, "HPACreated", "Create HorizontalPodAutoscaler successfully")

	return hpa, nil
}

func (c *EffectiveHPAController) NewHPAObject(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substitute *autoscalingapi.Substitute, tsp *predictionapi.TimeSeriesPrediction) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	metrics, err := c.GetHPAMetrics(ctx, ehpa, tsp)
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
		behavior = ehpa.Spec.Behavior
	} else {
		behavior = nil
	}
	hpa.Spec.Behavior = behavior

	// propagate target labels and annotations to hpa from ehpa; related to --ehpa-propagation-label-prefixes, --ehpa-propagation-annotation-prefixes,
	// --ehpa-propagation-labels, --ehpa-propagation-annotations
	c.propagateLabelAndAnnotation(ehpa, hpa)

	// EffectiveHPA control the underground hpa so set controller reference for hpa here
	if err := controllerutil.SetControllerReference(ehpa, hpa, c.Scheme); err != nil {
		return nil, err
	}

	return hpa, nil
}

func (c *EffectiveHPAController) UpdateHPAIfNeed(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, hpaExist *autoscalingv2.HorizontalPodAutoscaler, substitute *autoscalingapi.Substitute, tsp *predictionapi.TimeSeriesPrediction) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	var needUpdate bool
	hpa, err := c.NewHPAObject(ctx, ehpa, substitute, tsp)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedCreateHPAObject", err.Error())
		klog.Errorf("Failed to create object, HorizontalPodAutoscaler %s error %v", klog.KObj(hpa), err)
		return nil, err
	}

	if !equality.Semantic.DeepEqual(&hpaExist.Spec, &hpa.Spec) {
		needUpdate = true
		hpaExist.Spec = hpa.Spec
	}

	if !equality.Semantic.DeepEqual(&hpaExist.Annotations, &hpa.Annotations) {
		needUpdate = true
		hpaExist.Annotations = hpa.Annotations
	}

	if !equality.Semantic.DeepEqual(&hpaExist.Labels, &hpa.Labels) {
		needUpdate = true
		hpaExist.Labels = hpa.Labels
	}

	if needUpdate {
		hpaCopy := hpaExist.DeepCopy()
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			err := c.Update(ctx, hpaCopy)
			if err == nil {
				return nil
			}

			updated := &autoscalingv2.HorizontalPodAutoscaler{}
			errGet := c.Get(context.TODO(), types.NamespacedName{Namespace: hpaCopy.Namespace, Name: hpaCopy.Name}, updated)
			if errGet == nil {
				updated.Labels = hpaCopy.Labels
				updated.Annotations = hpaCopy.Annotations
				updated.Spec = hpaCopy.Spec
				hpaCopy = updated
			}

			return err
		})

		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedUpdateHPA", err.Error())
			klog.Errorf("Failed to update HorizontalPodAutoscaler %s error %v", klog.KObj(hpaExist), err)
			return nil, err
		}

		klog.Infof("Update HorizontalPodAutoscaler successful, HorizontalPodAutoscaler %s", klog.KObj(hpaExist))
	}

	return hpaExist, nil
}

// GetHPAMetrics loop metricSpec in EffectiveHorizontalPodAutoscaler and generate metricSpec for HPA
func (c *EffectiveHPAController) GetHPAMetrics(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, tsp *predictionapi.TimeSeriesPrediction) ([]autoscalingv2.MetricSpec, error) {
	var metrics []autoscalingv2.MetricSpec
	for _, metric := range ehpa.Spec.Metrics {
		copyMetric := metric.DeepCopy()
		metrics = append(metrics, *copyMetric)
	}

	if utils.IsEHPAPredictionEnabled(ehpa) {
		var metricsForPrediction []autoscalingv2.MetricSpec
		for _, metric := range metrics {
			var metricIdentifier string
			var averageValue *resource.Quantity
			switch metric.Type {
			case autoscalingv2.ResourceMetricSourceType:
				metricIdentifier = utils.GetMetricIdentifier(metric, metric.Resource.Name.String())
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

					if len(pods) == 0 {
						return nil, fmt.Errorf("no pods returns from scale object. ")
					}

					availablePods := utils.GetAvailablePods(pods)
					if len(availablePods) == 0 {
						return nil, fmt.Errorf("failed to get available pods. ")
					}

					requests, err := utils.CalculatePodRequests(availablePods, metric.Resource.Name)
					if err != nil {
						return nil, err
					}

					value := int64((float64(requests) * float64(*metric.Resource.Target.AverageUtilization) / 100) / float64(len(availablePods)))
					averageValue = resource.NewMilliQuantity(value, resource.DecimalSI)
				} else {
					averageValue = metric.Resource.Target.AverageValue
				}
			case autoscalingv2.ExternalMetricSourceType:
				metricIdentifier = utils.GetMetricIdentifier(metric, metric.External.Metric.Name)
				averageValue = metric.External.Target.AverageValue
			case autoscalingv2.PodsMetricSourceType:
				metricIdentifier = utils.GetMetricIdentifier(metric, metric.Pods.Metric.Name)
				averageValue = metric.Pods.Target.AverageValue
			}

			if metricIdentifier == "" {
				continue
			}

			name := utils.GetPredictionMetricName(metric.Type)
			if len(name) == 0 {
				continue
			}

			if !utils.IsEHPAHasPredictionMetric(ehpa) {
				continue
			}

			if _, err := utils.GetReadyPredictionMetric(name, metricIdentifier, tsp); err != nil {
				// metric is not predictable
				continue
			}

			// generate an external metric for Prediction metric
			external := &autoscalingv2.ExternalMetricSource{
				Metric: autoscalingv2.MetricIdentifier{
					Name: name,
					// add known.EffectiveHorizontalPodAutoscalerUidLabel=uid in metric.selector
					// MetricAdapter use label selector to match the matching TimeSeriesPrediction to return metrics
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"targetKind":         ehpa.Spec.ScaleTargetRef.Kind,
							"targetName":         ehpa.Spec.ScaleTargetRef.Name,
							"targetNamespace":    ehpa.Namespace,
							"resourceIdentifier": metricIdentifier,
						},
					},
				},
				Target: autoscalingv2.MetricTarget{
					Type:         autoscalingv2.AverageValueMetricType,
					AverageValue: averageValue,
				},
			}
			metricSpec := autoscalingv2.MetricSpec{External: external, Type: autoscalingv2.ExternalMetricSourceType}
			metricsForPrediction = append(metricsForPrediction, metricSpec)
		}

		metrics = append(metrics, metricsForPrediction...)
	}

	// Construct cron external metrics for cron scale
	if utils.IsEHPACronEnabled(ehpa) {
		metrics = append(metrics, GetCronMetricSpecsForHPA(ehpa)...)
	}

	return metrics, nil
}

// GetCronMetricSpecsForHPA return a hpa external metric specs from ehpa cron scale specs, this spec will be injected into hpa
// One ehpa mapping to one cron metric only, even though there are multiple cron specs
func GetCronMetricSpecsForHPA(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) []autoscalingv2.MetricSpec {
	var metricSpecs []autoscalingv2.MetricSpec
	metricSpecs = append(metricSpecs, autoscalingv2.MetricSpec{
		Type: autoscalingv2.ExternalMetricSourceType,
		External: &autoscalingv2.ExternalMetricSource{
			Metric: autoscalingv2.MetricIdentifier{
				Name: utils.GetCronMetricName(),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"targetKind":         ehpa.Spec.ScaleTargetRef.Kind,
						"targetName":         ehpa.Spec.ScaleTargetRef.Name,
						"targetNamespace":    ehpa.Namespace,
						"resourceIdentifier": "cron",
					},
				},
			},
			Target: autoscalingv2.MetricTarget{
				Type:         autoscalingv2.AverageValueMetricType,
				AverageValue: resource.NewQuantity(int64(metricprovider.DefaultCronTargetMetricValue), resource.DecimalSI),
			},
		},
	})
	return metricSpecs
}

func setHPACondition(status *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus, conditions []autoscalingv2.HorizontalPodAutoscalerCondition) {
	for _, cond := range conditions {
		setCondition(status, autoscalingapi.ConditionType(cond.Type), metav1.ConditionStatus(cond.Status), cond.Reason, cond.Message)
	}
}

func (c *EffectiveHPAController) propagateLabelAndAnnotation(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, hpa *autoscalingv2.HorizontalPodAutoscaler) {
	for key, value := range ehpa.Labels {
		for _, prefix := range c.Config.PropagationConfig.LabelPrefixes {
			if strings.HasPrefix(key, prefix) {
				if hpa.Labels == nil {
					hpa.Labels = make(map[string]string)
				}
				hpa.Labels[key] = value
			}
		}
		for _, labelKey := range c.Config.PropagationConfig.Labels {
			if key == labelKey {
				if hpa.Labels == nil {
					hpa.Labels = make(map[string]string)
				}
				hpa.Labels[key] = value
			}
		}
	}

	for key, value := range ehpa.Annotations {
		for _, prefix := range c.Config.PropagationConfig.AnnotationPrefixes {
			if strings.HasPrefix(key, prefix) {
				if hpa.Annotations == nil {
					hpa.Annotations = make(map[string]string)
				}
				hpa.Annotations[key] = value
			}
		}
		for _, annoKey := range c.Config.PropagationConfig.Annotations {
			if key == annoKey {
				if hpa.Annotations == nil {
					hpa.Annotations = make(map[string]string)
				}
				hpa.Annotations[key] = value
			}
		}
	}
}
