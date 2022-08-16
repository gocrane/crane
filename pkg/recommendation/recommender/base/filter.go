package base

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Filter out k8s resources that are not supported by the recommender.
func (br *BaseRecommender) Filter(ctx *framework.RecommendationContext) error {
	// 1. get object identity
	identity := ctx.Identity

	// 2. load recommender accepted kubernetes object
	accepted := br.Recommender.AcceptedResourceSelectors

	// 3. if not support, abort the recommendation flow
	supported := IsIdentitySupported(identity, accepted)
	if !supported {
		return fmt.Errorf("recommender %s is failed at fliter, your kubernetes resource is not supported for recommender %s.", br.Name(), br.Name())
	}

	return nil
}

// IsIdentitySupported check weather object identity fit resource selector.
func IsIdentitySupported(identity framework.ObjectIdentity, selectors []analysisapi.ResourceSelector) bool {
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
