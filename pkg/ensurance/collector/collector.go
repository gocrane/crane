package collector

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	ensuranceListers "github.com/gocrane/api/pkg/generated/listers/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"github.com/gocrane/crane/pkg/ensurance/collector/nodelocal"
	"github.com/gocrane/crane/pkg/ensurance/collector/noderesource"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

type StateCollector struct {
	nodeName          string
	nodeQOSLister     ensuranceListers.NodeQOSLister
	podLister         corelisters.PodLister
	nodeLister        corelisters.NodeLister
	healthCheck       *metrics.HealthCheck
	collectInterval   time.Duration
	ifaces            []string
	exclusiveCPUSet   func() cpuset.CPUSet
	collectors        *sync.Map
	cadvisorManager   cadvisor.Manager
	AnalyzerChann     chan map[string][]common.TimeSeries
	NodeResourceChann chan map[string][]common.TimeSeries
	PodResourceChann  chan map[string][]common.TimeSeries
	State             map[string][]common.TimeSeries
	rw                sync.RWMutex
}

func NewStateCollector(nodeName string, nodeQOSLister ensuranceListers.NodeQOSLister, podLister corelisters.PodLister,
	nodeLister corelisters.NodeLister, ifaces []string, healthCheck *metrics.HealthCheck, collectInterval time.Duration, exclusiveCPUSet func() cpuset.CPUSet, manager cadvisor.Manager) *StateCollector {
	analyzerChann := make(chan map[string][]common.TimeSeries)
	nodeResourceChann := make(chan map[string][]common.TimeSeries)
	podResourceChann := make(chan map[string][]common.TimeSeries)
	State := make(map[string][]common.TimeSeries)
	return &StateCollector{
		nodeName:          nodeName,
		nodeQOSLister:     nodeQOSLister,
		podLister:         podLister,
		nodeLister:        nodeLister,
		healthCheck:       healthCheck,
		collectInterval:   collectInterval,
		ifaces:            ifaces,
		AnalyzerChann:     analyzerChann,
		NodeResourceChann: nodeResourceChann,
		PodResourceChann:  podResourceChann,
		collectors:        &sync.Map{},
		cadvisorManager:   manager,
		exclusiveCPUSet:   exclusiveCPUSet,
		State:             State,
	}
}

func (s *StateCollector) Name() string {
	return "StateCollector"
}

func (s *StateCollector) Run(stop <-chan struct{}) {
	klog.Infof("Starting state collector.")
	s.UpdateCollectors()
	go func() {
		updateTicker := time.NewTicker(s.collectInterval)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				start := time.Now()
				metrics.UpdateLastTime(string(known.ModuleStateCollector), metrics.StepUpdateConfig, start)
				s.healthCheck.UpdateLastConfigUpdate(start)
				s.UpdateCollectors()
				metrics.UpdateDurationFromStart(string(known.ModuleStateCollector), metrics.StepUpdateConfig, start)
			case <-stop:
				s.StopCollectors()
				klog.Infof("StateCollector config updater exit")
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
				start := time.Now()
				metrics.UpdateLastTime(string(known.ModuleStateCollector), metrics.StepMain, start)
				s.healthCheck.UpdateLastActivity(start)
				s.Collect()
				metrics.UpdateDurationFromStart(string(known.ModuleStateCollector), metrics.StepMain, start)
			case <-stop:
				klog.Infof("StateCollector exit")
				return
			}
		}
	}()

	return
}

func (s *StateCollector) Collect() {
	wg := sync.WaitGroup{}
	start := time.Now()

	s.collectors.Range(func(key, value interface{}) bool {
		c := value.(Collector)

		wg.Add(1)
		metrics.UpdateLastTimeWithSubComponent(string(known.ModuleStateCollector), string(c.GetType()), metrics.StepCollect, start)

		go func(c Collector, data map[string][]common.TimeSeries) {
			defer wg.Done()
			defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleStateCollector), string(c.GetType()), metrics.StepCollect, start)

			if cdata, err := c.Collect(); err == nil {
				s.rw.Lock()
				for key, series := range cdata {
					data[key] = series
				}
				s.rw.Unlock()
			}
		}(c, s.State)

		return true
	})

	wg.Wait()

	s.AnalyzerChann <- s.State

	if nodeResource := utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource); nodeResource {
		s.NodeResourceChann <- s.State
	}

	if podResource := utilfeature.DefaultFeatureGate.Enabled(features.CranePodResource); podResource {
		s.PodResourceChann <- s.State
	}
}

