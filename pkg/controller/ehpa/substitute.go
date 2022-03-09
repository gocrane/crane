package ehpa

import (
	"context"
	"fmt"

	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
)

func (c *EffectiveHPAController) ReconcileSubstitute(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	subsList := &autoscalingapi.SubstituteList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
	}
	err := c.Client.List(ctx, subsList, opts...)
	if err != nil {
		if errors.IsNotFound(err) {
			return c.CreateSubstitute(ctx, ehpa, scale)
		} else {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetSubstitute", err.Error())
			klog.Errorf("Failed to get Substitute, ehpa %s error %v", klog.KObj(ehpa), err)
			return nil, err
		}
	} else if len(subsList.Items) == 0 {
		return c.CreateSubstitute(ctx, ehpa, scale)
	}

	return c.UpdateSubstituteIfNeed(ctx, ehpa, &subsList.Items[0], scale)
}

func (c *EffectiveHPAController) CreateSubstitute(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	substitute, err := c.NewSubstituteObject(ehpa, scale)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateSubstituteObject", err.Error())
		klog.Errorf("Failed to create object, Substitute %s error %v", klog.KObj(substitute), err)
		return nil, err
	}

	err = c.Client.Create(ctx, substitute)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedCreateSubstitute", err.Error())
		klog.Errorf("Failed to create Substitute %s error %v", klog.KObj(substitute), err)
		return nil, err
	}

	klog.Infof("Create Substitute successfully, Substitute %s", klog.KObj(substitute))
	c.Recorder.Event(ehpa, v1.EventTypeNormal, "SubstituteCreated", "Create Substitute successfully")

	return substitute, nil
}

func (c *EffectiveHPAController) NewSubstituteObject(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	name := fmt.Sprintf("ehpa-%s", ehpa.Name)
	substitute := &autoscalingapi.Substitute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ehpa.Namespace, // the same namespace to effective-hpa
			Name:      name,
			Labels: map[string]string{
				"app.kubernetes.io/name":                       name,
				"app.kubernetes.io/part-of":                    ehpa.Name,
				"app.kubernetes.io/managed-by":                 known.EffectiveHorizontalPodAutoscalerManagedBy,
				known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID),
			},
		},
		Spec: autoscalingapi.SubstituteSpec{
			SubstituteTargetRef: ehpa.Spec.ScaleTargetRef,
			Replicas:            scale.Spec.Replicas,
		},
	}

	// EffectiveHPA control the underground substitute so set controller reference for substitute here
	if err := controllerutil.SetControllerReference(ehpa, substitute, c.Scheme); err != nil {
		return nil, err
	}

	return substitute, nil
}

func (c *EffectiveHPAController) UpdateSubstituteIfNeed(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, substituteExist *autoscalingapi.Substitute, scale *autoscalingapiv1.Scale) (*autoscalingapi.Substitute, error) {
	if !equality.Semantic.DeepEqual(&substituteExist.Spec.SubstituteTargetRef, &ehpa.Spec.ScaleTargetRef) {
		substituteExist.Spec.SubstituteTargetRef = ehpa.Spec.ScaleTargetRef
		err := c.Update(ctx, substituteExist)
		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedUpdateSubstitute", err.Error())
			klog.Errorf("Failed to update Substitute %s, error %v", klog.KObj(substituteExist), err)
			return nil, err
		}

		klog.Infof("Update Substitute successful, Substitute %s", klog.KObj(substituteExist))
	}

	return substituteExist, nil
}
