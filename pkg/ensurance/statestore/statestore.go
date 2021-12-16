package statestore

import (
	"fmt"
	"sync"
	"time"

	"github.com/gocrane/crane/pkg/common"

	"k8s.io/client-go/tools/cache"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/ensurance/statestore/nodelocal"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
	"github.com/gocrane/crane/pkg/utils/log"
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
				log.Logger().V(4).Info("StateStore config check run periodically")
				if s.checkConfig() {
					s.index++
					log.Logger().V(4).Info("StateStore update event", "index", s.index)
					s.eventChannel <- types.UpdateEvent{Index: s.index}
				} else {
					log.Logger().V(4).Info("StateStore config false, not to update")
				}
				continue
			case <-stop:
				log.Logger().Info("StateStore config check exit")
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
				log.Logger().V(2).Info("StateStore run periodically")
				for _, c := range s.collectors {
					if data, err := c.Collect(); err == nil {
						for key, v := range data {
							s.StatusCache.Store(key, v)
						}
					} else {
						log.Logger().Error(err, "StateStore collect failed", c.GetType())
					}
				}
				continue
			case v := <-s.eventChannel:
				log.Logger().V(3).Info("StateStore update config index", "Index", v.Index)
				s.updateConfig()
			case <-stop:
				log.Logger().V(2).Info("StateStore exit")
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
	log.Logger().V(4).Info("StateStore checkConfig")

	// step1 copy neps
	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()

	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy).DeepCopy()
		if nep.Spec.NodeQualityProbe.NodeLocalGet == nil {
			log.Logger().V(4).Info("Warning: skip the config not node-local, it will support other kind of config in the future")
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

	log.Logger().V(4).Info("checkConfig", "nodeLocal", nodeLocal)

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
		log.Logger().V(4).Info(fmt.Sprintf("nep: %#v", nep))
		if nep.Spec.NodeQualityProbe.NodeLocalGet == nil {
			log.Logger().V(4).Info("Warning: skip the config not node-local, it will support other kind of config in the future")
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
