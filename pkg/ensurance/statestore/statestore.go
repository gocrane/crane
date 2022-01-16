package statestore

import (
	"sync"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/statestore/nodelocal"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

type stateStoreManager struct {
	nepInformer cache.SharedIndexInformer

	eventChannel chan types.UpdateEvent
	index        uint64
	configCache  sync.Map

	collectors  []collector
	StatusCache sync.Map
}

func NewStateStoreManager(nepInformer cache.SharedIndexInformer) StateStore {
	var eventChan = make(chan types.UpdateEvent)
	return &stateStoreManager{nepInformer: nepInformer, eventChannel: eventChan}
}

func (s *stateStoreManager) Name() string {
	return "StateStoreManager"
}

func (s *stateStoreManager) Run(stop <-chan struct{}) {
	// check need to update config
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				if s.checkConfig() {
					s.index++
					klog.V(6).Infof("StateStore update event, index: %v", s.index)
					s.eventChannel <- types.UpdateEvent{Index: s.index}
				} else {
					klog.V(6).Info("StateStore config false, not to update")
				}
				continue
			case <-stop:
				klog.Infof("StateStore config check exit")
				return
			}
		}
	}()

	// do collect periodically
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				for _, c := range s.collectors {
					if data, err := c.Collect(); err == nil {
						for key, v := range data {
							s.StatusCache.Store(key, v)
						}
					} else {
						klog.Errorf("Failed to collect metrics: %v", c.GetType(), err)
					}
				}
				continue
			case v := <-s.eventChannel:
				klog.V(6).Infof("StateStore update config, index: %v", v.Index)
				s.updateConfig()
			case <-stop:
				klog.Infof("StateStore collect exit")
				return
			}
		}
	}()

	return
}

func (s *stateStoreManager) List() map[string][]common.TimeSeries {
	var maps = make(map[string][]common.TimeSeries)

	s.StatusCache.Range(func(key, value interface{}) bool {
		var name = key.(string)
		var series = value.([]common.TimeSeries)
		maps[name] = series
		return true
	})

	return maps
}

func (s *stateStoreManager) checkConfig() bool {
	// step1 copy neps
	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()

	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy).DeepCopy()
		if nep.Spec.NodeQualityProbe.NodeLocalGet == nil {
			klog.V(4).Infof("Warning: skip the config as it not node-local, it will support other kind of config in the future")
			continue
		}
		neps = append(neps, nep)
	}

	// step2 check it needs to update
	var nodeLocal bool
	for _, n := range neps {
		if n.Spec.NodeQualityProbe.NodeLocalGet != nil {
			nodeLocal = true
			if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); !ok {
				return true
			}
		}
	}

	if !nodeLocal {
		if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); ok {
			return true
		}
	}

	return false
}

func (s *stateStoreManager) updateConfig() {
	// step1 copy neps
	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()
	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy).DeepCopy()
		if nep.Spec.NodeQualityProbe.NodeLocalGet == nil {
			klog.V(4).Infof("Warning: skip the config as it not node-local, it will support other kind of config in the future")
			continue
		}

		neps = append(neps, nep)
	}

	// step2 update the config
	var nodeLocal bool
	for _, n := range neps {
		if n.Spec.NodeQualityProbe.NodeLocalGet != nil {
			nodeLocal = true
			if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); !ok {
				nc := nodelocal.NewNodeLocal()
				s.collectors = append(s.collectors, nc)
				s.configCache.Store(string(types.NodeLocalCollectorType), types.MetricNameConfigs{})
			}
		}
	}

	if !nodeLocal {
		if _, ok := s.configCache.Load(string(types.NodeLocalCollectorType)); ok {
			s.configCache.Delete(string(types.NodeLocalCollectorType))
			var collectors []collector
			for _, c := range s.collectors {
				if c.GetType() != types.NodeLocalCollectorType {
					collectors = append(collectors, c)
				}
			}
			s.collectors = collectors
		}
	}

	return
}
