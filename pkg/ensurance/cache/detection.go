package cache

import (
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
)

type DetectionCondition struct {
	// the Objective Ensurance name
	ObjectiveEnsuranceName string
	// only dryRun
	DryRun bool
	// if the policy triggered action
	Triggered bool
	// if the policy triggered restored action
	Restored bool
	// action name
	ActionName string
	// node qos ensurance policy
	Nep *ensuranceapi.NodeQOSEnsurancePolicy
	// time for detection
	Time time.Time
	// the influenced pod list
	// node detection the pod list is empty
	BeInfluencedPods []types.NamespacedName
}

type DetectionConditionCache struct {
	mu        sync.Mutex // protects detectMap
	detectMap map[string]DetectionCondition
}

func (s *DetectionConditionCache) GetOrCreate(c DetectionCondition) DetectionCondition {
	s.mu.Lock()
	defer s.mu.Unlock()

	var key = GenerateDetectionKey(c)

	cacheDetection, ok := s.detectMap[key]
	if !ok {
		s.detectMap[key] = c
	}
	return cacheDetection
}

func (s *DetectionConditionCache) Get(key string) (DetectionCondition, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	detect, ok := s.detectMap[key]
	return detect, ok
}

func (s *DetectionConditionCache) Set(c DetectionCondition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var key = GenerateDetectionKey(c)
	s.detectMap[key] = c

	return
}

func (s *DetectionConditionCache) Exist(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.detectMap[key]
	return ok
}

func (s *DetectionConditionCache) ListDetections() []DetectionCondition {
	s.mu.Lock()
	defer s.mu.Unlock()

	detections := make([]DetectionCondition, 0, len(s.detectMap))

	for _, v := range s.detectMap {
		detections = append(detections, v)
	}
	return detections
}

func GenerateDetectionKey(c DetectionCondition) string {
	if c.Nep.Namespace == "" {
		return strings.Join([]string{"node", c.Nep.Name, c.ObjectiveEnsuranceName}, ".")
	} else {
		return strings.Join([]string{"pod", c.Nep.Name, c.Nep.Namespace, c.ObjectiveEnsuranceName}, ".")
	}
}

type DetectionStatus struct {
	IsTriggered bool
	LastTime    time.Time
}
