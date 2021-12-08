package recommend

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
)

// Context includes all resource used in recommendation progress
type Context struct {
	Recommendation analysisapi.Recommendation
	Scale          *autoscalingapiv1.Scale
	RestMapping    *meta.RESTMapping
	Deployment     *appsv1.Deployment
	StatefulSet    *appsv1.StatefulSet
	Pods           []v1.Pod
}

// Recommender take charge of the executor for recommendation
type Recommender struct {
	// Context contains all contexts durning the recommendation
	Context *Context

	// Inspectors is an array of Inspector that needed for this recommendation
	Inspectors []Inspector

	// Advisors is an array of Advisor that needed for this recommendation
	Advisors []Advisor
}

// ProposedRecommendation is the result for one recommendation
type ProposedRecommendation struct {
	// EffectiveHPA is the proposed recommendation for type HPA
	EffectiveHPA *analysisapi.EffectiveHorizontalPodAutoscalerRecommendation

	// ResourceRequest is the proposed recommendation for type Resource
	ResourceRequest *analysisapi.ResourceRequestRecommendation

	// Conditions is an array of current recommendation conditions.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type Inspector interface {
	// Inspect valid for Context to ensure the target is available for recommendation
	Inspect() error
}

type Advisor interface {
	// Advise analysis and give advice in ProposedRecommendation
	Advise(proposed *ProposedRecommendation) error
}
