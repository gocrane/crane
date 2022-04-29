package metricrouter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

type MetricRouterConfig struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	ApiServices []apiregistration.APIServiceSpec `json:"apiServices,omitempty"`
}

