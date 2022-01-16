package types

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/providers"
)

// Context includes all resource used in recommendation progress
type Context struct {
	ConfigProperties map[string]string
	Predictors       map[predictionapi.AlgorithmType]prediction.Interface
	DataSource       providers.Interface
	Recommendation   *analysisapi.Recommendation
	Scale            *autoscalingapiv1.Scale
	RestMapping      *meta.RESTMapping
	Deployment       *appsv1.Deployment
	StatefulSet      *appsv1.StatefulSet
	Pods             []corev1.Pod
	ReadyPodNumber   int
}

// ProposedRecommendation is the result for one recommendation
type ProposedRecommendation struct {
	// EffectiveHPA is the proposed recommendation for type HPA
	EffectiveHPA *analysisapi.EffectiveHorizontalPodAutoscalerRecommendation

	// ResourceRequest is the proposed recommendation for type Resource
	ResourceRequest *analysisapi.ResourceRequestRecommendation
}
