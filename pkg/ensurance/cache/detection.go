package cache

import (
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
)

type ActionContext struct {
	// the Objective Ensurance name
	RuleName string
	// strategy for the action
	Strategy ensuranceapi.AvoidanceActionStrategy
	// if the policy triggered action
	Triggered bool
	// if the policy triggered restored action
	Restored bool
	// action name
	ActionName string
	// node qos ensurance policy
	NodeQOS *ensuranceapi.NodeQOS
	// time for detection
	Time time.Time
	// the influenced pod list
	// node detection the pod list is empty
	BeInfluencedPods []types.NamespacedName
}

type ActionContextCache struct {
	mu               sync.Mutex // protects actionContextMap
	actionContextMap map[string]ActionContext
}

func (s *ActionContextCache) GetOrCreate(c ActionContext) ActionContext {
	s.mu.Lock()
	defer s.mu.Unlock()

	var key = GenerateDetectionKey(c)

	actionContext, ok := s.actionContextMap[key]
	if !ok {
		s.actionContextMap[key] = c
	}
	return actionContext
}

func (s *ActionContextCache) Get(key string) (ActionContext, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actionContext, ok := s.actionContextMap[key]
	return actionContext, ok
}

func (s *ActionContextCache) Set(c ActionContext) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var key = GenerateDetectionKey(c)
	s.actionContextMap[key] = c

	return
}

func (s *ActionContextCache) Exist(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.actionContextMap[key]
	return ok
}

func (s *ActionContextCache) ListDetections() []ActionContext {
	s.mu.Lock()
	defer s.mu.Unlock()

	actionContexts := make([]ActionContext, 0, len(s.actionContextMap))

	for _, v := range s.actionContextMap {
		actionContexts = append(actionContexts, v)
	}
	return actionContexts
}

func GenerateDetectionKey(c ActionContext) string {
	return strings.Join([]string{"node", c.NodeQOS.Name, c.RuleName}, ".")
}

type DetectionStatus struct {
	IsTriggered bool
	LastTime    time.Time
}
