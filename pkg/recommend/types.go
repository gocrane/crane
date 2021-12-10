package recommend

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type RecommendationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec RecommendationPolicySpec `json:"spec"`
}

type RecommendationPolicySpec struct {
	InspectorPolicy InspectorPolicy
}

type InspectorPolicy struct {
	PodAvailableRatio      float64
	PodMinReadySeconds     int32
	DeploymentMinReplicas  int32
	StatefulSetMinReplicas int32
	WorkloadMinReplicas    int32
}
