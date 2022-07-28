package recommendation

import (
	"encoding/json"
	"fmt"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/replicas"
	"io/ioutil"
	"k8s.io/klog"
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

func GetRecommendersFromConfiguration(file string) ([]Recommender, error) {
	config, err := LoadRecommenderConfigFromFile(file)
	if err != nil {
		klog.Errorf("load recommender configuration failed, %v", err)
		return nil, err
	}

	configRecommenders := config.Recommenders
	recommenders := make([]Recommender, len(configRecommenders))
	for _, recommender := range configRecommenders {
		switch recommender.Name {
		case ReplicasRecommender:
			recommenders = append(recommenders, replicas.NewReplicasRecommender(recommender))
		default:
			recommenders = append(recommenders, replicas.NewReplicasRecommender(recommender))
		}
	}
	return recommenders, nil
}
