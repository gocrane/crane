package collector

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	ensuranceListers "github.com/gocrane/api/pkg/generated/listers/ensurance/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/nodelocal"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/utils"
)

type MetricsCollector struct {
	nodeName   string
	nepLister  ensuranceListers.NodeQOSEnsurancePolicyLister
	podLister  corelisters.PodLister
	nodeLister corelisters.NodeLister
	collectors map[types.CollectType]Collector
	*StateStore
}

type StateStore struct {
	*sync.Map
}

func NewMetricsCollector(nodeName string, nepLister ensuranceListers.NodeQOSEnsurancePolicyLister, podLister corelisters.PodLister, nodeLister corelisters.NodeLister) (*MetricsCollector, *StateStore) {
	stateStore := &StateStore{&sync.Map{}}
	return &MetricsCollector{
		nodeName:   nodeName,
		nepLister:  nepLister,
		podLister:  podLister,
		nodeLister: nodeLister,
		StateStore: stateStore,
		collectors: map[types.CollectType]Collector{},
	}, stateStore
}

func (s *MetricsCollector) Name() string {
	return "MetricsCollector"
}

func (s *MetricsCollector) Run(stop <-chan struct{}) {
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				s.UpdateCollectors()
			case <-stop:
				klog.Infof("MetricsCollector config updater exit")
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
							s.StateStore.Store(key, v)
						}
					} else {
						klog.Errorf("Failed to collect metrics: %v", c.GetType(), err)
					}
				}
			case <-stop:
				klog.Infof("MetricsCollector exit")
				return
			}
		}
	}()

	return
}

func (s *MetricsCollector) UpdateCollectors() {
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
		if matched, err := utils.LabelSelectorMatched(node.Labels, &n.Spec.Selector); err != nil || !matched {
			continue
		}
		if n.Spec.NodeQualityProbe.NodeLocalGet == nil {
			klog.V(4).Infof("Probe type of NEP %s/%s is not node local, continue", n.Namespace, n.Name)
			continue
		}
		nodeLocal = true
		if _, exists := s.collectors[types.NodeLocalCollectorType]; !exists {
			nc := nodelocal.NewNodeLocal(s.podLister)
			s.collectors[types.NodeLocalCollectorType] = nc
		}
		break
	}

	if !nodeLocal {
		if _, exists := s.collectors[types.NodeLocalCollectorType]; exists {
			delete(s.collectors, types.NodeLocalCollectorType)
		}
	}
	return
}

func (s *StateStore) List() map[string][]common.TimeSeries {
	var maps = make(map[string][]common.TimeSeries)

	s.Range(func(key, value interface{}) bool {
		var name = key.(string)
		var series = value.([]common.TimeSeries)
		maps[name] = series
		return true
	})

	return maps
}
