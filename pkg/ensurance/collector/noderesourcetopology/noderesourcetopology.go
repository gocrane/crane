package noderesourcetopology

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/jaypipes/ghw"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	patchtypes "k8s.io/apimachinery/pkg/types"
	quotav1 "k8s.io/apiserver/pkg/quota/v1"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	kubeletconfiginternal "k8s.io/kubernetes/pkg/kubelet/apis/config"
	kubeletcpumanager "k8s.io/kubernetes/pkg/kubelet/cm/cpumanager"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/kubelet/stats/pidlimit"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	topologylisters "github.com/gocrane/api/pkg/generated/listers/topology/v1alpha1"
	topologyapi "github.com/gocrane/api/topology/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/topology"
	"github.com/gocrane/crane/pkg/utils"
)

type NodeResourceTopology struct {
	nodeName    string
	sysPath     string
	nrtLister   topologylisters.NodeResourceTopologyLister
	nodeLister  corelisters.NodeLister
	client      kubeclient.Interface
	craneClient craneclientset.Interface
}

func NewNodeResourceTopology(nodeName, sysPath string,
	nrtLister topologylisters.NodeResourceTopologyLister, nodeLister corelisters.NodeLister,
	client kubeclient.Interface, craneClient craneclientset.Interface,
) *NodeResourceTopology {
	return &NodeResourceTopology{
		nodeName:    nodeName,
		sysPath:     sysPath,
		nrtLister:   nrtLister,
		nodeLister:  nodeLister,
		client:      client,
		craneClient: craneClient,
	}
}

func (n *NodeResourceTopology) GetType() types.CollectType {
	return types.NodeResourceTopologyCollectorType
}

func (n *NodeResourceTopology) Collect() (map[string][]common.TimeSeries, error) {
	nrt, err := n.nrtLister.Get(n.nodeName)
	if err != nil {
		return nil, err
	}

	node, err := n.nodeLister.Get(n.nodeName)
	if err != nil {
		return nil, err
	}

	kubeletConfig, err := utils.GetKubeletConfig(context.TODO(), n.client, n.nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get config from kubelet endpoint: %v", err)
	}

	newNrt, err := BuildNodeResourceTopology(n.sysPath, kubeletConfig, node)
	if err != nil {
		return nil, fmt.Errorf("failed to build node resource topology: %v", err)
	}

	if err = CreateOrUpdateNodeResourceTopology(n.craneClient, nrt, newNrt); err != nil {
		return nil, fmt.Errorf("failed to create or update node resource topology: %v", err)
	}
	return nil, nil
}

func (n *NodeResourceTopology) Stop() error {
	return nil
}

