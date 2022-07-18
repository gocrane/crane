package apis

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
)

type RecommenderConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Recommender list
	Recommenders []Recommender `json:"recommenders"`

	// Recommender Plugin list
	RecommenderPlugins []RecommenderPlugin `json:"recommenderPlugins"`
}

type Recommender struct {
	// ResourceSelector indicates which resources(e.g. a set of Deployments) are accepted for plugin.
	// Override the accepted resources from plugin's interface
	AcceptedResourceSelectors []analysisapi.ResourceSelector `json:"acceptedResources"`
	// RecommenderName is the name for this recommendation that should be included in all recommender collections
	RecommenderName string `json:"recommenderName"`
	// Override Recommendation configs
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type RecommenderPlugin struct {
	// RecommenderName is the name for this recommendation that should be included in all recommender collections
	RecommenderName string `json:"recommenderName"`
	// Priority control the sequence when execute plugins
	Priority int32 `json:"priority,omitempty"`
	// ServerConfig
	ServerConfig ServerConfig `json:"serverConfig,omitempty"`
	// Override Recommendation configs
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type ServerConfig struct {
	UrlPrefix string `json:"urlPrefix,omitempty"`
}
