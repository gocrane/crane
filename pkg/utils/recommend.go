package utils

import (
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	analysisv1alpha1 "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
)

func GetGroupVersionResource(discoveryClient discovery.DiscoveryInterface, apiVersion string, kind string) (*schema.GroupVersionResource, error) {
	resList, err := discoveryClient.ServerResourcesForGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	var resName string
	for _, res := range resList.APIResources {
		if kind == res.Kind {
			resName = res.Name
			break
		}
	}
	if resName == "" {
		return nil, fmt.Errorf("invalid kind %s", kind)
	}

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}
	gvr := gv.WithResource(resName)
	return &gvr, nil
}

func IsRecommendationControlledByRule(recommend *analysisv1alpha1.Recommendation) bool {
	for _, ownerReference := range recommend.OwnerReferences {
		if ownerReference.Kind == "RecommendationRule" {
			return true
		}
	}
	return false
}

func SetRunNumber(recommendation *analysisv1alpha1.Recommendation, runNumber int32) {
	if recommendation.Annotations == nil {
		recommendation.Annotations = map[string]string{}
	}
	recommendation.Annotations[known.RunNumberAnnotation] = strconv.Itoa(int(runNumber))
}

func GetRunNumber(recommendation *analysisv1alpha1.Recommendation) (int32, error) {
	val, ok := recommendation.Annotations[known.RunNumberAnnotation]
	if ok && len(val) != 0 {
		runNumberInt, err := strconv.ParseInt(val, 10, 32)
		return int32(runNumberInt), err
	}

	return 0, fmt.Errorf("get runNumber failed")
}

func GetRecommendationRuleOwnerReference(recommend *analysisv1alpha1.Recommendation) *metav1.OwnerReference {
	for _, ownerReference := range recommend.OwnerReferences {
		if ownerReference.Kind == "RecommendationRule" {
			return &ownerReference
		}
	}
	return nil
}
