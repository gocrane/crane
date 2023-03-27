package topology

import (
	"fmt"
	"testing"

	topologyapi "github.com/gocrane/api/topology/v1alpha1"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/cpu"
	"github.com/jaypipes/ghw/pkg/topology"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/utils/pointer"
)

func Test_getNodeAllocatable(t *testing.T) {
	type args struct {
		capacity           v1.ResourceList
		reserved           v1.ResourceList
		systemReservedCPUs cpuset.CPUSet
		nodeReservedCPUs   cpuset.CPUSet
	}
	tests := []struct {
		name string
		args args
		want resource.Quantity
	}{
		{
			name: "numa node has reserved cpus",
			args: args{
				capacity: map[v1.ResourceName]resource.Quantity{
					v1.ResourceCPU: resource.MustParse("4"),
				},
				reserved: map[v1.ResourceName]resource.Quantity{
					v1.ResourceCPU: resource.MustParse("2"),
				},
				systemReservedCPUs: cpuset.NewCPUSet(1, 2),
				nodeReservedCPUs:   cpuset.NewCPUSet(1),
			},
			want: resource.MustParse("3"),
		},
		{
			name: "numa node does not has reserved cpus",
			args: args{
				capacity: map[v1.ResourceName]resource.Quantity{
					v1.ResourceCPU: resource.MustParse("4"),
				},
				reserved: map[v1.ResourceName]resource.Quantity{
					v1.ResourceCPU: resource.MustParse("2"),
				},
				systemReservedCPUs: cpuset.NewCPUSet(),
				nodeReservedCPUs:   cpuset.NewCPUSet(),
			},
			want: resource.MustParse("2"),
		},
		{
			name: "numa node resource exceed reserved cpus",
			args: args{
				capacity: map[v1.ResourceName]resource.Quantity{
					v1.ResourceCPU: resource.MustParse("4"),
				},
				reserved: map[v1.ResourceName]resource.Quantity{
					v1.ResourceCPU: resource.MustParse("5"),
				},
				systemReservedCPUs: cpuset.NewCPUSet(1, 2, 3, 4, 5),
				nodeReservedCPUs:   cpuset.NewCPUSet(1, 2, 3, 4, 5),
			},
			want: resource.MustParse("0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNodeAllocatable(tt.args.capacity, tt.args.reserved, tt.args.systemReservedCPUs, tt.args.nodeReservedCPUs)
			if !got[v1.ResourceCPU].Equal(tt.want) {
				t.Errorf("getNodeAllocatable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func buildTopologyInfo(numNodes int, numCoresPerNuma, numThreadsPerCore int) *ghw.TopologyInfo {
	info := &ghw.TopologyInfo{
		Nodes: []*ghw.TopologyNode{},
	}

	corePivot := pointer.Int(0)
	threadPivot := pointer.Int(0)
	for i := 0; i < numNodes; i++ {
		node := buildTopologyNode(i, corePivot, threadPivot, numCoresPerNuma, numThreadsPerCore)
		info.Nodes = append(info.Nodes, node)
	}

	fmt.Println(info.String())
	return info
}

func buildTopologyNode(nodeID int, corePivot, threadPivot *int, numCores, numThreadsPerCore int) *ghw.TopologyNode {
	node := &ghw.TopologyNode{
		ID: nodeID,
	}

	for i := 0; i < numCores; i++ {
		processorCore := &cpu.ProcessorCore{ID: i + *corePivot}
		for j := 0; j < numThreadsPerCore; j++ {
			processorCore.LogicalProcessors = append(processorCore.LogicalProcessors, j+*threadPivot)
		}
		*threadPivot += numThreadsPerCore
		node.Cores = append(node.Cores, processorCore)
	}
	*corePivot += numCores

	return node
}

func TestNRTBuilder_buildZoneScopeFields(t *testing.T) {
	type fields struct {
		reserved           v1.ResourceList
		reservedCPUs       int
		topologyInfo       *topology.Info
		systemReservedCPUs cpuset.CPUSet
	}
	type args struct {
		nrt *topologyapi.NodeResourceTopology
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *topologyapi.NodeResourceTopology
	}{
		{
			name: "",
			fields: fields{
				reserved: map[v1.ResourceName]resource.Quantity{
					v1.ResourceCPU: resource.MustParse("2"),
				},
				reservedCPUs:       2,
				topologyInfo:       buildTopologyInfo(2, 4, 2),
				systemReservedCPUs: cpuset.NewCPUSet(0, 9),
			},
			args: args{
				nrt: &topologyapi.NodeResourceTopology{},
			},
			want: &topologyapi.NodeResourceTopology{
				Zones: topologyapi.ZoneList{
					{
						Name: "node0",
						Type: topologyapi.ZoneTypeNode,
						Resources: &topologyapi.ResourceInfo{
							Capacity: map[v1.ResourceName]resource.Quantity{
								v1.ResourceCPU: resource.MustParse("8"),
							},
							Allocatable: map[v1.ResourceName]resource.Quantity{
								v1.ResourceCPU: resource.MustParse("7"),
							},
							ReservedCPUNums: 1,
						},
					},
					{
						Name: "node1",
						Type: topologyapi.ZoneTypeNode,
						Resources: &topologyapi.ResourceInfo{
							Capacity: map[v1.ResourceName]resource.Quantity{
								v1.ResourceCPU: resource.MustParse("8"),
							},
							Allocatable: map[v1.ResourceName]resource.Quantity{
								v1.ResourceCPU: resource.MustParse("7"),
							},
							ReservedCPUNums: 1,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &NRTBuilder{
				reserved:           tt.fields.reserved,
				reservedCPUs:       tt.fields.reservedCPUs,
				topologyInfo:       tt.fields.topologyInfo,
				systemReservedCPUs: tt.fields.systemReservedCPUs,
			}
			b.buildZoneScopeFields(tt.args.nrt)

			for _, z1 := range tt.args.nrt.Zones {
				var found bool
				for _, z2 := range tt.want.Zones {
					if z1.Name == z2.Name {
						found = true
						allocatableCPU1 := z1.Resources.Allocatable[v1.ResourceCPU]
						allocatableCPU2 := z2.Resources.Allocatable[v1.ResourceCPU]
						if !allocatableCPU1.Equal(allocatableCPU2) {
							t.Errorf("allocatableCPU not equal, %s, %s", allocatableCPU1.String(), allocatableCPU2.String())
						}
						capacityCPU1 := z1.Resources.Capacity[v1.ResourceCPU]
						capacityCPU2 := z2.Resources.Capacity[v1.ResourceCPU]
						if !capacityCPU1.Equal(capacityCPU2) {
							t.Errorf("capacityCPU not equal, %s, %s", capacityCPU1.String(), capacityCPU2.String())
						}

						if z1.Resources.ReservedCPUNums != z2.Resources.ReservedCPUNums {
							t.Errorf("ReservedCPUNums not equal, %d, %d", z1.Resources.ReservedCPUNums, z2.Resources.ReservedCPUNums)
						}
					}
				}
				if !found {
					t.Errorf("expect found node %s", z1.Name)
				}
			}

		})
	}
}
