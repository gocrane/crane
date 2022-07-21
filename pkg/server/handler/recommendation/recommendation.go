package recommendation

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
)

type Handler struct {
	client client.Client
}

func NewRecommendationHandler(config *config.Config) *Handler {
	return &Handler{
		client: config.Client,
	}
}

// ListRecommendations list the recommendations in cluster.
func (h *Handler) ListRecommendations(c *gin.Context) {
	recommendList := &analysisapi.RecommendationList{}
	err := h.client.List(context.TODO(), recommendList)
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
