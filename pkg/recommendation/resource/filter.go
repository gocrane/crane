package resource

import (
	"context"
	"fmt"
	"github.com/gocrane/crane/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Filter out k8s resources that are not supported by the recommender.
func (rr *ResourceRecommender) Filter(ctx *framework.RecommendationContext) error {
	// 1. get object identity
	identity := ctx.Identity

	// 2. load recommender accepted kubernetes object
	accepted := rr.Recommender.AcceptedResourceSelectors

	// 3. if not support, abort the recommendation flow
	supported := IsIdentitySupported(identity, accepted)
	if !supported {
		return fmt.Errorf("recommender %s is failed at fliter, your kubernetes resource is not supported for recommender %s.", rr.Name(), rr.Name())
	}

	// 4. generate metric
	resourceCpu := corev1.ResourceCPU
	target := &corev1.ObjectReference{}
	labelSelector := labels.SelectorFromSet(ctx.Identity.Labels)
	caller := fmt.Sprintf(rr.Name(), klog.KObj(&ctx.RecommendationRule), ctx.RecommendationRule.UID)
	metricNamer := metricnaming.ResourceToWorkloadMetricNamer(target, &resourceCpu, labelSelector, caller)
	if err := metricNamer.Validate(); err != nil {
		return err
	}
	ctx.MetricNamer = metricNamer

	// fill Object
	// TODO: move to util
	key := client.ObjectKey{
		Name:      ctx.Identity.Name,
		Namespace: ctx.Identity.Namespace,
	}

	unstructed := &unstructured.Unstructured{}
	unstructed.SetAPIVersion(ctx.Identity.APIVersion)
	unstructed.SetKind(ctx.Identity.Kind)
	var err error
	if err = ctx.Client.Get(ctx.Context, key, unstructed); err != nil {
		return err
	}

	// fill PodTemplate
	podTemplateObject, found, err := unstructured.NestedMap(unstructed.Object, "spec", "template")
	if !found || err != nil {
		return fmt.Errorf("get template from unstructed object %s failed. ", klog.KObj(unstructed))
	}

	framework.ToK8SObject(podTemplateObject, ctx.PodTemplate)

	// fill Scale
	targetRef := autoscalingv2.CrossVersionObjectReference{
		APIVersion: ctx.Recommendation.Spec.TargetRef.APIVersion,
		Kind:       ctx.Recommendation.Spec.TargetRef.Kind,
		Name:       ctx.Recommendation.Spec.TargetRef.Name,
	}

	var scale *autoscalingv1.Scale
	if ctx.Recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		scale, _, err = utils.GetScale(context.TODO(), ctx.RestMapper, ctx.ScaleClient, ctx.Recommendation.Spec.TargetRef.Namespace, targetRef)
		if err != nil {
			return err
		}
		ctx.Scale = scale
	}

	// fill Pods
	var pods []corev1.Pod
	if ctx.Recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		pods, err = utils.GetPodsFromScale(ctx.Client, scale)
	} else {
		var daemonSet appsv1.DaemonSet
		err = framework.ToK8SObject(ctx.Object, &daemonSet)
		pods, err = framework.GetDaemonSetPods(ctx.Client, ctx.Recommendation.Spec.TargetRef.Namespace, ctx.Recommendation.Spec.TargetRef.Name)
		ctx.Pods = pods
	}
	if err != nil {
		return err
	}

	// filter workloads that are downing
	if len(ctx.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	pod := ctx.Pods[0]
	if len(pod.OwnerReferences) == 0 {
		return fmt.Errorf("owner reference not found")
	}

	return nil
}

// IsIdentitySupported check weather object identity fit resource selector.
func IsIdentitySupported(identity framework.ObjectIdentity, selectors []v1alpha1.ResourceSelector) bool {
	supported := false
	for _, selector := range selectors {
		newSelector := v1alpha1.ResourceSelector{
			Name:       identity.Name,
			APIVersion: identity.APIVersion,
			Kind:       identity.Kind,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: identity.Labels,
			},
		}

		supported = reflect.DeepEqual(newSelector, selector)
	}

	return supported
}
