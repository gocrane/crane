package nodelocal

import (
	"fmt"
	"strings"

	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
	"github.com/gocrane/crane/pkg/utils"
	"github.com/gocrane/crane/pkg/utils/clogs"
)

type newCollectorFunc func() (nodeLocalCollector, error)

var nodeLocalMetric = make(map[string][]types.MetricName, 10)
var nodeLocalFunc = make(map[string]newCollectorFunc, 10)

func registerMetrics(collectorName string, metricsNames []types.MetricName, newCollector newCollectorFunc) {
	if _, ok := nodeLocalMetric[collectorName]; ok {
		clogs.Log().V(2).Info(
			fmt.Sprintf("Warning: node local metrics collectorName %s is registered, not to register again", collectorName))
		return
	}

	nodeLocalMetric[collectorName] = metricsNames
	nodeLocalFunc[collectorName] = newCollector
}

type NodeLocal struct {
	Name types.CollectType
	nlcs []nodeLocalCollector
}

func NewNodeLocal() *NodeLocal {
	clogs.Log().V(1).Info("NewNodeLocal")

	n := NodeLocal{
		Name: types.NodeLocalCollectorType,
	}

	// the first version collect all metrics
	// Open the on demand，in the future
	for _, f := range nodeLocalFunc {
		if c, err := f(); err == nil {
			n.nlcs = append(n.nlcs, c)
		} else {
			clogs.Log().Error(err, "NewNodeLocal init failed")
		}
	}

	return &n
}

func (n *NodeLocal) GetType() types.CollectType {
	return n.Name
}

func (n *NodeLocal) Collect() (map[string][]utils.TimeSeries, error) {
	clogs.Log().V(5).Info("Node local collecting")

	var status = make(map[string][]utils.TimeSeries)
	for _, c := range n.nlcs {
		if data, err := c.collect(); err == nil {
			for key, d := range data {
				status[key] = d
			}
		} else {
			if !strings.Contains(err.Error(), "collect_init") {
				clogs.Log().Error(err, fmt.Sprintf("NodeLocal collect %s", c.name()))
			}
		}
	}

	clogs.Log().V(5).Info("Node local collecting", "status", status)

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