func BuildNodeResourceTopology(sysPath string, kubeletConfig *kubeletconfiginternal.KubeletConfiguration,
	node *corev1.Node) (*topologyapi.NodeResourceTopology, error) {
	topo, err := ghw.Topology(ghw.WithPathOverrides(ghw.PathOverrides{
		"/sys": sysPath,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to detect topology info by GHW: %v", err)
	}

	reservedSystemCPUs, err := parseReservedSystemCPUs(kubeletConfig)
	if err != nil {
		return nil, err
	}
	kubeReserved, err := parseResourceList(kubeletConfig.KubeReserved)
	if err != nil {
		return nil, err
	}
	systemReserved, err := parseResourceList(kubeletConfig.SystemReserved)
	if err != nil {
		return nil, err
	}
	reserved := quotav1.Add(kubeReserved, systemReserved)

	cpuManagerPolicy := topologyapi.CPUManagerPolicyStatic
	// If kubelet cpumanager policy is static, we should set the agent cpu manager policy to none.
	if kubeletConfig.CPUManagerPolicy == string(kubeletcpumanager.PolicyStatic) {
		cpuManagerPolicy = topologyapi.CPUManagerPolicyNone
	}

	nrtBuilder := topology.NewNRTBuilder()
	nrtBuilder.WithNode(node)
	nrtBuilder.WithReserved(reserved)
	nrtBuilder.WithReservedCPUs(getNumReservedCPUs(reserved))
	nrtBuilder.WithTopologyInfo(topo)
	nrtBuilder.WithCPUManagerPolicy(cpuManagerPolicy)
	nrtBuilder.WithSystemReservedCPUs(reservedSystemCPUs)
	nrtBuilder.WithAttributes(map[string]string{
		topologyapi.ReservedSystemCPUsAttributes: kubeletConfig.ReservedSystemCPUs,
	})

	newNrt := nrtBuilder.Build()
	_ = controllerutil.SetControllerReference(node, newNrt, scheme.Scheme)
	return newNrt, nil
}

func CreateOrUpdateNodeResourceTopology(craneClient craneclientset.Interface, old, new *topologyapi.NodeResourceTopology) error {
	if old == nil {
		_, err := craneClient.TopologyV1alpha1().NodeResourceTopologies().Create(context.TODO(), new, metav1.CreateOptions{})
		return err
	}
	new.TypeMeta = old.TypeMeta
	new.ObjectMeta = old.ObjectMeta

	if equality.Semantic.DeepEqual(old, new) {
		return nil
	}

	oldData, err := json.Marshal(old)
	if err != nil {
		return err
	}
	newData, err := json.Marshal(new)
	if err != nil {
		return err
	}
	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return fmt.Errorf("failed to create merge patch: %v", err)
	}
	_, err = craneClient.TopologyV1alpha1().NodeResourceTopologies().Patch(context.TODO(), new.Name, patchtypes.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

// parseResourceList parses the given configuration map into an API
// ResourceList or returns an error.
func parseResourceList(m map[string]string) (corev1.ResourceList, error) {
	if len(m) == 0 {
		return nil, nil
	}
	rl := make(corev1.ResourceList)
	for k, v := range m {
		switch corev1.ResourceName(k) {
		// CPU, memory, local storage, and PID resources are supported.
		case corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage, pidlimit.PIDs:
			q, err := apiresource.ParseQuantity(v)
			if err != nil {
				return nil, err
			}
			if q.Sign() == -1 {
				return nil, fmt.Errorf("resource quantity for %q cannot be negative: %v", k, v)
			}
			rl[corev1.ResourceName(k)] = q
		default:
			return nil, fmt.Errorf("cannot reserve %q resource", k)
		}
	}
	return rl, nil
}

// getNumReservedCPUs will get the number of reserve cpus by reserved resource request.
func getNumReservedCPUs(nodeAllocatableReservation corev1.ResourceList) int {
	reservedCPUs, ok := nodeAllocatableReservation[corev1.ResourceCPU]
	if !ok || reservedCPUs.IsZero() {
		// The static policy cannot initialize without this information.
		return 0
	}

	reservedCPUsFloat := float64(reservedCPUs.MilliValue()) / 1000
	numReservedCPUs := int(math.Ceil(reservedCPUsFloat))
	return numReservedCPUs
}

// parseReservedSystemCPUs will parse kubelet ReservedSystemCPUs and overwrite the cpus in KubeReserved and SystemReserved
// copy code from: https://github.com/kubernetes/kubernetes/blob/master/cmd/kubelet/app/server.go#L671
func parseReservedSystemCPUs(kubeletConfig *kubeletconfiginternal.KubeletConfiguration) (cpuset.CPUSet, error) {
	reservedSystemCPUs, err := utils.GetReservedCPUs(kubeletConfig.ReservedSystemCPUs)
	if err != nil {
		return reservedSystemCPUs, fmt.Errorf("parse reserved cpus: %v", err)
	}
	if reservedSystemCPUs.Size() > 0 {
		// at cmd option validation phase it is tested either --system-reserved-cgroup or --kube-reserved-cgroup is specified, so overwrite should be ok
		klog.InfoS("Option --reserved-cpus is specified, it will overwrite the cpu setting in KubeReserved and SystemReserved",
			"kubeReservedCPUs", kubeletConfig.KubeReserved, "systemReservedCPUs", kubeletConfig.SystemReserved)
		if kubeletConfig.KubeReserved != nil {
			delete(kubeletConfig.KubeReserved, "cpu")
		}
		if kubeletConfig.SystemReserved == nil {
			kubeletConfig.SystemReserved = make(map[string]string)
		}
		kubeletConfig.SystemReserved["cpu"] = strconv.Itoa(reservedSystemCPUs.Size())
		klog.InfoS("After cpu setting is overwritten", "kubeReservedCPUs", kubeletConfig.KubeReserved, "systemReservedCPUs", kubeletConfig.SystemReserved)
	}
	return reservedSystemCPUs, nil
}
