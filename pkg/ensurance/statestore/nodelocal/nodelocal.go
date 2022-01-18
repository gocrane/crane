package nodelocal

import (
	"strings"

	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

type newCollectorFunc func(podLister corelisters.PodLister) (nodeLocalCollector, error)

var nodeLocalMetric = make(map[string][]types.MetricName, 10)
var nodeLocalFunc = make(map[string]newCollectorFunc, 10)

func registerMetrics(collectorName string, metricsNames []types.MetricName, newCollector newCollectorFunc) {
	if _, ok := nodeLocalMetric[collectorName]; ok {
		klog.Infof("Warning: node local metrics collectorName %s is registered, not to register again", collectorName)
		return
	}

	nodeLocalMetric[collectorName] = metricsNames
	nodeLocalFunc[collectorName] = newCollector
}

type NodeLocal struct {
	Name types.CollectType
	nlcs []nodeLocalCollector
}

func NewNodeLocal(podLister corelisters.PodLister) *NodeLocal {
	klog.V(2).Infof("NewNodeLocal")

	n := NodeLocal{
		Name: types.NodeLocalCollectorType,
	}

	// the first version collect all metrics
	// Open the on demand，in the future
	for _, f := range nodeLocalFunc {
		if c, err := f(podLister); err == nil {
			n.nlcs = append(n.nlcs, c)
		} else {
			klog.Errorf("Failed to now node local collector: %v", err)
		}
	}

	return &n
}

func (n *NodeLocal) GetType() types.CollectType {
	return n.Name
}

func (n *NodeLocal) Collect() (map[string][]common.TimeSeries, error) {
	klog.V(6).Infof("Node local collecting")

	var status = make(map[string][]common.TimeSeries)
	for _, c := range n.nlcs {
		if data, err := c.collect(); err == nil {
			for key, d := range data {
				status[key] = d
			}
		} else {
			if !strings.Contains(err.Error(), "collect_init") {
				klog.Errorf("Failed to collect node local metrics: %v", c.name(), err)
			}
		}
	}

	klog.V(10).Info("Node local collecting, status: %v", status)

	return status, nil
}

func CheckMetricNameExist(name types.MetricName) bool {
	for _, v := range nodeLocalMetric {
		for _, vv := range v {
			if vv == name {
				return true
			}
		}
	}
	return false
}
