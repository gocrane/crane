# Recommendation Definition
- This proposal aims at definition for universal resource optimization. 

## Table of Contents

<!-- TOC -->


<!-- /TOC -->

## Proposal

### RecommendationRule

RecommendationRule defines which resources are required to recommend and what is the runInterval. We want it simple enough for most users.

```go
// RecommendationRuleSpec defines resources and runInterval to recommend
type RecommendationRuleSpec struct {
	// ResourceSelector indicates how to select resources(e.g. a set of Deployments) for a Recommendation.
	// +required
	// +kubebuilder:validation:Required
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

	// NamespaceSelector indicates resource namespaces to select from
	NamespaceSelector NamespaceSelector `json:"namespaceSelector"`

	// RunInterval between two recommendation
	RunInterval time.Duration `json:"runInterval,omitempty"`
}

// ResourceSelector describes how the resources will be selected.
type ResourceSelector struct {
	// Kind of the resource, e.g. Deployment
	Kind string `json:"kind"`

	// API version of the resource, e.g. "apps/v1"
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// Name of the resource.
	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// NamespaceSelector describes how to select namespaces for recommend
type NamespaceSelector struct {
    // Select all namespace if true
    Any bool `json:"any,omitempty"`
    // List of namespace names to select from.
    MatchNames []string `json:"matchNames,omitempty"`
}
```

- cluster scope CRD
- use ResourceSelector to choose multiple resources to be recommended
- use NamespaceSelector to choose namespace for resources

### Recommender

Recommender Configuration is centralized configuration that is read as a config file for craned. Recommender Configuration indicate which recommenders are enabled and the metadata for recommender.

```go
type RecommenderSpec struct {
    // ResourceSelector indicates which resources(e.g. a set of Deployments) are accepted for plugin.
    // Override the accepted resources from plugin's interface
    AcceptedResourceSelectors []ResourceSelector `json:"AcceptedResources"`

    // RecommenderName is the name for this recommendation that should be included in all recommender collections
    RecommenderName string `json:"pluginName"`

    // Category indicate the category for this recommender
    Category string `json:"pluginName"`

    // Override Recommendation configs
    // +optional
    Config map[string]string `json:"config,omitempty"`
}

```

- AcceptedResourceSelectors make you narrow down the resources for recommender or broaden it
- Category is a predefined list that indicate what type of it, e.g. WorkloadResourceRecommender, ServiceIdleRecommender
- RecommenderName is the name for recommender, every recommender in Recommendation Framework should have a unique name
- Config is a map to let user config properties for their recommender

### RecommenderPlugin

RecommenderPlugin is another section in Recommender Config. Users can develop their own Plugin and make it take effect to existing Recommender.

```go
type RecommenderPluginSpec struct {

    // RecommenderName is the name for this recommendation that should be included in all recommender collections
    RecommenderName string `json:"pluginName"`

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

```

### Recommendation

Recommendation is a content holder for recommendation result. We hope that the recommendation data can be applied directly to kubernetes cluster(Recommendation as a code) and Different type recommendation have different recommendation yaml, so the content is stored in recommendation as `Data`.

```go
type Recommendation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Data runtime.RawExtension `json:"data"`
}
```

Recommendation has some labels to indicate key information:
1. RecommendationRule name
2. RecommendationRule uid
3. target uid
4. target namespace
5. target name
6. target kind

Questions:

1. Is Recommendation namespaced scope? if so, how to handle MachineRecommendations

#### Alternatives

Use a constructive model for Recommendation.

```go

type RecommendationSpec struct {
	
	Resource string 
	
	Value string
	
	RecommendValue string
	
	Action string
	
}

```


#### Risks and Mitigations

1. Backward compatibility: Use v1alpha2 for above CRDs or Use new group? recommend.crane.io

