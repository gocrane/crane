package timeseriesprediction

import (
	"encoding/json"
	"sync"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
)

// TSPTemplate 定义TSP的模板结构
type TSPTemplate struct {
	PredictionWindowSeconds int32                            `json:"predictionWindowSeconds"`
	PredictionMetrics       []predictionapi.PredictionMetric `json:"predictionMetrics"`
}

// TSPTemplateStore 存储TSP模板的线程安全存储
type TSPTemplateStore struct {
	sync.RWMutex
	templates map[string]TSPTemplate // key为recommender name
}

// NewTSPTemplateStore 创建一个新的模板存储
func NewTSPTemplateStore() *TSPTemplateStore {
	return &TSPTemplateStore{
		templates: make(map[string]TSPTemplate),
	}
}

// UpdateFromConfigMap 从ConfigMap更新模板
func (s *TSPTemplateStore) UpdateFromConfigMap(configMap *corev1.ConfigMap) {
	s.Lock()
	defer s.Unlock()

	// 清空当前存储
	s.templates = make(map[string]TSPTemplate)

	// 遍历ConfigMap的每个key，解析为模板
	for recommenderName, templateYAML := range configMap.Data {
		var template TSPTemplate
		// 将YAML转换为JSON再解析，以支持YAML格式
		jsonData, err := yaml.ToJSON([]byte(templateYAML))
		if err != nil {
			klog.Errorf("Failed to convert template YAML to JSON for recommender %s: %v", recommenderName, err)
			continue
		}
		if err := json.Unmarshal(jsonData, &template); err != nil {
			klog.Errorf("Failed to unmarshal template for recommender %s: %v", recommenderName, err)
			continue
		}
		// 进行基本的校验
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

// GetTemplate 获取指定recommender的模板
func (s *TSPTemplateStore) GetTemplate(recommenderName string) (TSPTemplate, bool) {
	s.RLock()
	defer s.RUnlock()
	template, exists := s.templates[recommenderName]
	if !exists {
		return TSPTemplate{}, false
	}

	// 返回副本
	templateCopy := TSPTemplate{
		PredictionWindowSeconds: template.PredictionWindowSeconds,
		PredictionMetrics:       make([]predictionapi.PredictionMetric, len(template.PredictionMetrics)),
	}
	for i := range template.PredictionMetrics {
		templateCopy.PredictionMetrics[i] = *template.PredictionMetrics[i].DeepCopy()
	}
	return templateCopy, exists
}
