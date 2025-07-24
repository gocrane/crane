package noderesource

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/utils"
)

type NodeResource struct {
	nodeName   string
	nodeLister v1.NodeLister
	podLister  v1.PodLister
}

func NewNodeResourceCollector(nodeName string, nodeLister v1.NodeLister, podLister v1.PodLister) *NodeResource {
	klog.V(4).Infof("NodeResourceCollector create")
	return &NodeResource{
		nodeName:   nodeName,
		nodeLister: nodeLister,
		podLister:  podLister,
	}
}

func (n *NodeResource) GetType() types.CollectType {
	return types.NodeResourceCollectorType
}

func (n *NodeResource) Collect() (map[string][]common.TimeSeries, error) {
	klog.V(6).Infof("NodeResourceCollector Collect")
	node, err := n.nodeLister.Get(n.nodeName)
	if err != nil {
		return nil, err
	}
	pods, err := n.podLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	allExtCpu := node.Status.Allocatable.Name(corev1.ResourceName(fmt.Sprintf(utils.ExtResourcePrefixFormat, corev1.ResourceCPU.String())), resource.DecimalSI).MilliValue()
	var distributeExtCpu int64 = 0
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			if quantity, ok := container.Resources.Requests[corev1.ResourceName(fmt.Sprintf(utils.ExtResourcePrefixFormat, corev1.ResourceCPU.String()))]; ok {
				distributeExtCpu += quantity.MilliValue()
			}
		}
	}
	klog.V(4).Infof("Allocatable Elastic CPU: %d, allocated Elastic CPU: %d", allExtCpu, distributeExtCpu)
	return map[string][]common.TimeSeries{string(types.MetricNameExtCpuTotalDistribute): {{Samples: []common.Sample{{Value: (float64(distributeExtCpu) / float64(allExtCpu)) * 100, Timestamp: time.Now().Unix()}}}}}, nil
}

func (n *NodeResource) Stop() error {
	return nil
}
