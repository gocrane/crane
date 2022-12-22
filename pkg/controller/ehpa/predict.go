package ehpa

import (
	"context"
	"fmt"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	prometheus_adapter "github.com/gocrane/crane/pkg/prometheus-adapter"
	"github.com/gocrane/crane/pkg/utils"
)

func (c *EffectiveHPAController) ReconcilePredication(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) (*predictionapi.TimeSeriesPrediction, error) {
	predictionList := &predictionapi.TimeSeriesPredictionList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
	}
	err := c.Client.List(ctx, predictionList, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			return c.CreatePrediction(ctx, ehpa)
		} else {
			c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedGetPrediction", err.Error())
			klog.Errorf("Failed to get TimeSeriesPrediction, ehpa %s error %v", klog.KObj(ehpa), err)
			return nil, err
		}
	} else if len(predictionList.Items) == 0 {
		return c.CreatePrediction(ctx, ehpa)
	}

	return c.UpdatePredictionIfNeed(ctx, ehpa, &predictionList.Items[0])
}

func (c *EffectiveHPAController) GetPredication(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) (*predictionapi.TimeSeriesPrediction, error) {
	predictionList := &predictionapi.TimeSeriesPredictionList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
	}
	err := c.Client.List(ctx, predictionList, opts...)
	if err != nil {
		return nil, err
	} else if len(predictionList.Items) == 0 {
		return nil, nil
	}

	return &predictionList.Items[0], nil
}

func (c *EffectiveHPAController) CreatePrediction(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) (*predictionapi.TimeSeriesPrediction, error) {
	prediction, err := c.NewPredictionObject(ehpa)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedCreatePredictionObject", err.Error())
		klog.Errorf("Failed to create object, TimeSeriesPrediction %s error %v", klog.KObj(prediction), err)
		return nil, err
	}

	err = c.Client.Create(ctx, prediction)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedCreatePrediction", err.Error())
		klog.Errorf("Failed to create TimeSeriesPrediction %s error %v", klog.KObj(prediction), err)
		return nil, err
	}

	klog.Infof("Creation TimeSeriesPrediction %s successfully", klog.KObj(prediction))
	c.Recorder.Event(ehpa, v1.EventTypeNormal, "PredictionCreated", "Create TimeSeriesPrediction successfully")

	return prediction, nil
}

func (c *EffectiveHPAController) UpdatePredictionIfNeed(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, predictionExist *predictionapi.TimeSeriesPrediction) (*predictionapi.TimeSeriesPrediction, error) {
	prediction, err := c.NewPredictionObject(ehpa)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedCreatePredictionObject", err.Error())
		klog.Errorf("Failed to create object, TimeSeriesPrediction %s error %v", klog.KObj(prediction), err)
		return nil, err
	}

	if !equality.Semantic.DeepEqual(&predictionExist.Spec, &prediction.Spec) {
		predictionExist.Spec = prediction.Spec
		err := c.Update(ctx, predictionExist)
		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeWarning, "FailedUpdatePrediction", err.Error())
			klog.Errorf("Failed to update TimeSeriesPrediction %s", klog.KObj(predictionExist))
			return nil, err
		}

		klog.Infof("Update TimeSeriesPrediction successful, TimeSeriesPrediction %s", klog.KObj(predictionExist))
	}

	return predictionExist, nil
}

