# Recommendation Definition
- This proposal aims at definition for universal resource optimization. 

## Table of Contents

<!-- TOC -->


<!-- /TOC -->

## Motivation



## Proposal

### Api Definition

RecommendationRule defines which resources are required to recommend and what is the runInterval.

```go
// RecommendationRuleSpec defines resources and runInterval to recommend
type RecommendationRuleSpec struct {
	// ResourceSelector indicates how to select resources(e.g. a set of Deployments) for an Recommendation.
	// +required
	// +kubebuilder:validation:Required
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

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

namespace ?
```

Recommendation is a content holder for recommendation result. We hope that the recommendation data can be applied directly to kubernetes cluster(Recommendation as a code) and Different type recommendation have different recommendation yaml, so the content is stored in recommendation as `Data`.

```go
type Recommendation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Data runtime.RawExtension `json:"data"`
}
```

### Recommendation Configuration

Recommendation Configuration is centralized configuration that contains every rule for universal resource optimization. It not only includes RecommendationRules that use defines but also contains RecommendationPlugins.




