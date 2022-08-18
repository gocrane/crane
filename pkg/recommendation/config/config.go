package config

import (
	"fmt"
	"io/ioutil"

	klog "k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
)

func LoadRecommenderConfigFromFile(filePath string) (*apis.RecommenderConfiguration, error) {
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

func loadConfigFromBytes(buf []byte) (*apis.RecommenderConfiguration, error) {
	config := &apis.RecommenderConfiguration{}
	err := yaml.Unmarshal(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshal the byte array: %v", err)
	}

	klog.V(4).Info("Load recommendation framework configuration set successfully.")
	return config, nil
}

func GetRecommendersFromConfiguration(file string) ([]apis.Recommender, error) {
	config, err := LoadRecommenderConfigFromFile(file)
	if err != nil {
		klog.Errorf("load recommender configuration failed, %v", err)
		return nil, err
	}

	return config.Recommenders, nil
}

func GetKeysOfMap(m map[string]string) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

func SlicesContainSlice(src []string, target []string) bool {
	contain := true
	for _, value := range target {
		if !contains(src, value) {
			contain = false
		}
	}
	return contain
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
