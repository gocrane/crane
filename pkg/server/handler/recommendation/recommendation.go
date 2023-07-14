package recommendation

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gocrane/crane/pkg/server/service/recommendation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	patchtypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/utils"
)

type Handler struct {
	recommendSvc    recommendation.Service
	client          client.Client
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
}

func NewRecommendationHandler(config *config.Config) *Handler {
	dynamicClient := dynamic.NewForConfigOrDie(config.KubeConfig)
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(config.KubeConfig)
	svc := recommendation.NewService(config)
	return &Handler{
		recommendSvc:    svc,
		client:          config.Client,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
	}
}

// ListRecommendations list the recommendations in cluster.
func (h *Handler) ListRecommendations(c *gin.Context) {
	recommendList, err := h.recommendSvc.ListRecommendations(context.Background(), c.Query("filter_options"), c.Query("include_unlimited"))
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, recommendList)
}

// ListRecommendationRules list the recommendationRules in cluster.
func (h *Handler) ListRecommendationRules(c *gin.Context) {
	recommendationRuleList := &analysisapi.RecommendationRuleList{}
	err := h.client.List(context.TODO(), recommendationRuleList)
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	ginwrapper.WriteResponse(c, nil, recommendationRuleList)
}

// CreateRecommendationRule create a recommendationRules from request.
func (h *Handler) CreateRecommendationRule(c *gin.Context) {
	recommendationRule := &analysisapi.RecommendationRule{}
	if err := c.ShouldBindJSON(&recommendationRule); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	err := h.client.Create(context.TODO(), recommendationRule)
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	ginwrapper.WriteResponse(c, nil, nil)
}

// UpdateRecommendationRule update a recommendationRules from request.
func (h *Handler) UpdateRecommendationRule(c *gin.Context) {
	recommendationRule := &analysisapi.RecommendationRule{}
	if err := c.ShouldBindJSON(&recommendationRule); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	if c.Param("recommendationRuleName") != recommendationRule.Name {
		ginwrapper.WriteResponse(c, fmt.Errorf("RecommendationRule name is unexpect"), nil)
		return
	}

	recommendationRuleExist := &analysisapi.RecommendationRule{}
	if err := h.client.Get(context.TODO(), types.NamespacedName{Name: recommendationRule.Name}, recommendationRuleExist); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	recommendationRuleExist.Spec = recommendationRule.Spec
	err := h.client.Update(context.TODO(), recommendationRuleExist)
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	ginwrapper.WriteResponse(c, nil, nil)
}

// AdoptRecommendation adopt a recommendation from request.
func (h *Handler) AdoptRecommendation(c *gin.Context) {
	recommendationExist := &analysisapi.Recommendation{}
	if err := h.client.Get(context.TODO(), types.NamespacedName{Namespace: c.Param("namespace"), Name: c.Param("recommendationName")}, recommendationExist); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	if string(recommendationExist.Spec.Type) == recommender.ReplicasRecommender ||
		string(recommendationExist.Spec.Type) == recommender.ResourceRecommender {
		gvr, err := utils.GetGroupVersionResource(h.discoveryClient, recommendationExist.Spec.TargetRef.APIVersion, recommendationExist.Spec.TargetRef.Kind)
		if err != nil {
			ginwrapper.WriteResponse(c, err, nil)
			return
		}

		_, err = h.dynamicClient.Resource(*gvr).Namespace(recommendationExist.Spec.TargetRef.Namespace).Patch(context.TODO(), recommendationExist.Spec.TargetRef.Name, patchtypes.StrategicMergePatchType, []byte(recommendationExist.Status.RecommendedInfo), metav1.PatchOptions{})
		if err != nil {
			ginwrapper.WriteResponse(c, err, nil)
			return
		}

		ginwrapper.WriteResponse(c, nil, nil)
	} else {
		ginwrapper.WriteResponse(c, fmt.Errorf("Recommendation type %s is not supported for adoption ", string(recommendationExist.Spec.Type)), nil)
		return
	}
}

// DeleteRecommendationRule delete a recommendationRules from request.
func (h *Handler) DeleteRecommendationRule(c *gin.Context) {
	recommendationRuleExist := &analysisapi.RecommendationRule{}
	if err := h.client.Get(context.TODO(), types.NamespacedName{Name: c.Param("recommendationRuleName")}, recommendationRuleExist); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	err := h.client.Delete(context.TODO(), recommendationRuleExist)
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	ginwrapper.WriteResponse(c, nil, nil)
}