func (s *StateCollector) UpdateCollectors() {
	allNodeQOSs, err := s.nodeQOSLister.List(labels.Everything())
	if err != nil {
		klog.Warningf("Failed to list NodeQOS, err %v", err)
	}
	node, err := s.nodeLister.Get(s.nodeName)
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return
	}
	var nodeLocal bool
	for _, n := range allNodeQOSs {
		if matched, err := utils.LabelSelectorMatched(node.Labels, n.Spec.Selector); err != nil || !matched {
			continue
		}

		if n.Spec.NodeQualityProbe.NodeLocalGet == nil {
			klog.V(4).Infof("Probe type of NodeQOS %s/%s is not node local, continue", n.Namespace, n.Name)
			continue
		}

		nodeLocal = true

		if _, exists := s.collectors.Load(types.NodeLocalCollectorType); !exists {
			nc := nodelocal.NewNodeLocal(s.ifaces, s.exclusiveCPUSet)
			s.collectors.Store(types.NodeLocalCollectorType, nc)
		}

		if _, exists := s.collectors.Load(types.CadvisorCollectorType); !exists {
			s.collectors.Store(types.CadvisorCollectorType, cadvisor.NewCadvisorCollector(s.podLister, s.GetCadvisorManager()))
		}

		break
	}
	// if node resource controller is enabled, it indicates local metrics need to be collected no matter nep is defined or not
	if nodeResourceGate := utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource); nodeResourceGate {
		if _, exists := s.collectors.Load(types.NodeLocalCollectorType); !exists {
			nc := nodelocal.NewNodeLocal(s.ifaces, s.exclusiveCPUSet)
			s.collectors.Store(types.NodeLocalCollectorType, nc)
		}

		if _, exists := s.collectors.Load(types.CadvisorCollectorType); !exists {
			s.collectors.Store(types.CadvisorCollectorType, cadvisor.NewCadvisorCollector(s.podLister, s.GetCadvisorManager()))
		}
		if _, exists := s.collectors.Load(types.NodeResourceCollectorType); !exists {
			c := noderesource.NewNodeResourceCollector(s.nodeName, s.nodeLister, s.podLister)
			if c != nil {
				s.collectors.Store(types.NodeResourceCollectorType, c)
			}
		}
		nodeLocal = true
	}
	if !nodeLocal {
		stopCollectors := []types.CollectType{types.NodeLocalCollectorType, types.CadvisorCollectorType}

		for _, collector := range stopCollectors {
			if value, exists := s.collectors.Load(collector); exists {
				s.collectors.Delete(collector)
				c := value.(Collector)
				if err = c.Stop(); err != nil {
					klog.Errorf("Failed to stop the %s manager.", collector)
				}
			}
		}
	}

	return
}

func (s *StateCollector) GetCollectors() *sync.Map {
	return s.collectors
}

func (s *StateCollector) GetCadvisorManager() cadvisor.Manager {
	return s.cadvisorManager
}

func (s *StateCollector) StopCollectors() {
	klog.Infof("StopCollectors")

	s.collectors.Range(func(key, value interface{}) bool {
		s.collectors.Delete(key)
		if key == types.CadvisorCollectorType {
			c := value.(Collector)
			if err := c.Stop(); err != nil {
				klog.Errorf("Failed to stop the cadvisor manager.")
			}
		}
		return true
	})

	return
}

func CheckMetricNameExist(name string) bool {
	if nodelocal.CheckMetricNameExist(name) {
		return true
	}

	if cadvisor.CheckMetricNameExist(name) {
		return true
	}

	return false
}

func (s *StateCollector) GetStateFunc() func() map[string][]common.TimeSeries {
	return func() map[string][]common.TimeSeries {
		s.Collect()
		return s.State
	}
}
