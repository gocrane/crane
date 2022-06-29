package ehpa

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetPrediction", err.Error())
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
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreatePredictionObject", err.Error())
		klog.Errorf("Failed to create object, TimeSeriesPrediction %s error %v", klog.KObj(prediction), err)
		return nil, err
	}

	err = c.Client.Create(ctx, prediction)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreatePrediction", err.Error())
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
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreatePredictionObject", err.Error())
		klog.Errorf("Failed to create object, TimeSeriesPrediction %s error %v", klog.KObj(prediction), err)
		return nil, err
	}

	if !equality.Semantic.DeepEqual(&predictionExist.Spec, &prediction.Spec) {
		predictionExist.Spec = prediction.Spec
		err := c.Update(ctx, predictionExist)
		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedUpdatePrediction", err.Error())
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

	var predictionMetrics []predictionapi.PredictionMetric
	for _, metric := range ehpa.Spec.Metrics {
		// Convert resource metric into prediction metric
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			metricName := utils.GetPredictionMetricName(metric.Resource.Name)
			if len(metricName) == 0 {
				continue
			}

			predictionMetrics = append(predictionMetrics, predictionapi.PredictionMetric{
				ResourceIdentifier: metricName,
				Type:               predictionapi.ResourceQueryMetricType,
				ResourceQuery:      &metric.Resource.Name,
				Algorithm: predictionapi.Algorithm{
					AlgorithmType: ehpa.Spec.Prediction.PredictionAlgorithm.AlgorithmType,
					DSP:           ehpa.Spec.Prediction.PredictionAlgorithm.DSP,
					Percentile:    ehpa.Spec.Prediction.PredictionAlgorithm.Percentile,
				},
			})
		}
		if metric.Type == autoscalingv2.ExternalMetricSourceType {
			metricName := utils.GetExternalPredictionMetricName(metric.External.Metric.Name)

			if len(metricName) == 0 {
				continue
			}

			expressionQuery := getExpressionQuery(metric.External.Metric.Name, ehpa.Annotations)
			if len(expressionQuery) == 0 {
				continue
			}

			predictionMetrics = append(predictionMetrics, predictionapi.PredictionMetric{
				ResourceIdentifier: metricName,
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
	}
	prediction.Spec.PredictionMetrics = predictionMetrics

	// EffectiveHPA control the underground prediction so set controller reference for it here
	if err := controllerutil.SetControllerReference(ehpa, prediction, c.Scheme); err != nil {
		return nil, err
	}

	return prediction, nil
}

func getExpressionQuery(metricName string, annotations map[string]string) string {
	for k, v := range annotations {
		if strings.HasPrefix(k, known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix) {
			compileRegex := regexp.MustCompile(fmt.Sprintf("%s(.*)", known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix))
			matchArr := compileRegex.FindStringSubmatch(k)
			if len(matchArr) == 2 && matchArr[1][1:] == metricName {
				return v
			}
		}
	}

	return ""
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

func isPredictionReady(status *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus) bool {
	for _, cond := range status.Conditions {
		if cond.Type == string(autoscalingapi.PredictionReady) && cond.Status == metav1.ConditionTrue {
			return true
		}
	}

	return false
}
