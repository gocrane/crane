package nodelocal

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

type nodeLocalContext struct {
	nodeState       *nodeState
	exclusiveCPUSet func() cpuset.CPUSet
}

type collectFunc func(nodeLocalContext *nodeLocalContext) (map[string][]common.TimeSeries, error)

var nodeLocalMetric = make(map[string][]types.MetricName, 10)
var collectFuncMap = make(map[string]collectFunc, 10)

func registerCollector(collectorName string, metricsNames []types.MetricName, collectorFunc collectFunc) {
	if _, ok := nodeLocalMetric[collectorName]; ok {
		klog.Infof("Warning: node local metrics collectorName %s is registered, not to register again", collectorName)
		return
	}

	nodeLocalMetric[collectorName] = metricsNames
	collectFuncMap[collectorName] = collectorFunc
}

type nodeState struct {
	cpuCoreNumbers   uint64
	latestCpuState   *CpuTimeStampState
	latestDiskStates map[string]DiskState
	ifaces           sets.String
	latestNetStates  map[string]NetTimeStampState
}

type NodeLocal struct {
	name            types.CollectType
	nodeState       *nodeState
	exclusiveCPUSet func() cpuset.CPUSet
}

func NewNodeLocal(ifaces []string, exclusiveCPUSet func() cpuset.CPUSet) *NodeLocal {
	klog.V(2).Infof("New NodeLocal collector on interfaces %v", ifaces)

	n := NodeLocal{
		name:            types.NodeLocalCollectorType,
		nodeState:       &nodeState{ifaces: sets.NewString(ifaces...)},
		exclusiveCPUSet: exclusiveCPUSet,
	}

	return &n
}

func (n *NodeLocal) GetType() types.CollectType {
	return n.name
}

func (n *NodeLocal) Collect() (map[string][]common.TimeSeries, error) {
	klog.V(6).Infof("Node local collecting")

	var status = make(map[string][]common.TimeSeries)
	nodeLocalContext := &nodeLocalContext{
		nodeState:       n.nodeState,
		exclusiveCPUSet: n.exclusiveCPUSet,
	}
	for name, collect := range collectFuncMap {
		if data, err := collect(nodeLocalContext); err == nil {
			for key, d := range data {
				status[key] = d
			}
		} else {
			if !strings.Contains(err.Error(), types.CollectInitErrorText) {
				klog.Errorf("Failed to collect node local metrics: %v", name, err)
			}
		}
	}

	klog.V(6).Info("Node local collecting, status: %#v", status)

	return status, nil
}

func (n *NodeLocal) Stop() error {
	return nil
}

func CheckMetricNameExist(name string) bool {
	for _, v := range nodeLocalMetric {
		for _, vv := range v {
			if string(vv) == name {
				return true
			}
		}
	}
	return false
}
