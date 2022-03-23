package recommendation

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	vpatypes "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

func (c *Controller) UpdateRecommendation(ctx context.Context, recommendation *analysisapi.Recommendation, proposed *types.ProposedRecommendation, status *analysisapi.RecommendationStatus) error {
	var value string
	if proposed.ResourceRequest != nil {
		valueBytes, err := yaml.Marshal(proposed.ResourceRequest)
		if err != nil {
			return err
		}
		value = string(valueBytes)
	} else if proposed.EffectiveHPA != nil {
		valueBytes, err := yaml.Marshal(proposed.EffectiveHPA)
		if err != nil {
			return err
		}
		value = string(valueBytes)
	}

	status.RecommendedValue = value
	if recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeStatus {
		return nil
	}

	unstructed := &unstructured.Unstructured{}
	unstructed.SetAPIVersion(recommendation.Spec.TargetRef.APIVersion)
	unstructed.SetKind(recommendation.Spec.TargetRef.Kind)
	err := c.Client.Get(ctx, client.ObjectKey{Name: recommendation.Spec.TargetRef.Name, Namespace: recommendation.Spec.TargetRef.Namespace}, unstructed)
	if err != nil {
		return fmt.Errorf("Get target object failed: %v. ", err)
	}

	if recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeStatusAndAnnotation || recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeAuto {
		annotation := unstructed.GetAnnotations()
		if annotation == nil {
			annotation = map[string]string{}
		}

		switch recommendation.Spec.Type {
		case analysisapi.AnalysisTypeResource:
			annotation[known.ResourceRecommendationValueAnnotation] = value
		case analysisapi.AnalysisTypeHPA:
			annotation[known.HPARecommendationValueAnnotation] = value
		}

		unstructed.SetAnnotations(annotation)
		err = c.Client.Update(ctx, unstructed)
		if err != nil {
			return fmt.Errorf("Update target annotation failed: %v. ", err)
		}
	}

	// Only support Auto Type for EHPA recommendation
	if recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeAuto {
		if proposed.EffectiveHPA != nil {
			ehpa, err := utils.GetEHPAFromScaleTarget(ctx, c.Client, recommendation.Namespace, recommendation.Spec.TargetRef)
			if err != nil {
				return fmt.Errorf("Get EHPA from target failed: %v. ", err)
			}
			if ehpa == nil {
				ehpa = &autoscalingapi.EffectiveHorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: recommendation.Spec.TargetRef.Namespace,
						Name:      recommendation.Spec.TargetRef.Name,
					},
					Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
						MinReplicas:   proposed.EffectiveHPA.MinReplicas,
						MaxReplicas:   *proposed.EffectiveHPA.MaxReplicas,
						Metrics:       proposed.EffectiveHPA.Metrics,
						ScaleStrategy: autoscalingapi.ScaleStrategyPreview,
						Prediction:    proposed.EffectiveHPA.Prediction,
						ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
							Kind:       recommendation.Spec.TargetRef.Kind,
							APIVersion: recommendation.Spec.TargetRef.APIVersion,
							Name:       recommendation.Spec.TargetRef.Name,
						},
					},
				}

				err = c.Client.Create(ctx, ehpa)
				if err == nil {
					c.Recorder.Event(ehpa, v1.EventTypeNormal, "UpdateValue", "Created EffectiveHorizontalPodAutoscaler.")
					klog.Infof("Create EffectiveHorizontalPodAutoscaler successfully, recommendation %s", klog.KObj(recommendation))
				}
				return err
			} else {
				// we don't override ScaleStrategy, because we always use preview to be the default version,
				// if user change it, we don't want to override it.
				// The reason for Prediction is the same.
				ehpaUpdate := ehpa.DeepCopy()
				ehpaUpdate.Spec.MinReplicas = proposed.EffectiveHPA.MinReplicas
				ehpaUpdate.Spec.MaxReplicas = *proposed.EffectiveHPA.MaxReplicas
				ehpaUpdate.Spec.Metrics = proposed.EffectiveHPA.Metrics

				if !equality.Semantic.DeepEqual(&ehpaUpdate.Spec, &ehpa.Spec) {
					err = c.Client.Update(ctx, ehpaUpdate)
					if err == nil {
						c.Recorder.Event(ehpa, v1.EventTypeNormal, "UpdateValue", "Updated EffectiveHorizontalPodAutoscaler.")
						klog.Infof("Update EffectiveHorizontalPodAutoscaler successfully, recommendation %s", klog.KObj(recommendation))
					}
					return err
				}
			}
		}

		if proposed.ResourceRequest != nil {
			evpa, err := utils.GetEVPAFromScaleTarget(ctx, c.Client, recommendation.Namespace, recommendation.Spec.TargetRef)
			if err != nil {
				return fmt.Errorf("Get EVPA from target failed: %v. ", err)
			}
			if evpa == nil {
				off := vpatypes.UpdateModeOff
				var policies []autoscalingapi.ContainerResourcePolicy
				podTemplate, err := utils.GetPodTemplate(ctx, evpa.Namespace, evpa.Spec.TargetRef.Name, evpa.Spec.TargetRef.Kind, evpa.Spec.TargetRef.APIVersion, c.Client)
				if err != nil {
					return err
				}

				for _, container := range podTemplate.Spec.Containers {
					policy := autoscalingapi.ContainerResourcePolicy{
						ContainerName: container.Name,
					}
					policies = append(policies, policy)
				}

				evpa = &autoscalingapi.EffectiveVerticalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: recommendation.Spec.TargetRef.Namespace,
						Name:      recommendation.Spec.TargetRef.Name,
					},
					Spec: autoscalingapi.EffectiveVerticalPodAutoscalerSpec{
						ResourcePolicy: &autoscalingapi.PodResourcePolicy{
							ContainerPolicies: policies,
						},
						UpdatePolicy: &vpatypes.PodUpdatePolicy{
							UpdateMode: &off,
						},
						TargetRef: &autoscalingv2.CrossVersionObjectReference{
							Kind:       recommendation.Spec.TargetRef.Kind,
							APIVersion: recommendation.Spec.TargetRef.APIVersion,
							Name:       recommendation.Spec.TargetRef.Name,
						},
					},
				}

				err = c.Client.Create(ctx, evpa)
				if err == nil {
					c.Recorder.Event(evpa, v1.EventTypeNormal, "UpdateValue", "Created EffectiveVerticalPodAutoscaler.")
					klog.Infof("Create EffectiveVerticalPodAutoscaler successfully, recommendation %s", klog.KObj(recommendation))
				}
				return err
			}
			// no need to update evpa now
		}
	}

	return nil
}
