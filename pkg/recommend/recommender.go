package recommend

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/utils"
)

func NewRecommender(kubeClient client.Client, restMapper meta.RESTMapper, scaleClient scale.ScalesGetter,
	recommendation *analysisapi.Recommendation, predictors map[predictionapi.AlgorithmType]prediction.Interface, logger logr.Logger) (*Recommender, error) {
	c, err := GetContext(kubeClient, restMapper, scaleClient, recommendation, predictors, logger)
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
	proposed = &ProposedRecommendation{}
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

	return proposed, nil
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
	recommendation *analysisapi.Recommendation, predictors map[predictionapi.AlgorithmType]prediction.Interface, logger logr.Logger) (*Context, error) {
	c := &Context{Logger: logger}

	targetRef := autoscalingv2.CrossVersionObjectReference{
		APIVersion: recommendation.Spec.TargetRef.APIVersion,
		Kind:       recommendation.Spec.TargetRef.Kind,
		Name:       recommendation.Spec.TargetRef.Name,
	}

	var scale *autoscalingv1.Scale
	var mapping *meta.RESTMapping
	var err error
	if recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		scale, mapping, err = utils.GetScale(context.TODO(), restMapper, scaleClient, recommendation.Spec.TargetRef.Namespace, targetRef)
		if err != nil {
			return nil, err
		}
		c.Scale = scale
		c.RestMapping = mapping
	}

	unstructured := &unstructured.Unstructured{}
	unstructured.SetKind(recommendation.Spec.TargetRef.Kind)
	unstructured.SetAPIVersion(recommendation.Spec.TargetRef.APIVersion)

	if err := kubeClient.Get(context.TODO(), client.ObjectKey{Namespace: recommendation.Spec.TargetRef.Namespace, Name: recommendation.Spec.TargetRef.Name}, unstructured); err != nil {
		return nil, err
	}

	if recommendation.Spec.TargetRef.Kind == "Deployment" && mapping.GroupVersionKind.Group == "apps" {
		var deployment appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &deployment); err != nil {
			return nil, err
		}
		c.Deployment = &deployment
	}

	if recommendation.Spec.TargetRef.Kind == "StatefulSet" && mapping.GroupVersionKind.Group == "apps" {
		var statefulSet appsv1.StatefulSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &statefulSet); err != nil {
			return nil, err
		}
		c.StatefulSet = &statefulSet
	}

	var pods []corev1.Pod
	if recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		pods, err = utils.GetPodsFromScale(kubeClient, scale)
	} else {
		pods, err = getDaemonSetPods(kubeClient, recommendation.Spec.TargetRef.Namespace, recommendation.Spec.TargetRef.Name)
	}
	if err != nil {
		return nil, err
	}

	c.Pods = pods
	c.Predictors = predictors
	c.Recommendation = *recommendation

	return c, nil
}

func getDaemonSetPods(kubeClient client.Client, namespace string, name string) ([]corev1.Pod, error) {
	ds := appsv1.DaemonSet{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &ds)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(ds.Spec.Selector.MatchLabels),
	}

	podList := &corev1.PodList{}
	err = kubeClient.List(context.TODO(), podList, opts...)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}
