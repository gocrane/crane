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
	recommendtypes "github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

func (c *Controller) UpdateRecommendation(ctx context.Context, recommendation *analysisapi.Recommendation) (bool, error) {
	var proposedEHPA recommendtypes.EffectiveHorizontalPodAutoscalerRecommendation
	var proposedResource recommendtypes.ProposedRecommendation
	needUpdate := false

	if recommendation.Spec.Type == analysisapi.AnalysisTypeResource {
		err := yaml.Unmarshal([]byte(recommendation.Status.RecommendedValue), &proposedResource)
		if err != nil {
			return false, err
		}
	}

	if recommendation.Spec.Type == analysisapi.AnalysisTypeHPA {
		err := yaml.Unmarshal([]byte(recommendation.Status.RecommendedValue), &proposedEHPA)
		if err != nil {
			return false, err
		}
	}

	if recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeStatus {
		return false, nil
	}

	unstructed := &unstructured.Unstructured{}
	unstructed.SetAPIVersion(recommendation.Spec.TargetRef.APIVersion)
	unstructed.SetKind(recommendation.Spec.TargetRef.Kind)
	err := c.Client.Get(ctx, client.ObjectKey{Name: recommendation.Spec.TargetRef.Name, Namespace: recommendation.Spec.TargetRef.Namespace}, unstructed)
	if err != nil {
		return false, fmt.Errorf("get target object failed: %v. ", err)
	}

	if recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeStatusAndAnnotation || recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeAuto {
		annotation := unstructed.GetAnnotations()
		if annotation == nil {
			annotation = map[string]string{}
		}

		switch recommendation.Spec.Type {
		case analysisapi.AnalysisTypeResource:
			if annotation[known.ResourceRecommendationValueAnnotation] != recommendation.Status.RecommendedValue {
				annotation[known.ResourceRecommendationValueAnnotation] = recommendation.Status.RecommendedValue
				needUpdate = true
			}
		case analysisapi.AnalysisTypeHPA:
			if annotation[known.HPARecommendationValueAnnotation] != recommendation.Status.RecommendedValue {
				annotation[known.HPARecommendationValueAnnotation] = recommendation.Status.RecommendedValue
				needUpdate = true
			}
		}

		if needUpdate {
			unstructed.SetAnnotations(annotation)
			err = c.Client.Update(ctx, unstructed)
			if err != nil {
				return false, fmt.Errorf("update target annotation failed: %v. ", err)
			}
		}
	}

	// Only support Auto Type for EHPA recommendation
	if recommendation.Spec.AdoptionType == analysisapi.AdoptionTypeAuto {
		if recommendation.Spec.Type == analysisapi.AnalysisTypeHPA {
			ehpa, err := utils.GetEHPAFromScaleTarget(ctx, c.Client, recommendation.Spec.TargetRef.Namespace, recommendation.Spec.TargetRef)
			if err != nil {
				return false, fmt.Errorf("get EHPA from target failed: %v. ", err)
			}
			if ehpa == nil {
				ehpa = &autoscalingapi.EffectiveHorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: recommendation.Spec.TargetRef.Namespace,
						Name:      recommendation.Spec.TargetRef.Name,
					},
					Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
						MinReplicas:   proposedEHPA.MinReplicas,
						MaxReplicas:   *proposedEHPA.MaxReplicas,
						Metrics:       proposedEHPA.Metrics,
						ScaleStrategy: autoscalingapi.ScaleStrategyPreview,
						Prediction:    proposedEHPA.Prediction,
						ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
							Kind:       recommendation.Spec.TargetRef.Kind,
							APIVersion: recommendation.Spec.TargetRef.APIVersion,
							Name:       recommendation.Spec.TargetRef.Name,
						},
					},
				}

				if err = c.Client.Create(ctx, ehpa); err == nil {
					return false, err
				}
				c.Recorder.Event(ehpa, v1.EventTypeNormal, "UpdateValue", "Created EffectiveHorizontalPodAutoscaler.")
				klog.Infof("Create EffectiveHorizontalPodAutoscaler successfully, recommendation %s", klog.KObj(recommendation))
				needUpdate = true
			} else {
				// we don't override ScaleStrategy, because we always use preview to be the default version,
				// if user change it, we don't want to override it.
				// The reason for Prediction is the same.
				ehpaUpdate := ehpa.DeepCopy()
				ehpaUpdate.Spec.MinReplicas = proposedEHPA.MinReplicas
				ehpaUpdate.Spec.MaxReplicas = *proposedEHPA.MaxReplicas
				ehpaUpdate.Spec.Metrics = proposedEHPA.Metrics

				if !equality.Semantic.DeepEqual(&ehpaUpdate.Spec, &ehpa.Spec) {
					if err = c.Client.Update(ctx, ehpaUpdate); err != nil {
						return false, err

					}
					c.Recorder.Event(ehpa, v1.EventTypeNormal, "UpdateValue", "Updated EffectiveHorizontalPodAutoscaler.")
					klog.Infof("Update EffectiveHorizontalPodAutoscaler successfully, recommendation %s", klog.KObj(recommendation))
					needUpdate = true
				}
			}
		}

		if recommendation.Spec.Type == analysisapi.AnalysisTypeResource {
			evpa, err := utils.GetEVPAFromScaleTarget(ctx, c.Client, recommendation.Spec.TargetRef.Namespace, recommendation.Spec.TargetRef)
			if err != nil {
				return false, fmt.Errorf("get EVPA from target failed: %v. ", err)
			}
			if evpa == nil {
				off := vpatypes.UpdateModeOff
				var policies []autoscalingapi.ContainerResourcePolicy
				podTemplate, err := utils.GetPodTemplate(ctx, evpa.Namespace, evpa.Spec.TargetRef.Name, evpa.Spec.TargetRef.Kind, evpa.Spec.TargetRef.APIVersion, c.Client)
				if err != nil {
					return false, err
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

				if err = c.Client.Create(ctx, evpa); err != nil {
					return false, err
				}
				c.Recorder.Event(evpa, v1.EventTypeNormal, "UpdateValue", "Created EffectiveVerticalPodAutoscaler.")
				klog.Infof("Create EffectiveVerticalPodAutoscaler successfully, recommendation %s", klog.KObj(recommendation))
				needUpdate = true
			}
			// no need to update evpa now
		}
	}

	return needUpdate, nil
}
