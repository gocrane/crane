package replicas

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/controller/analytics"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

// Filter out k8s resources that are not supported by the recommender.
func (rr *ReplicasRecommender) Filter(ctx *framework.RecommendationContext) error {
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
	obj := &corev1.ObjectReference{}
	labelSelector := labels.SelectorFromSet(ctx.Identity.Labels)
	caller := fmt.Sprintf(rr.Name(), klog.KObj(&ctx.RecommendationRule), ctx.RecommendationRule.UID)
	metricNamer := metricnaming.ResourceToWorkloadMetricNamer(obj, &resourceCpu, labelSelector, caller)
	if err := metricNamer.Validate(); err != nil {
		return err
	}
	ctx.MetricNamer = metricNamer

	// TODO(xieydd) 5,6,7 will be extracted to public methods
	// 5. get pod template
	template, err := utils.GetPodTemplate(context.TODO(),
		identity.Namespace,
		identity.Name,
		identity.Kind,
		identity.APIVersion,
		ctx.Client)
	if err != nil {
		return err
	}
	ctx.PodTemplate = template

	configSet := rr.Recommender.Config

	// 6. get scale object
	targetRef := autoscalingv2.CrossVersionObjectReference{
		APIVersion: identity.APIVersion,
		Kind:       identity.Kind,
		Name:       identity.Name,
	}

	var scale *autoscalingv1.Scale
	//var mapping *meta.RESTMapping
	if identity.Kind != "DaemonSet" {
		scale, _, err = utils.GetScale(context.TODO(), ctx.RestMapper, ctx.ScaleClient, identity.Namespace, targetRef)
		if err != nil {
			return err
		}
		//ctx.RestMapping = mapping
	}

	workloadMinReplicas, err := strconv.ParseInt(configSet["replicas.workload-min-replicas"], 10, 32)
	if err != nil {
		return err
	}

	if scale != nil && scale.Spec.Replicas < int32(workloadMinReplicas) {
		return fmt.Errorf("workload replicas %d should be larger than %d ", scale.Spec.Replicas, int32(workloadMinReplicas))
	}

	for _, container := range template.Spec.Containers {
		if container.Resources.Requests.Cpu() == nil {
			return fmt.Errorf("container %s resource cpu request is empty ", container.Name)
		}

		if container.Resources.Limits.Cpu() == nil {
			return fmt.Errorf("container %s resource cpu limit is empty ", container.Name)
		}
	}

	// 7. get pods and fliter
	unstructured := &unstructured.Unstructured{}
	unstructured.SetKind(identity.Kind)
	unstructured.SetAPIVersion(identity.APIVersion)

	if err := ctx.Client.Get(context.TODO(), client.ObjectKey{Namespace: identity.Namespace, Name: identity.Name}, unstructured); err != nil {
		return err
	}

	var pods []corev1.Pod
	if identity.Kind != "DaemonSet" {
		pods, err = utils.GetPodsFromScale(ctx.Client, scale)
	} else {
		var daemonSet appsv1.DaemonSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &daemonSet); err != nil {
			return err
		}
		ctx.DaemonSet = &daemonSet
		pods, err = utils.GetDaemonSetPods(ctx.Client, identity.Namespace, identity.Name)
	}
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("existing pods should be larger than 0 ")
	}

	podMinReadySeconds, err := strconv.ParseInt(configSet["replicas.pod-min-ready-seconds"], 10, 32)
	if err != nil {
		return err
	}

	podAvailableRatio, err := strconv.ParseFloat(configSet["replicas.pod-available-ratio"], 64)
	if err != nil {
		return err
	}

	readyPods := 0
	for _, pod := range pods {
		if utils.IsPodAvailable(&pod, int32(podMinReadySeconds), metav1.Now()) {
			readyPods++
		}
	}

	if readyPods == 0 {
		return fmt.Errorf("pod available number must larger than zero. ")
	}

	availableRatio := float64(readyPods) / float64(len(pods))
	if availableRatio < podAvailableRatio {
		return fmt.Errorf("pod available ratio is %.3f less than %.3f ", availableRatio, podAvailableRatio)
	}

	return nil
}

// IsIdentitySupported check weather object identity fit resource selector.
func IsIdentitySupported(identity analytics.ObjectIdentity, selectors []analysisapi.ResourceSelector) bool {

	supported := false
	for _, selector := range selectors {
		newSelector := analysisapi.ResourceSelector{
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
