package noderesource

import (
	"fmt"
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	noderesourceMamager "github.com/gocrane/crane/pkg/noderesource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
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
	klog.V(4).Infof("NodeResourceCollector Collect")
	node, err := n.nodeLister.Get(n.nodeName)
	if err != nil {
		return nil, err
	}
	pods, err := n.podLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	allExtCpu := node.Status.Allocatable.Name(corev1.ResourceName(fmt.Sprintf(noderesourceMamager.ExtResourcePrefix, corev1.ResourceCPU.String())), resource.DecimalSI).Value()
	var distributeExtCpu int64 = 0
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			if quantity, ok := container.Resources.Requests[corev1.ResourceName(fmt.Sprintf(noderesourceMamager.ExtResourcePrefix, corev1.ResourceCPU.String()))]; ok {
				distributeExtCpu += quantity.Value()
			}
		}
	}
	klog.V(4).Infof("allExtCpu: %d, distributeExtCpu: %d", allExtCpu, distributeExtCpu)
	return map[string][]common.TimeSeries{string(types.MetricNameExtCpuTotalDistribute): {{Samples: []common.Sample{{Value: (float64(distributeExtCpu) / float64(allExtCpu)) * 100, Timestamp: time.Now().Unix()}}}}}, nil
}
