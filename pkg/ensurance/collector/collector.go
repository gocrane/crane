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

type StateCollector struct {
	nodeName   string
	nepLister  ensuranceListers.NodeQOSEnsurancePolicyLister
	podLister  corelisters.PodLister
	nodeLister corelisters.NodeLister
	ifaces     []string
	collectors *sync.Map
	StateChann chan map[string][]common.TimeSeries
}

type StateStore struct {
	*sync.Map
}

func NewStateCollector(nodeName string, nepLister ensuranceListers.NodeQOSEnsurancePolicyLister, podLister corelisters.PodLister, nodeLister corelisters.NodeLister, ifaces []string) *StateCollector {
	stateChann := make(chan map[string][]common.TimeSeries)
	return &StateCollector{
		nodeName:   nodeName,
		nepLister:  nepLister,
		podLister:  podLister,
		nodeLister: nodeLister,
		ifaces:     ifaces,
		StateChann: stateChann,
		collectors: &sync.Map{},
	}
}

func (s *StateCollector) Name() string {
	return "StateCollector"
}

func (s *StateCollector) Run(stop <-chan struct{}) {
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				s.UpdateCollectors()
			case <-stop:
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
				s.collectors.Range(func(key, value interface{}) bool {
					c := value.(Collector)
					if data, err := c.Collect(); err == nil {
						s.StateChann <- data
					} else {
						klog.Errorf("Failed to collect metrics: %v", c.GetType(), err)
					}
					return true
				})
			case <-stop:
				klog.Infof("StateCollector exit")
				return
			}
		}
	}()

	return
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
		if matched, err := utils.LabelSelectorMatched(node.Labels, &n.Spec.Selector); err != nil || !matched {
			continue
		}
		if n.Spec.NodeQualityProbe.NodeLocalGet == nil {
			klog.V(4).Infof("Probe type of NEP %s/%s is not node local, continue", n.Namespace, n.Name)
			continue
		}
		nodeLocal = true
		if _, exists := s.collectors.Load(types.NodeLocalCollectorType); exists {
			nc := nodelocal.NewNodeLocal(s.podLister, s.ifaces)
			s.collectors.Store(types.NodeLocalCollectorType, nc)
			break
		}
	}

	if !nodeLocal {
		if _, exists := s.collectors.Load(types.NodeLocalCollectorType); exists {
			s.collectors.Delete(types.NodeLocalCollectorType)
		}
	}
	return
}
