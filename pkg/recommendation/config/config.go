package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	klog "k8s.io/klog/v2"
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
	err := json.Unmarshal(buf, config)
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

	configRecommenders := config.Recommenders
	recommenders := make([]apis.Recommender, len(configRecommenders))
	for _, r := range configRecommenders {
		recommenders = append(recommenders, r)
	}
	return recommenders, nil
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
