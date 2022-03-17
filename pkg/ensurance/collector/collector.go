package collector

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	ensuranceListers "github.com/gocrane/api/pkg/generated/listers/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"github.com/gocrane/crane/pkg/ensurance/collector/nodelocal"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

type StateCollector struct {
	nodeName          string
	nepLister         ensuranceListers.NodeQOSEnsurancePolicyLister
	podLister         corelisters.PodLister
	nodeLister        corelisters.NodeLister
	healthCheck       *metrics.HealthCheck
	collectInterval   time.Duration
	ifaces            []string
	collectors        *sync.Map
	cadvisorManager   cadvisor.Manager
	cadvisorCollect   Collector
	AnalyzerChann     chan map[string][]common.TimeSeries
	NodeResourceChann chan map[string][]common.TimeSeries
	PodResourceChann  chan map[string][]common.TimeSeries
}

func NewStateCollector(nodeName string, nepLister ensuranceListers.NodeQOSEnsurancePolicyLister, podLister corelisters.PodLister,
	nodeLister corelisters.NodeLister, ifaces []string, healthCheck *metrics.HealthCheck, collectInterval time.Duration) *StateCollector {
	analyzerChann := make(chan map[string][]common.TimeSeries)
	nodeResourceChann := make(chan map[string][]common.TimeSeries)
	podResourceChann := make(chan map[string][]common.TimeSeries)

	c := cadvisor.NewCadvisor(podLister)

	return &StateCollector{
		nodeName:          nodeName,
		nepLister:         nepLister,
		podLister:         podLister,
		nodeLister:        nodeLister,
		healthCheck:       healthCheck,
		collectInterval:   collectInterval,
		ifaces:            ifaces,
		AnalyzerChann:     analyzerChann,
		NodeResourceChann: nodeResourceChann,
		PodResourceChann:  podResourceChann,
		collectors:        &sync.Map{},
		cadvisorManager:   &c.Manager,
		cadvisorCollect:   c,
	}
}

func (s *StateCollector) Name() string {
	return "StateCollector"
}

func (s *StateCollector) Run(stop <-chan struct{}) {
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

	var data = make(map[string][]common.TimeSeries)
	var mux sync.Mutex

	s.collectors.Range(func(key, value interface{}) bool {
		c := value.(Collector)

		wg.Add(1)
		metrics.UpdateLastTimeWithSubComponent(string(known.ModuleStateCollector), string(c.GetType()), metrics.StepCollect, start)

		go func(c Collector, data map[string][]common.TimeSeries) {
			defer wg.Done()
			defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleStateCollector), string(c.GetType()), metrics.StepCollect, start)

			if cdata, err := c.Collect(); err == nil {
				mux.Lock()
				for key, series := range cdata {
					data[key] = series
				}
				mux.Unlock()
			}
		}(c, data)

		return true
	})

	wg.Wait()

	s.AnalyzerChann <- data

	if nodeResource := utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource); nodeResource {
		s.NodeResourceChann <- data
	}

	if podResource := utilfeature.DefaultFeatureGate.Enabled(features.CranePodResource); podResource {
		s.PodResourceChann <- data
	}
}

func (s *StateCollector) UpdateCollectors() {
	allNeps, err := s.nepLister.List(labels.Everything())
	if err != nil {
		klog.Warningf("Failed to list NodeQOSEnsurancePolicy, err %v", err)
	}
	node, err := s.nodeLister.Get(s.nodeName)
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return
	}
	var nodeLocal bool
	for _, n := range allNeps {
		if matched, err := utils.LabelSelectorMatched(node.Labels, n.Spec.Selector); err != nil || !matched {
			continue
		}

		if n.Spec.NodeQualityProbe.NodeLocalGet == nil {
			klog.V(4).Infof("Probe type of NEP %s/%s is not node local, continue", n.Namespace, n.Name)
			continue
		}

		nodeLocal = true

		if _, exists := s.collectors.Load(types.NodeLocalCollectorType); !exists {
			nc := nodelocal.NewNodeLocal(s.ifaces)
			s.collectors.Store(types.NodeLocalCollectorType, nc)
		}

		if _, exists := s.collectors.Load(types.CadvisorCollectorType); !exists {
			s.collectors.Store(types.CadvisorCollectorType, s.cadvisorCollect)
		}

		break
	}

	if !nodeLocal {
		stopCollectors := []types.CollectType{types.NodeLocalCollectorType, types.CadvisorCollectorType}

		for _, collector := range stopCollectors {
			if _, exists := s.collectors.Load(collector); exists {
				s.collectors.Delete(collector)
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
		return true
	})

	if s.cadvisorCollect != nil {
		if err := s.cadvisorCollect.Stop(); err != nil {
			klog.Errorf("Failed to stop the cadvisor manager.")
		}
	}

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
