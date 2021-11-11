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
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane-io/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane-io/api/prediction/v1alpha1"
	"github.com/gocrane-io/crane/pkg/known"
)

func (p *AdvancedHPAController) ReconcilePodPredication(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) error {
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
			p.Log.Error(err, "Failed to get PodGroupPrediction", "ahpa.UID", ahpa.UID)
			return err
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

func (p *AdvancedHPAController) CreatePodPrediction(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) error {
	podPrediction := p.NewPodPredictionObject(ahpa)
	err := p.Client.Create(ctx, podPrediction)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreatePrediction", err.Error())
		p.Log.Error(err, "Failed to create", "PodGroupPrediction", podPrediction)
		return err
	}

	p.Log.Info("Create PodGroupPrediction successfully", "ahpa.Namespace", ahpa.Namespace, "ahpa.Name", ahpa.Name)
	p.Recorder.Event(ahpa, v1.EventTypeNormal, "PodGroupPredictionCreated", "Create PodGroupPrediction successfully")

	return nil
}

func (p *AdvancedHPAController) UpdatePodPredictionIfNeed(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler, podPredictionExist *predictionapi.PodGroupPrediction) error {
	podPrediction := p.NewPodPredictionObject(ahpa)

	if !equality.Semantic.DeepEqual(podPredictionExist.Spec, podPrediction.Spec) {
		p.Log.Info("PodGroupPrediction is unsynced according to AdvancedHorizontalPodAutoscaler, should be updated", "currentPodPrediction", podPredictionExist.Spec, "expectPodPrediction", podPrediction.Spec)

		podPredictionExist.Spec = podPrediction.Spec
		err := p.Update(ctx, podPredictionExist)
		if err != nil {
			p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedUpdatePrediction", err.Error())
			p.Log.Error(err, "Failed to update", "PodGroupPrediction", podPredictionExist)
			return err
		}

		p.Log.Info("Update PodGroupPrediction successful", "ahpa.Namespace", ahpa.Namespace, "ahpa.Name", ahpa.Name)
	}

	return nil
}

func (p *AdvancedHPAController) NewPodPredictionObject(ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) *predictionapi.PodGroupPrediction {
	name := fmt.Sprintf("ahpa-%s", ahpa.Name)
	prediction := &predictionapi.PodGroupPrediction{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ahpa.Namespace, // the same namespace to ahpa
			Name:      name,
			Labels: map[string]string{
				"app.kubernetes.io/name":                      name,
				"app.kubernetes.io/part-of":                   ahpa.Name,
				"app.kubernetes.io/managed-by":                "advanced-hpa-controller",
				known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID),
			},
		},
		Spec: predictionapi.PodGroupPredictionSpec{
			PredictionWindow:        metav1.Duration{Duration: time.Duration(*ahpa.Spec.PredictionConfig.PredictionWindow) * time.Second},
			Mode:                    predictionapi.PredictionModeRange,
			MetricPredictionConfigs: make([]predictionapi.MetricPredictionConfig, 0),
			WorkloadRef:             &ahpa.Spec.ScaleTargetRef,
		},
	}

	var metricPredictionConfigs []predictionapi.MetricPredictionConfig
	for _, metric := range ahpa.Spec.Metrics {
		// Only support external metric now
		if metric.Type == autoscalingv2.ExternalMetricSourceType {
			metricPredictionConfigs = append(metricPredictionConfigs, predictionapi.MetricPredictionConfig{
				MetricName:    metric.External.Metric.Name,
				AlgorithmType: ahpa.Spec.PredictionConfig.PredictionAlgorithm.AlgorithmType,
				DSP:           ahpa.Spec.PredictionConfig.PredictionAlgorithm.DSP,
				Percentile:    ahpa.Spec.PredictionConfig.PredictionAlgorithm.Percentile,
			})
		}
	}
	prediction.Spec.MetricPredictionConfigs = metricPredictionConfigs

	return prediction
}
