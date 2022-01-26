package types

import (
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
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
	EffectiveHPA *EffectiveHorizontalPodAutoscalerRecommendation

	// ResourceRequest is the proposed recommendation for type Resource
	ResourceRequest *ResourceRequestRecommendation
}

type EffectiveHorizontalPodAutoscalerRecommendation struct {
	MinReplicas *int32                     `yaml:"minReplicas,omitempty"`
	MaxReplicas *int32                     `yaml:"maxReplicas,omitempty"`
	Metrics     []autoscalingv2.MetricSpec `yaml:"metrics,omitempty"`
	Prediction  *autoscalingapi.Prediction `yaml:"prediction,omitempty"`
}

type ResourceRequestRecommendation struct {
	Containers []ContainerRecommendation `yaml:"containers,omitempty"`
}

type ContainerRecommendation struct {
	ContainerName string       `yaml:"containerName,omitempty"`
	Target        ResourceList `yaml:"target,omitempty"`
}

type ResourceList map[corev1.ResourceName]string
