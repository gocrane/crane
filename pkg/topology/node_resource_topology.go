package topology

import (
	"sort"

	topologyapi "github.com/gocrane/api/topology/v1alpha1"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/topology"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/utils"
)

// NRTBuilder helps to build NRT. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type NRTBuilder struct {
	node                  *corev1.Node
	cpuManagerPolicy      topologyapi.CPUManagerPolicy
	topologyManagerPolicy topologyapi.TopologyManagerPolicy
	reserved              corev1.ResourceList
	reservedCPUs          int
	topologyInfo          *topology.Info
}

// NewNRTBuilder returns a new NRTBuilder.
func NewNRTBuilder() *NRTBuilder {
	return &NRTBuilder{}
}

// WithNode sets the node property of a Builder.
func (b *NRTBuilder) WithNode(node *corev1.Node) {
	b.node = node
	if aware := utils.IsNodeAwareOfTopology(b.node.Labels); aware != nil && *aware == false {
		b.topologyManagerPolicy = topologyapi.TopologyManagerPolicyNone
	} else {
		b.topologyManagerPolicy = topologyapi.TopologyManagerPolicySingleNUMANodePodLevel
	}
}

// WithCPUManagerPolicy sets the cpuManagerPolicy property of a Builder.
func (b *NRTBuilder) WithCPUManagerPolicy(cpuManagerPolicy topologyapi.CPUManagerPolicy) {
	b.cpuManagerPolicy = cpuManagerPolicy
}

// WithReserved sets the reserved property of a Builder.
func (b *NRTBuilder) WithReserved(reserved corev1.ResourceList) {
	b.reserved = reserved
}

// WithReservedCPUs sets the reserved property of a Builder.
func (b *NRTBuilder) WithReservedCPUs(reservedCPUs int) {
	b.reservedCPUs = reservedCPUs
}

// WithTopologyInfo sets the topologyInfo property of a Builder.
func (b *NRTBuilder) WithTopologyInfo(topologyInfo *topology.Info) {
	sort.Slice(topologyInfo.Nodes, func(i, j int) bool {
		return topologyInfo.Nodes[i].ID < topologyInfo.Nodes[j].ID
	})
	b.topologyInfo = topologyInfo
}

// Build initializes and build all fields of NRT.
func (b *NRTBuilder) Build() *topologyapi.NodeResourceTopology {
	nrt := &topologyapi.NodeResourceTopology{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.node.Name,
		},
	}
	b.buildNodeScopeFields(nrt)
	b.buildZoneScopeFields(nrt)
	return nrt
}

func (b *NRTBuilder) buildNodeScopeFields(nrt *topologyapi.NodeResourceTopology) {
	nrt.CraneManagerPolicy.CPUManagerPolicy = b.cpuManagerPolicy
	nrt.CraneManagerPolicy.TopologyManagerPolicy = b.topologyManagerPolicy
	nrt.Reserved = b.reserved.DeepCopy()
}

func (b *NRTBuilder) buildZoneScopeFields(nrt *topologyapi.NodeResourceTopology) {
	var zones topologyapi.ZoneList
	reserved := b.reserved.DeepCopy()
	reservedCPUs := b.reservedCPUs
	j := 0
	for _, node := range b.topologyInfo.Nodes {
		if node.ID < 0 || node.ID > len(b.topologyInfo.Nodes) {
			klog.Warningf("ID %d is not between 0 and len(nodes), ignore", node.ID)
			continue
		}
		zone := topologyapi.Zone{
			Name:      utils.BuildZoneName(node.ID),
			Type:      topologyapi.ZoneTypeNode,
			Costs:     buildCostsPerNUMANode(node),
			Resources: buildNodeResource(node, reserved, &reservedCPUs),
		}
		for j < len(nrt.Zones) && nrt.Zones[j].Name != zone.Name {
			j++
		}
		if j < len(nrt.Zones) {
			zone.Attributes = nrt.Zones[j].Attributes
		}
		zones = append(zones, zone)
	}
	nrt.Zones = zones
}

// buildCostsPerNUMANode builds the cost map to reach all the known NUMA zones
// (mapping (NUMA zone) -> cost) starting from the given NUMA zone.
func buildCostsPerNUMANode(node *ghw.TopologyNode) []topologyapi.CostInfo {
	nodeCosts := make([]topologyapi.CostInfo, 0, len(node.Distances))
	for nodeIDDst, dist := range node.Distances {
		nodeCosts = append(nodeCosts, topologyapi.CostInfo{
			Name:  utils.BuildZoneName(nodeIDDst),
			Value: int64(dist),
		})
	}
	return nodeCosts
}

func buildNodeResource(node *ghw.TopologyNode, reserved corev1.ResourceList, reservedCPUs *int) *topologyapi.ResourceInfo {
	logicalCores := 0
	for _, core := range node.Cores {
		logicalCores += len(core.LogicalProcessors)
	}
	capacity := make(corev1.ResourceList)
	capacity[corev1.ResourceCPU] = *resource.NewQuantity(int64(logicalCores), resource.DecimalSI)
	if node.Memory != nil {
		capacity[corev1.ResourceMemory] = *resource.NewQuantity(node.Memory.TotalUsableBytes, resource.BinarySI)
	}
	allocatable := getNodeAllocatable(capacity, reserved)
	var reservedCPUNums int
	if logicalCores >= *reservedCPUs {
		reservedCPUNums = *reservedCPUs
		*reservedCPUs = 0
	} else {
		reservedCPUNums = logicalCores
		*reservedCPUs -= logicalCores
	}
	return &topologyapi.ResourceInfo{
		Capacity:        capacity,
		Allocatable:     allocatable,
		ReservedCPUNums: int32(reservedCPUNums),
	}
}

func getNodeAllocatable(capacity, reserved corev1.ResourceList) corev1.ResourceList {
	result := make(corev1.ResourceList)
	if reserved == nil {
		return result
	}
	for k, v := range capacity {
		value := v.DeepCopy()
		value.Sub(reserved[k])
		if value.Sign() >= 0 {
			delete(reserved, k)
		} else {
			// Negative Allocatable resources don't make sense.
			quantity := reserved[k]
			quantity.Sub(value)
			reserved[k] = quantity
			value.Set(0)
		}
		result[k] = value
	}
	return result
}
