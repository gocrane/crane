package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Filter out k8s resources that are not supported by the recommender.
func (s *ServiceRecommender) Filter(ctx *framework.RecommendationContext) error {
	var err error

	// filter resource that not match objectIdentity
	if err = s.BaseRecommender.Filter(ctx); err != nil {
		return err
	}
	var svc corev1.Service
	if err = framework.ObjectConversion(ctx.Object, &svc); err != nil {
		return err
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return fmt.Errorf("service: %v type: %s is not a LoadBalancer", ctx.Object.GetName(), svc.Spec.Type)
	}

	// filter Endpoints not empty
	var ep corev1.Endpoints
	if err = ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(ctx.Object), &ep); client.IgnoreNotFound(err) != nil {
		return err
	}
	for _, ss := range ep.Subsets {
		if len(ss.Addresses) != 0 {
			return fmt.Errorf("service: %v addresses: %v not empty", ctx.Object.GetName(), ss.Addresses)
		}
		if len(ss.NotReadyAddresses) != 0 {
			return fmt.Errorf("service: %v NotReadyAddresses: %v not empty", ctx.Object.GetName(), ss.NotReadyAddresses)
		}
	}

	if err = framework.RetrievePods(ctx); err != nil {
		return err
	}

	return nil
}
