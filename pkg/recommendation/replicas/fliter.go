package replicas

import (
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/controller/analytics"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/recommendation/framework"
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
	target := &corev1.ObjectReference{}
	labelSelector := labels.SelectorFromSet(ctx.Identity.Labels)
	caller := fmt.Sprintf(rr.Name(), klog.KObj(&ctx.RecommendationRule), ctx.RecommendationRule.UID)
	metricNamer := metricnaming.ResourceToWorkloadMetricNamer(target, &resourceCpu, labelSelector, caller)
	if err := metricNamer.Validate(); err != nil {
		return err
	}
	ctx.MetricNamer = metricNamer
	return nil
}

// IsIdentitySupported check weather object identity fit resource selector.
func IsIdentitySupported(identity analytics.ObjectIdentity, selectors []v1alpha1.ResourceSelector) bool {

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
