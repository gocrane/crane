package recommendation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/resource"
	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	patchtypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

type Service interface {
	ListRecommendations(ctx context.Context, opts string, includeUnlimited string) (*analysisapi.RecommendationList, error)
	ListRecommendationRules(ctx context.Context) (*analysisapi.RecommendationRuleList, error)
	CreateRecommendationRule(ctx context.Context, r *analysisapi.RecommendationRule) error
	UpdateRecommendationRule(ctx context.Context, recommendationRuleName string, r *analysisapi.RecommendationRule) error
	AdoptRecommendation(ctx context.Context, namespace string, recommendationName string) error
	DeleteRecommendationRule(ctx context.Context, recommendationRuleName string) error
}

type recommendationService struct {
	client          client.Client
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	scaleClient     scale.ScalesGetter
	PredictorMgr    predictormgr.Manager
	Provider        providers.History
}

func NewService(config *config.Config) *recommendationService {
	dynamicClient := dynamic.NewForConfigOrDie(config.KubeConfig)
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(config.KubeConfig)

	return &recommendationService{
		client:          config.Client,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
	}
}

func (s *recommendationService) ListRecommendations(ctx context.Context, optsStr string, includeUnlimited string) (*analysisapi.RecommendationList, error) {
	recommendList := &analysisapi.RecommendationList{}
	err := s.client.List(context.TODO(), recommendList)
	if err != nil {
		return nil, err
	}
	opts, err := parseFilterOption(optsStr)
	if err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		newItems := make([]analysisapi.Recommendation, 0)
		for _, r := range recommendList.Items {
			allPassed := true
			for _, opt := range opts {
				if !opt.Check(r, includeUnlimited) {
					allPassed = false
				}
			}
			if allPassed {
				newItems = append(newItems, r)
			}
		}
		recommendList.Items = newItems
	}
	return recommendList, nil
}

//func (s *recommendationService) ListLowLoadWorkloads(ctx context.Context) (*analysisapi.RecommendationList, error) {
//	recommendList := &analysisapi.RecommendationList{}
//	err := s.client.List(context.TODO(), recommendList)
//	if err != nil {
//		return nil, err
//	}
//	if len(recommendList.Items) == 0 {
//		return recommendList, nil
//	}
//	newRecommendList := make([]analysisapi.Recommendation, 0)
//	for _, r := range recommendList.Items {
//		framework.NewRecommendationContext()
//		rctx := &framework.RecommendationContext{Recommendation: &r, Client: s.client}
//		framework.RetrievePods(rctx)
//		r.Status.
//	}
//
//}

type filterOption struct {
	// ResourceType valid value: cpu, mem
	ResourceType string
	// RequirementType valid value: limit, request
	RequirementType string
	// Ratio = currentResource / recommendedResource
	Ratio float64
	// CompareType valid values: eq, gt, lt, ge, le
	CompareType string
}

func (fo *filterOption) ParseFromStr(s string) error {
	os := strings.Split(s, ",")
	if len(os) < 4 {
		return errors.New("invalid filter option format, option example: cpu,limit,ge,1.5@mem,request,lt,1.0")
	}
	resouceType, requirementType, compareType, ratioStr := os[0], os[1], os[2], os[3]
	ratio, err := strconv.ParseFloat(ratioStr, 64)
	if err != nil {
		return errors.New("invalid ratio, must a float value")
	}

	if resouceType != "cpu" && resouceType != "mem" {
		return errors.New("invalid resource type, must be 'cpu' or 'mem'")
	}

	if requirementType != "limit" && requirementType != "request" {
		return errors.New("invalid requirement type, must be 'limit' or 'request'")
	}

	if compareType != "eq" && compareType != "ge" && compareType != "gt" && compareType != "le" && compareType != "lt" {
		return errors.New("invalid requirement type, valid value: eq, gt, lt, ge, le")
	}
	fo.Ratio = ratio
	fo.CompareType = compareType
	fo.RequirementType = requirementType
	fo.ResourceType = resouceType
	return nil
}

