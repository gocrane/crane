package recommend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"k8s.io/klog/v2"

	analysisv1alpha1 "github.com/gocrane/api/analysis/v1alpha1"
	analysis "github.com/gocrane/api/pkg/generated/clientset/versioned/scheme"
)

func LoadConfigSetFromFile(filePath string) (*analysisv1alpha1.ConfigSet, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path not specified")
	}
	configSetBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file path %q: %+v", filePath, err)
	}

	ret, err := loadConfigSetFromBytes(configSetBytes)
	if err != nil {
		return nil, fmt.Errorf("%v: from file %v", err.Error(), filePath)
	}

	return ret, nil
}

func loadConfigSetFromBytes(configSetBytes []byte) (*analysisv1alpha1.ConfigSet, error) {
	configSet := &analysisv1alpha1.ConfigSet{}

	decoder := analysis.Codecs.UniversalDecoder(analysisv1alpha1.SchemeGroupVersion)

	_, gvk, err := decoder.Decode(configSetBytes, nil, configSet)
	if err != nil {
		return nil, fmt.Errorf("failed decoding: %v", err)
	}
	klog.Info("Loaded gvk :%s", gvk)

	klog.V(4).Info("Load recommendation config set successfully.")
	return configSet, nil
}

func GetProperties(configSet *analysisv1alpha1.ConfigSet, dst analysisv1alpha1.Target) map[string]string {
	var selectedProps map[string]string
	maxMatchLevel := -1
	for i, config := range configSet.Configs {
		if len(config.Targets) == 0 && maxMatchLevel < 0 {
			maxMatchLevel = 0
			selectedProps = configSet.Configs[i].Properties
		} else {
			for _, src := range config.Targets {
				matchLevel := targetsMatchLevel(src, dst)
				if matchLevel > maxMatchLevel {
					maxMatchLevel = matchLevel
					selectedProps = configSet.Configs[i].Properties
				}
			}
		}
	}
	bytes, _ := json.Marshal(dst)
	klog.Infof("Got properties %v for target %s", selectedProps, string(bytes))
	return selectedProps
}

func targetsMatchLevel(src, dst analysisv1alpha1.Target) int {
	level := 0
	if src.Namespace != "" && src.Namespace != dst.Namespace {
		return -1
	}
	if src.Kind != "" && src.Kind != dst.Kind {
		return -1
	}
	if src.Name != "" && src.Name != dst.Name {
		return -1
	}
	if src.Namespace != "" && src.Namespace == dst.Namespace {
		level++
	}
	if src.Kind != "" && src.Kind == dst.Kind {
		level++
	}
	if src.Name != "" && src.Name == dst.Name {
		level++
	}
	return level
}
