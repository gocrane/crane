package hpa

import (
	"context"
	"fmt"
	"time"

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
)

func (p *AdvancedHPAController) ReconcilePodPredication(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*predictionapi.PodGroupPrediction, error) {
	predictionList := &predictionapi.PodGroupPredictionList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID)}),
	}
	err := p.Client.List(ctx, predictionList, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			return p.CreatePodPrediction(ctx, ahpa)
		} else {
			p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedGetPrediction", err.Error())
			p.Log.Error(err, "Failed to get PodGroupPrediction", "advanced-hpa", klog.KObj(ahpa))
			return nil, err
		}
	} else if len(predictionList.Items) == 0 {
		return p.CreatePodPrediction(ctx, ahpa)
	}

	return p.UpdatePodPredictionIfNeed(ctx, ahpa, &predictionList.Items[0])
}

func (p *AdvancedHPAController) GetPodPredication(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*predictionapi.PodGroupPrediction, error) {
	predictionList := &predictionapi.PodGroupPredictionList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID)}),
	}
	err := p.Client.List(ctx, predictionList, opts...)
	if err != nil {
		return nil, err
	} else if len(predictionList.Items) == 0 {
		return nil, nil
	}

	return &predictionList.Items[0], nil
}

func (p *AdvancedHPAController) CreatePodPrediction(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*predictionapi.PodGroupPrediction, error) {
	podPrediction, err := p.NewPodPredictionObject(ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreatePredictionObject", err.Error())
		p.Log.Error(err, "Failed to create object", "PodGroupPrediction", podPrediction)
		return nil, err
	}

	err = p.Client.Create(ctx, podPrediction)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreatePrediction", err.Error())
		p.Log.Error(err, "Failed to create", "PodGroupPrediction", podPrediction)
		return nil, err
	}

	p.Log.Info("Create successfully", "PodGroupPrediction", klog.KObj(podPrediction))
	p.Recorder.Event(ahpa, v1.EventTypeNormal, "PodGroupPredictionCreated", "Create PodGroupPrediction successfully")

	return podPrediction, nil
}

func (p *AdvancedHPAController) UpdatePodPredictionIfNeed(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler, podPredictionExist *predictionapi.PodGroupPrediction) (*predictionapi.PodGroupPrediction, error) {
	podPrediction, err := p.NewPodPredictionObject(ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreatePredictionObject", err.Error())
		p.Log.Error(err, "Failed to create object", "PodGroupPrediction", podPrediction)
		return nil, err
	}

	if !equality.Semantic.DeepEqual(&podPredictionExist.Spec, &podPrediction.Spec) {
		p.Log.V(4).Info("PodGroupPrediction is unsynced according to AdvancedHorizontalPodAutoscaler, should be updated", "currentPodPrediction", podPredictionExist.Spec, "expectPodPrediction", podPrediction.Spec)

		podPredictionExist.Spec = podPrediction.Spec
		err := p.Update(ctx, podPredictionExist)
		if err != nil {
			p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedUpdatePrediction", err.Error())
			p.Log.Error(err, "Failed to update", "PodGroupPrediction", podPredictionExist)
			return nil, err
		}

		p.Log.Info("Update PodGroupPrediction successful", "PodGroupPrediction", klog.KObj(podPrediction))
	}

	return podPredictionExist, nil
}

func (p *AdvancedHPAController) NewPodPredictionObject(ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*predictionapi.PodGroupPrediction, error) {
	name := fmt.Sprintf("advancedhpa-%s", ahpa.Name)
	prediction := &predictionapi.PodGroupPrediction{
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
		Spec: predictionapi.PodGroupPredictionSpec{
			PredictionWindow:        metav1.Duration{Duration: time.Duration(*ahpa.Spec.Prediction.PredictionWindowSeconds) * time.Second},
			Mode:                    predictionapi.PredictionModeRange,
			MetricPredictionConfigs: make([]predictionapi.MetricPredictionConfig, 0),
			WorkloadRef:             &ahpa.Spec.ScaleTargetRef,
		},
	}

	var metricPredictionConfigs []predictionapi.MetricPredictionConfig
	for _, metric := range ahpa.Spec.Metrics {
		// Convert resource metric into prediction metric
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			metricName, err := GetPredictionMetricName(metric.Resource.Name)
			if err != nil {
				return nil, err
			}

			metricPredictionConfigs = append(metricPredictionConfigs, predictionapi.MetricPredictionConfig{
				MetricName:    metricName,
				AlgorithmType: ahpa.Spec.Prediction.PredictionAlgorithm.AlgorithmType,
				DSP:           ahpa.Spec.Prediction.PredictionAlgorithm.DSP.DeepCopy(),
				Percentile:    ahpa.Spec.Prediction.PredictionAlgorithm.Percentile.DeepCopy(),
			})
		}
	}
	prediction.Spec.MetricPredictionConfigs = metricPredictionConfigs

	// AdvancedHPA control the underground prediction so set controller reference for it here
	if err := controllerutil.SetControllerReference(ahpa, prediction, p.Scheme); err != nil {
		return nil, err
	}

	return prediction, nil
}

func IsPredictionEnabled(ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) bool {
	return ahpa.Spec.Prediction != nil && ahpa.Spec.Prediction.PredictionWindowSeconds != nil && ahpa.Spec.Prediction.PredictionAlgorithm != nil
}

func setPredictionCondition(status *autoscalingapi.AdvancedHorizontalPodAutoscalerStatus, conditions []predictionapi.PodGroupPredictionCondition) {
	for _, cond := range conditions {
		if cond.Type == predictionapi.PredictionConditionPredicting {
			if len(cond.Reason) > 0 && len(cond.Message) > 0 {
				setCondition(status, autoscalingapi.PredictionReady, cond.Status, cond.Reason, cond.Message)
			}
		}
	}
}