func (fo *filterOption) Check(r analysisapi.Recommendation, includeUnlimited string) bool {
	if string(r.Spec.Type) != recommender.ResourceRecommender {
		return false
	}
	current := &resource.PatchResource{}
	err := json.Unmarshal([]byte(r.Status.CurrentInfo), current)
	if err != nil {
		return false
	}
	recommended := &resource.PatchResource{}
	err = json.Unmarshal([]byte(r.Status.RecommendedInfo), recommended)
	if err != nil {
		return false
	}
	curResListMap := make(map[string]corev1.ResourceList)
	recResListMap := make(map[string]corev1.ResourceList)
	switch fo.RequirementType {
	case "limit":
		for _, c := range current.Spec.Template.Spec.Containers {
			curResListMap[c.Name] = c.Resources.Limits
		}
		for _, c := range recommended.Spec.Template.Spec.Containers {
			// crane only recommend requests resource
			recResListMap[c.Name] = c.Resources.Requests
		}
	case "request":
		for _, c := range current.Spec.Template.Spec.Containers {
			curResListMap[c.Name] = c.Resources.Requests
		}
		for _, c := range recommended.Spec.Template.Spec.Containers {
			recResListMap[c.Name] = c.Resources.Requests
		}
	}
	curResValMap := make(map[string]int64)
	recResValMap := make(map[string]int64)
	switch fo.ResourceType {
	case "cpu":
		for k, v := range curResListMap {
			q := v[corev1.ResourceCPU]
			curResValMap[k] = q.Value()
		}
		for k, v := range recResListMap {
			q := v[corev1.ResourceCPU]
			recResValMap[k] = q.Value()
		}
	case "mem":
		for k, v := range curResListMap {
			q := v[corev1.ResourceMemory]
			curResValMap[k] = q.Value()
		}
		for k, v := range recResListMap {
			q := v[corev1.ResourceMemory]
			recResValMap[k] = q.Value()
		}
	}
	ratioMap := make(map[string]float64)
	for k, v := range curResValMap {
		if v == 0 {
			if includeUnlimited == "true" {
				return true
			} else {
				return false
			}
		}
		if recResValMap[k] == 0 {
			return false
		}
		ratioMap[k] = float64(v) / float64(recResValMap[k])
	}

	switch fo.CompareType {
	// eq, gt, lt, ge, le
	case "eq":
		for _, rt := range ratioMap {
			if rt == fo.Ratio {
				return true
			}
		}
	case "gt":
		for _, rt := range ratioMap {
			if rt > fo.Ratio {
				return true
			}
		}
	case "lt":
		for _, rt := range ratioMap {
			if rt < fo.Ratio {
				return true
			}
		}
	case "ge":
		for _, rt := range ratioMap {
			if rt >= fo.Ratio {
				return true
			}
		}
	case "le":
		for _, rt := range ratioMap {
			if rt <= fo.Ratio {
				return true
			}
		}
	}
	return false
}

func parseFilterOption(optStr string) ([]*filterOption, error) {
	if optStr == "" {
		return []*filterOption{}, nil
	}
	opts := strings.Split(optStr, "@")
	fos := make([]*filterOption, 0)
	for _, o := range opts {
		fo := &filterOption{}
		err := fo.ParseFromStr(o)
		if err != nil {
			return nil, err
		}
		fos = append(fos, fo)
	}

	return fos, nil
}

func (s *recommendationService) ListRecommendationRules(ctx context.Context) (*analysisapi.RecommendationRuleList, error) {
	recommendationRuleList := &analysisapi.RecommendationRuleList{}
	err := s.client.List(context.TODO(), recommendationRuleList)
	if err != nil {
		return nil, err
	}
	return recommendationRuleList, nil
}

func (s *recommendationService) CreateRecommendationRule(ctx context.Context, r *analysisapi.RecommendationRule) error {
	return s.client.Create(context.TODO(), r)
}

func (s *recommendationService) UpdateRecommendationRule(ctx context.Context, recommendationRuleName string, r *analysisapi.RecommendationRule) error {

	if recommendationRuleName != r.Name {
		return errors.New("RecommendationRule name is unexpect")
	}

	recommendationRuleExist := &analysisapi.RecommendationRule{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{Name: r.Name}, recommendationRuleExist); err != nil {
		return err
	}

	recommendationRuleExist.Spec = r.Spec
	err := s.client.Update(context.TODO(), recommendationRuleExist)
	if err != nil {
		return err
	}
	return nil
}

func (s *recommendationService) AdoptRecommendation(ctx context.Context, namespace string, recommendationName string) error {
	recommendationExist := &analysisapi.Recommendation{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: recommendationName}, recommendationExist); err != nil {
		return err
	}

	if string(recommendationExist.Spec.Type) == recommender.ReplicasRecommender ||
		string(recommendationExist.Spec.Type) == recommender.ResourceRecommender {
		gvr, err := utils.GetGroupVersionResource(s.discoveryClient, recommendationExist.Spec.TargetRef.APIVersion, recommendationExist.Spec.TargetRef.Kind)
		if err != nil {
			return err
		}

		_, err = s.dynamicClient.Resource(*gvr).Namespace(recommendationExist.Spec.TargetRef.Namespace).Patch(context.TODO(), recommendationExist.Spec.TargetRef.Name, patchtypes.StrategicMergePatchType, []byte(recommendationExist.Status.RecommendedInfo), metav1.PatchOptions{})
		if err != nil {
			return err
		}

		return nil
	} else {
		return fmt.Errorf("Recommendation type %s is not supported for adoption ", string(recommendationExist.Spec.Type))
	}
}

func (s *recommendationService) DeleteRecommendationRule(ctx context.Context, recommendationRuleName string) error {
	recommendationRuleExist := &analysisapi.RecommendationRule{}
	if err := s.client.Get(context.TODO(), types.NamespacedName{Name: recommendationRuleName}, recommendationRuleExist); err != nil {
		return err
	}

	err := s.client.Delete(context.TODO(), recommendationRuleExist)
	if err != nil {
		return err
	}

	return nil
}
