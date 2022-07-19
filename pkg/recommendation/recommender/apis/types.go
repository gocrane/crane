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
	// Override the accepted resources from recommender's interface
	AcceptedResourceSelectors []analysisapi.ResourceSelector `json:"acceptedResources"`
	// Name should be existed in all predefined recommenders
	Name string `json:"name"`
	// Override Recommender configs
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type RecommenderPlugin struct {
	// Name is the name for this plugin
	Name string `json:"name"`
	// Priority control the sequence when execute plugins
	Priority int32 `json:"priority,omitempty"`
	// ServerConfig
	ServerConfig ServerConfig `json:"serverConfig,omitempty"`
	// Override Recommender configs
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type ServerConfig struct {
	UrlPrefix string `json:"urlPrefix,omitempty"`
}
