package hpa

import (
	"context"
	"fmt"
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	"github.com/gocrane/crane/pkg/known"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (p *EffectiveHPAController) ReconcileSubstitute(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	subsList := &autoscalingapi.SubstituteList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
	}
	err := p.Client.List(ctx, subsList, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			return p.CreateSubstitute(ctx, ehpa, scale)
		} else {
			p.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetSubstitute", err.Error())
			p.Log.Error(err, "Failed to get Substitute", "effective-hpa", klog.KObj(ehpa))
			return nil, err
		}
	} else if len(subsList.Items) == 0 {
		return p.CreateSubstitute(ctx, ehpa, scale)
	}

	return &subsList.Items[0], nil
}


func (p *EffectiveHPAController) CreateSubstitute(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	substitute, err := p.NewSubstituteObject(ctx, ehpa, scale)
	if err != nil {
		p.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateSubstituteObject", err.Error())
		p.Log.Error(err, "Failed to create object", "Substitute", substitute)
		return nil, err
	}

	err = p.Client.Create(ctx, substitute)
	if err != nil {
		p.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateSubstitute", err.Error())
		p.Log.Error(err, "Failed to create", "Substitute", substitute)
		return nil, err
	}

	p.Log.Info("Create Substitute successfully", "Substitute", substitute)
	p.Recorder.Event(ehpa, v1.EventTypeNormal, "SubstituteCreated", "Create Substitute successfully")

	return substitute, nil
}

func (p *EffectiveHPAController) NewSubstituteObject(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	name := fmt.Sprintf("ehpa-%s", ehpa.Name)
	substitute := &autoscalingapi.Substitute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ehpa.Namespace, // the same namespace to effective-hpa
			Name:      name,
			Labels: map[string]string{
				"app.kubernetes.io/name":                      name,
				"app.kubernetes.io/part-of":                   ehpa.Name,
				"app.kubernetes.io/managed-by":                known.EffectiveHorizontalPodAutoscalerManagedBy,
				known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID),
			},
		},
		Spec: autoscalingapi.SubstituteSpec{
			SubstituteTargetRef: ehpa.Spec.ScaleTargetRef,
			Replicas: &scale.Spec.Replicas,
		},
		Status: autoscalingapi.SubstituteStatus{
			LabelSelector: scale.Status.Selector,
			Replicas: scale.Status.Replicas,
		},
	}

	// EffectiveHPA control the underground substitute so set controller reference for substitute here
	if err := controllerutil.SetControllerReference(ehpa, substitute, p.Scheme); err != nil {
		return nil, err
	}

	return substitute, nil
}

func (p *EffectiveHPAController) UpdateSubstituteIfNeed(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substituteExist *autoscalingapi.Substitute, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	substitute, err := p.NewSubstituteObject(ctx, ehpa, scale)
	if err != nil {
		p.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateSubstituteObject", err.Error())
		p.Log.Error(err, "Failed to create object", "Substitute", substitute)
		return nil, err
	}

	if !equality.Semantic.DeepEqual(&substituteExist.Spec, &substitute.Spec) {
		p.Log.V(4).Info("Substitute is unsynced according to EffectiveHorizontalPodAutoscaler, should be updated", "currentSubstitute", substituteExist.Spec, "expectSubstitute", substitute.Spec)

		substituteExist.Spec = substitute.Spec
		err := p.Update(ctx, substituteExist)
		if err != nil {
			p.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedUpdateSubstitute", err.Error())
			p.Log.Error(err, "Failed to update", "Substitute", substituteExist)
			return nil, err
		}

		p.Log.Info("Update Substitute successful", "Substitute", substituteExist)
	}

	return substituteExist, nil
}