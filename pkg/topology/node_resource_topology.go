package topology

import (
	"sort"
	"strconv"

	topologyapi "github.com/gocrane/api/topology/v1alpha1"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/topology"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

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
	attributes            map[string]string
	systemReservedCPUs    cpuset.CPUSet
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

// WithSystemReservedCPUs sets the systemReservedCPUs property of a Builder.
func (b *NRTBuilder) WithSystemReservedCPUs(systemReservedCPUs cpuset.CPUSet) {
	b.systemReservedCPUs = systemReservedCPUs
}

// WithTopologyInfo sets the topologyInfo property of a Builder.
func (b *NRTBuilder) WithTopologyInfo(topologyInfo *topology.Info) {
	sort.Slice(topologyInfo.Nodes, func(i, j int) bool {
		return topologyInfo.Nodes[i].ID < topologyInfo.Nodes[j].ID
	})
	b.topologyInfo = topologyInfo
}

// WithAttributes sets the attributes of a Builder.
func (b *NRTBuilder) WithAttributes(attributes map[string]string) {
	b.attributes = attributes
}

// Build initializes and build all fields of NRT.
func (b *NRTBuilder) Build() *topologyapi.NodeResourceTopology {
	nrt := &topologyapi.NodeResourceTopology{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.node.Name,
		},
		Attributes: b.attributes,
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
			Resources: buildNodeResource(node, reserved, &reservedCPUs, b.systemReservedCPUs),
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

func buildNodeResource(node *ghw.TopologyNode, reserved corev1.ResourceList,
	totalReservedCPUNums *int, systemReservedCPUs cpuset.CPUSet) *topologyapi.ResourceInfo {
	logicalCores := 0
	var logicalCoreList []int
	for _, core := range node.Cores {
		logicalCores += len(core.LogicalProcessors)
		logicalCoreList = append(logicalCoreList, core.LogicalProcessors...)
	}
	logicalCoreSet := cpuset.NewCPUSet(logicalCoreList...)

	capacity := make(corev1.ResourceList)
	capacity[corev1.ResourceCPU] = *resource.NewQuantity(int64(logicalCores), resource.DecimalSI)
	if node.Memory != nil {
		capacity[corev1.ResourceMemory] = *resource.NewQuantity(node.Memory.TotalUsableBytes, resource.BinarySI)
	}

	reservedCPUNums := 0
	nodeReservedCPUs := cpuset.NewCPUSet()
	// if `systemReservedCPUs` specified, build reserved cpu according to the systemReservedCPUs only
	if systemReservedCPUs.Size() != 0 {
		nodeReservedCPUs = logicalCoreSet.Intersection(systemReservedCPUs)
		reservedCPUNums = nodeReservedCPUs.Size()
	} else {
		if logicalCores >= *totalReservedCPUNums {
			reservedCPUNums = *totalReservedCPUNums
		} else {
			reservedCPUNums = logicalCores
		}
	}
	// update total reserved cpu nums
	*totalReservedCPUNums -= reservedCPUNums

	allocatable := getNodeAllocatable(capacity, reserved, systemReservedCPUs, nodeReservedCPUs)

	return &topologyapi.ResourceInfo{
		Capacity:        capacity,
		Allocatable:     allocatable,
		ReservedCPUNums: int32(reservedCPUNums),
	}
}

func getNodeAllocatable(capacity, reserved corev1.ResourceList, systemReservedCPUs, nodeReservedCPUs cpuset.CPUSet) corev1.ResourceList {
	result := make(corev1.ResourceList)
	if reserved == nil {
		return result
	}

	for k, v := range capacity {
		value := v.DeepCopy()
		toReserve := reserved[k]
		if k == corev1.ResourceCPU && systemReservedCPUs.Size() != 0 {
			toReserve = resource.MustParse(strconv.Itoa(nodeReservedCPUs.Size()))
		}

		value.Sub(toReserve)
		if value.Sign() >= 0 {
			delete(reserved, k)
		} else {
			// Negative Allocatable resources don't make sense.
			quantity := toReserve
			quantity.Sub(value)
			reserved[k] = quantity
			value.Set(0)
		}
		result[k] = value
	}
	return result
}
