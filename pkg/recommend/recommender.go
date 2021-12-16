package recommend

import (
	"context"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/utils"
)

func NewRecommender(kubeClient client.Client, restMapper meta.RESTMapper, scaleClient scale.ScalesGetter,
	recommendation *analysisapi.Recommendation, predictors map[predictionapi.AlgorithmType]prediction.Interface) (*Recommender, error) {
	c, err := GetContext(kubeClient, restMapper, scaleClient, recommendation, predictors)
	if err != nil {
		return nil, err
	}

	return &Recommender{
		Context:    c,
		Inspectors: NewInspectors(c),
		Advisors:   NewAdvisors(c),
	}, nil
}

func (r *Recommender) Offer() (proposed *ProposedRecommendation, err error) {
	// Run inspectors to validate target is ready to recommend
	for _, inspector := range r.Inspectors {
		err := inspector.Inspect()
		if err != nil {
			proposed.Conditions = append(proposed.Conditions, toInspectorCondition(err))
		}
	}

	// If true means some inspectors return error
	if len(proposed.Conditions) != 0 {
		return nil, err
	}

	// Run advisors to propose recommends
	for _, advisor := range r.Advisors {
		err = advisor.Advise(proposed)
		if err != nil {
			return nil, err
		}
	}

	return
}

func toInspectorCondition(err error) metav1.Condition {
	return metav1.Condition{
		Type:               "Inspection",
		Status:             metav1.ConditionFalse,
		Reason:             "FailedInspection",
		Message:            err.Error(),
		LastTransitionTime: metav1.Now(),
	}
}

func GetContext(kubeClient client.Client, restMapper meta.RESTMapper, scaleClient scale.ScalesGetter,
	recommendation *analysisapi.Recommendation, predictors map[predictionapi.AlgorithmType]prediction.Interface) (*Context, error) {
	c := &Context{}
	ref := autoscalingv2.CrossVersionObjectReference{Kind: recommendation.Kind, Name: recommendation.Name, APIVersion: recommendation.APIVersion}
	scale, mapping, err := utils.GetScale(context.TODO(), restMapper, scaleClient, recommendation.Namespace, ref)
	if err != nil {
		return nil, err
	}
	c.Scale = scale
	c.RestMapping = mapping

	unstructured := &unstructured.Unstructured{}
	if err = kubeClient.Get(context.TODO(), client.ObjectKey{Namespace: recommendation.Namespace, Name: recommendation.Spec.TargetRef.Name}, unstructured); err != nil {
		return nil, err
	}

	if recommendation.Spec.TargetRef.Kind == "Deployment" && mapping.GroupVersionKind.Group == "apps" {
		var deployment appsv1.Deployment
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &deployment); err != nil {
			return nil, err
		}
		c.Deployment = &deployment
	}

	if recommendation.Spec.TargetRef.Kind == "StatefulSet" && mapping.GroupVersionKind.Group == "apps" {
		var statefulSet appsv1.StatefulSet
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &statefulSet); err != nil {
			return nil, err
		}
		c.StatefulSet = &statefulSet
	}

	pods, err := utils.GetPodsFromScale(kubeClient, scale)
	if err != nil {
		return nil, err
	}
	c.Pods = pods

	c.Predictors = predictors

	return c, nil
}
