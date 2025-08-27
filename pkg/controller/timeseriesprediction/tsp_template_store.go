package timeseriesprediction

import (
	"encoding/json"
	"sync"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
)

// TSPTemplate defines the template structure for TSP
type TSPTemplate struct {
	PredictionWindowSeconds int32                            `json:"predictionWindowSeconds"`
	PredictionMetrics       []predictionapi.PredictionMetric `json:"predictionMetrics"`
}

// TSPTemplateStore is a thread-safe store for TSP templates
type TSPTemplateStore struct {
	sync.RWMutex
	templates map[string]TSPTemplate // key为recommender name
}

// NewTSPTemplateStore creates a new template store
func NewTSPTemplateStore() *TSPTemplateStore {
	return &TSPTemplateStore{
		templates: make(map[string]TSPTemplate),
	}
}

// UpdateFromConfigMap updates templates from ConfigMap
func (s *TSPTemplateStore) UpdateFromConfigMap(configMap *corev1.ConfigMap) {
	s.Lock()
	defer s.Unlock()

	// Clear current store
	s.templates = make(map[string]TSPTemplate)

	// Iterate each key in ConfigMap and parse as template
	for recommenderName, templateYAML := range configMap.Data {
		var template TSPTemplate
		// Convert YAML to JSON for parsing, to support YAML format
		jsonData, err := yaml.ToJSON([]byte(templateYAML))
		if err != nil {
			klog.Errorf("Failed to convert template YAML to JSON for recommender %s: %v", recommenderName, err)
			continue
		}
		if err := json.Unmarshal(jsonData, &template); err != nil {
			klog.Errorf("Failed to unmarshal template for recommender %s: %v", recommenderName, err)
			continue
		}
		// Basic validation
		if template.PredictionWindowSeconds <= 0 {
			klog.Warningf("Invalid PredictionWindowSeconds in template for recommender %s: %d, using default", recommenderName, template.PredictionWindowSeconds)
			continue
		}
		if len(template.PredictionMetrics) == 0 {
			klog.Warningf("Empty PredictionMetrics in template for recommender %s", recommenderName)
			continue
		}
		s.templates[recommenderName] = template
		klog.Infof("Loaded template for recommender: %s", recommenderName)
	}
}

// GetTemplate retrieves the template for the specified recommender
func (s *TSPTemplateStore) GetTemplate(recommenderName string) (TSPTemplate, bool) {
	s.RLock()
	defer s.RUnlock()
	template, exists := s.templates[recommenderName]
	if !exists {
		return TSPTemplate{}, false
	}

	// Return a copy
	templateCopy := TSPTemplate{
		PredictionWindowSeconds: template.PredictionWindowSeconds,
		PredictionMetrics:       make([]predictionapi.PredictionMetric, len(template.PredictionMetrics)),
	}
	for i := range template.PredictionMetrics {
		templateCopy.PredictionMetrics[i] = *template.PredictionMetrics[i].DeepCopy()
	}
	return templateCopy, exists
}