func (c *EffectiveHPAController) NewPredictionObject(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) (*predictionapi.TimeSeriesPrediction, error) {
	name := fmt.Sprintf("ehpa-%s", ehpa.Name)
	prediction := &predictionapi.TimeSeriesPrediction{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ehpa.Namespace, // the same namespace to effectivehpa
			Name:      name,
			Labels: map[string]string{
				"app.kubernetes.io/name":                       name,
				"app.kubernetes.io/part-of":                    ehpa.Name,
				"app.kubernetes.io/managed-by":                 known.EffectiveHorizontalPodAutoscalerManagedBy,
				"app.kubernetes.io/target-kind":                ehpa.Spec.ScaleTargetRef.Kind,
				"app.kubernetes.io/target-namespace":           ehpa.Namespace,
				"app.kubernetes.io/target-name":                ehpa.Spec.ScaleTargetRef.Name,
				known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID),
			},
		},
		Spec: predictionapi.TimeSeriesPredictionSpec{
			PredictionWindowSeconds: *ehpa.Spec.Prediction.PredictionWindowSeconds,
			TargetRef: v1.ObjectReference{
				Kind:       ehpa.Spec.ScaleTargetRef.Kind,
				Namespace:  ehpa.Namespace,
				Name:       ehpa.Spec.ScaleTargetRef.Name,
				APIVersion: ehpa.Spec.ScaleTargetRef.APIVersion,
			},
		},
	}

	// get MetricRules
	mrs := prometheus_adapter.GetMetricRules()

	var predictionMetrics []predictionapi.PredictionMetric
	for _, metric := range ehpa.Spec.Metrics {
		var metricName string
		//get metricIdentifier by metric.Type and metricName
		var metricIdentifier string
		switch metric.Type {
		case autoscalingv2.ResourceMetricSourceType:
			metricName = metric.Resource.Name.String()
			metricIdentifier = utils.GetMetricIdentifier(metric, metric.Resource.Name.String())
		case autoscalingv2.ExternalMetricSourceType:
			metricName = metric.External.Metric.Name
			metricIdentifier = utils.GetMetricIdentifier(metric, metric.External.Metric.Name)
		case autoscalingv2.PodsMetricSourceType:
			metricName = metric.Pods.Metric.Name
			metricIdentifier = utils.GetMetricIdentifier(metric, metric.Pods.Metric.Name)
		}

		if metricIdentifier == "" {
			continue
		}

		//get matchLabels
		var matchLabels []string
		var metricRule *prometheus_adapter.MetricRule

		// Supreme priority: annotation
		expressionQuery := utils.GetExpressionQueryAnnotation(metricIdentifier, ehpa.Annotations)
		if expressionQuery == "" {
			var nameReg string
			// get metricRule from prometheus-adapter
			switch metric.Type {
			case autoscalingv2.ResourceMetricSourceType:
				if len(mrs.MetricRulesResource) > 0 {
					metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesResource, metricName)
					if metricRule == nil {
						klog.Errorf("Got MetricRulesResource prometheus-adapter-resource Failed MetricName[%s]", metricName)
					} else {
						klog.V(4).Infof("Got MetricRulesResource prometheus-adapter-resource MetricMatches[%s] SeriesName[%s]", metricRule.MetricMatches, metricRule.SeriesName)
						nameReg = utils.GetPodNameReg(ehpa.Spec.ScaleTargetRef.Name, ehpa.Spec.ScaleTargetRef.Kind)
					}
				}
			case autoscalingv2.PodsMetricSourceType:
				if len(mrs.MetricRulesCustomer) > 0 {
					metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesCustomer, metricName)
					if metricRule == nil {
						klog.Errorf("Got MetricRulesCustomer prometheus-adapter-customer Failed MetricName[%s]", metricName)
					} else {
						klog.V(4).Infof("Got MetricRulesCustomer prometheus-adapter-customer MetricMatches[%s] SeriesName[%s]", metricRule.MetricMatches, metricRule.SeriesName)
						nameReg = utils.GetPodNameReg(ehpa.Spec.ScaleTargetRef.Name, ehpa.Spec.ScaleTargetRef.Kind)

						if metric.Pods.Metric.Selector != nil {
							for _, i := range utils.MapSortToArray(metric.Pods.Metric.Selector.MatchLabels) {
								matchLabels = append(matchLabels, i)
							}
						}
					}
				}
			case autoscalingv2.ExternalMetricSourceType:
				if len(mrs.MetricRulesExternal) > 0 {
					metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, metricName)
					if metricRule == nil {
						klog.Errorf("Got MetricRulesExternal prometheus-adapter-external Failed MetricName[%s]", metricName)
					} else {
						klog.V(4).Infof("Got MetricRulesExternal prometheus-adapter-external MetricMatches[%s] SeriesName[%s]", metricRule.MetricMatches, metricRule.SeriesName)
						if metric.External.Metric.Selector != nil {
							for _, i := range utils.MapSortToArray(metric.External.Metric.Selector.MatchLabels) {
								matchLabels = append(matchLabels, i)
							}
						}
					}
				}
			}

			if metricRule != nil {
				// Second priority: get default expressionQuery
				var err error
				expressionQuery, err = metricRule.QueryForSeries(ehpa.Namespace, nameReg, append(mrs.ExtensionLabels, matchLabels...))
				if err != nil {
					klog.Errorf("Got promSelector prometheus-adapter %v %v", metricRule, err)
				} else {
					klog.V(4).Infof("Got expressionQuery [%s] from prometheus-adapter ", expressionQuery)
				}
			}

			// Third priority: get default expressionQuery
			if expressionQuery == "" {
				//if annotation not matched, and configmap is not set, build expressionQuerydefault by metric and ehpa.TargetName
				expressionQuery = utils.GetExpressionQueryDefault(metric, ehpa.Namespace, ehpa.Spec.ScaleTargetRef.Name, ehpa.Spec.ScaleTargetRef.Kind)
				klog.V(4).Infof("Got expressionQuery [%s] by default", expressionQuery)
			}
		}

		if expressionQuery == "" {
			continue
		}

		predictionMetrics = append(predictionMetrics, predictionapi.PredictionMetric{
			ResourceIdentifier: metricIdentifier,
			Type:               predictionapi.ExpressionQueryMetricType,
			ExpressionQuery: &predictionapi.ExpressionQuery{
				Expression: expressionQuery,
			},
			Algorithm: predictionapi.Algorithm{
				AlgorithmType: ehpa.Spec.Prediction.PredictionAlgorithm.AlgorithmType,
				DSP:           ehpa.Spec.Prediction.PredictionAlgorithm.DSP,
				Percentile:    ehpa.Spec.Prediction.PredictionAlgorithm.Percentile,
			},
		})
	}
	prediction.Spec.PredictionMetrics = predictionMetrics

	// EffectiveHPA control the underground prediction so set controller reference for it here
	if err := controllerutil.SetControllerReference(ehpa, prediction, c.Scheme); err != nil {
		return nil, err
	}

	return prediction, nil
}

func setPredictionCondition(status *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus, conditions []metav1.Condition) {
	for _, cond := range conditions {
		if cond.Type == string(predictionapi.TimeSeriesPredictionConditionReady) {
			if len(cond.Reason) > 0 {
				setCondition(status, autoscalingapi.PredictionReady, cond.Status, cond.Reason, cond.Message)
			}
		}
	}
}
