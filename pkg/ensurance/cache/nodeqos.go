package cache

import (
	"sync"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
)

type NodeQOSCache struct {
	mu         sync.Mutex
	nodeQOSMap map[string]*ensuranceapi.NodeQOS
}

func (s *NodeQOSCache) Init() {
	s.nodeQOSMap = make(map[string]*ensuranceapi.NodeQOS)
}

// ListKeys implements the interface required by DeltaFIFO to list the keys we
// already know about.
func (s *NodeQOSCache) ListKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.nodeQOSMap))
	for k := range s.nodeQOSMap {
		keys = append(keys, k)
	}
	return keys
}

func (s *NodeQOSCache) Get(name string) (*ensuranceapi.NodeQOS, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nodeQOS, ok := s.nodeQOSMap[name]
	return nodeQOS, ok
}

func (s *NodeQOSCache) Exist(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.nodeQOSMap[name]
	return ok
}

func (s *NodeQOSCache) GetOrCreate(nodeQOS *ensuranceapi.NodeQOS) *ensuranceapi.NodeQOS {
	s.mu.Lock()
	defer s.mu.Unlock()
	cache, ok := s.nodeQOSMap[nodeQOS.Name]
	if !ok {
		s.nodeQOSMap[nodeQOS.Name] = nodeQOS
	}
	return cache
}

func (s *NodeQOSCache) Set(nodeQOS *ensuranceapi.NodeQOS) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeQOSMap[nodeQOS.Name] = nodeQOS
}

func (s *NodeQOSCache) Delete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nodeQOSMap, name)
}
