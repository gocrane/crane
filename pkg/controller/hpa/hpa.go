package hpa

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	autoscalingapi "github.com/gocrane-io/api/autoscaling/v1alpha1"
	"github.com/gocrane-io/crane/pkg/known"
)

func (p *AdvancedHPAController) ReconcileHPA(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) error {
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
			p.Log.Error(err, "Failed to get HPA", "ahpa.UID", ahpa.UID)
			return err
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

func (p *AdvancedHPAController) CreateHPA(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) error {
	hpa, err := p.NewHPAObject(ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreateHPAObject", err.Error())
		p.Log.Error(err, "Failed to create object", "HorizontalPodAutoscaler", hpa)
		return err
	}

	err = p.Client.Create(ctx, hpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreateHPA", err.Error())
		p.Log.Error(err, "Failed to create", "HorizontalPodAutoscaler", hpa)
		return err
	}

	p.Log.Info("Create HorizontalPodAutoscaler successfully", "ahpa.Namespace", ahpa.Namespace, "ahpa.Name", ahpa.Name)
	p.Recorder.Event(ahpa, v1.EventTypeNormal, "HPACreated", "Create HorizontalPodAutoscaler successfully")

	return nil
}

func (p *AdvancedHPAController) NewHPAObject(ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	name := fmt.Sprintf("ahpa-%s", ahpa.Name)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
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
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: ahpa.Spec.ScaleTargetRef,
			MinReplicas:    ahpa.Spec.MinReplicas,
			MaxReplicas:    ahpa.Spec.MaxReplicas,
			Metrics:        p.GetHPAMetrics(ahpa),
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

func (p *AdvancedHPAController) UpdateHPAIfNeed(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler, hpaExist *autoscalingv2.HorizontalPodAutoscaler) error {
	hpa, err := p.NewHPAObject(ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedCreateHPAObject", err.Error())
		p.Log.Error(err, "Failed to create object", "HorizontalPodAutoscaler", hpa)
		return err
	}

	if !equality.Semantic.DeepEqual(hpaExist.Spec, hpa.Spec) {
		p.Log.Info("HorizontalPodAutoscaler is unsynced according to AdvancedHorizontalPodAutoscaler, should be updated", "currentHPA", hpaExist.Spec, "expectHPA", hpa.Spec)

		hpaExist.Spec = hpa.Spec
		err := p.Update(ctx, hpa)
		if err != nil {
			p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedUpdateHPA", err.Error())
			p.Log.Error(err, "Failed to update", "HorizontalPodAutoscaler", hpaExist)
			return err
		}

		p.Log.Info("Update HorizontalPodAutoscaler successful", "ahpa.Namespace", ahpa.Namespace, "ahpa.Name", ahpa.Name)
	}

	return nil
}

func (p *AdvancedHPAController) GetHPAMetrics(ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) []autoscalingv2.MetricSpec {
	var metrics []autoscalingv2.MetricSpec
	for _, metric := range ahpa.Spec.Metrics {
		copyMetric := metric.DeepCopy()
		if metric.Type == autoscalingv2.ExternalMetricSourceType {
			// add known.AdvancedHorizontalPodAutoscalerUidLabel=uid in metric.selector
			// MetricAdapter use label selector to match the matching PodGroupPrediction to return metrics
			copyMetric.External.Metric.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					known.AdvancedHorizontalPodAutoscalerUidLabel: string(ahpa.UID),
				},
			}
		}
		metrics = append(metrics, *copyMetric)
	}

	return metrics
}
