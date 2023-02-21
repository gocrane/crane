package config

import (
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// QOSConfig represents the configuration for QOS.
type QOSConfig struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta

	QOSInitializer QOSInitializer `json:"qosInitializer,omitempty"`
}

type QOSInitializer struct {
	// Enable the QoS Initializer
	Enable bool `json:"enable,omitempty"`

	// LabelSelector is a label query over pods that should match the PodQOS
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// A scope selector represents the AND of the selectors represented
	// by the scoped-resource selector requirements.
	ScopeSelector *ScopeSelector `json:"scopeSelector,omitempty"`

	// Container Template for injection
	InitContainerTemplate *corev1.Container `json:"initContainerTemplate,omitempty"`

	// Volume Template for injection
	VolumeTemplate *corev1.Volume `json:"volumeTemplate,omitempty"`
}

type ScopeSelector struct {
	// A list of scope selector requirements by scope of the resources.
	// +optional
	MatchExpressions []ScopedResourceSelectorRequirement `json:"matchExpressions,omitempty"`
}

type ScopeName string

// A scoped-resource selector requirement is a selector that contains values, a scope name, and an operator
// that relates the scope name and values.
type ScopedResourceSelectorRequirement struct {
	// The name of the scope that the selector applies to.
	ScopeName ScopeName `json:"scopeName"`
	// Represents a scope's relationship to a set of values.
	// Valid operators are In, NotIn.
	Operator corev1.ScopeSelectorOperator `json:"operator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty.
	// This array is replaced during a strategic merge patch.
	// +optional
	Values []string `json:"values,omitempty"`
}

func LoadQOSConfigFromFile(filePath string) (*QOSConfig, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path not specified")
	}
	configSetBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file path %q: %+v", filePath, err)
	}

	ret, err := loadConfigFromBytes(configSetBytes)
	if err != nil {
		return nil, fmt.Errorf("%v: from file %v", err.Error(), filePath)
	}

	return ret, nil
}

func loadConfigFromBytes(buf []byte) (*QOSConfig, error) {
	config := &QOSConfig{}
	err := yaml.Unmarshal(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal the byte array: %v", err)
	}

	klog.V(4).Info("Load QOS framework configuration set successfully.")
	return config, nil
}
