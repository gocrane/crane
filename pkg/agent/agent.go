package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/jaypipes/ghw"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/yaml"
	quotav1 "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/apiserver/pkg/server/routes"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
	kubeletconfiginternal "k8s.io/kubernetes/pkg/kubelet/apis/config"
	kubeletcpumanager "k8s.io/kubernetes/pkg/kubelet/cm/cpumanager"
	"k8s.io/kubernetes/pkg/kubelet/stats/pidlimit"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/api/pkg/generated/informers/externalversions/ensurance/v1alpha1"
	predictionv1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	topologyapi "github.com/gocrane/api/topology/v1alpha1"

	"github.com/gocrane/crane/pkg/ensurance/analyzer"
	"github.com/gocrane/crane/pkg/ensurance/cm"
	"github.com/gocrane/crane/pkg/ensurance/collector"
	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/resource"
	"github.com/gocrane/crane/pkg/topology"
)

type Agent struct {
	ctx           context.Context
	name          string
	nodeName      string
	kubeClient    kubernetes.Interface
	craneClient   craneclientset.Interface
	managers      []manager.Manager
	kubeletConfig *kubeletconfiginternal.KubeletConfiguration
}

func NewAgent(ctx context.Context,
	nodeName, runtimeEndpoint, cgroupDriver, sysPath string,
	kubeClient kubernetes.Interface,
	craneClient craneclientset.Interface,
	kubeletConfig *kubeletconfiginternal.KubeletConfiguration,
	podInformer coreinformers.PodInformer,
	nodeInformer coreinformers.NodeInformer,
	nodeQOSInformer v1alpha1.NodeQOSInformer,
	podQOSInformer v1alpha1.PodQOSInformer,
	actionInformer v1alpha1.AvoidanceActionInformer,
	tspInformer predictionv1.TimeSeriesPredictionInformer,
	nodeResourceReserved map[string]string,
	ifaces []string,
	healthCheck *metrics.HealthCheck,
	CollectInterval time.Duration,
	executeExcess string,
) (*Agent, error) {
	var managers []manager.Manager
	var noticeCh = make(chan executor.AvoidanceExecutor)
	agent := &Agent{
		ctx:           ctx,
		name:          getAgentName(nodeName),
		nodeName:      nodeName,
		kubeClient:    kubeClient,
		craneClient:   craneClient,
		kubeletConfig: kubeletConfig,
	}

	utilruntime.Must(ensuranceapi.AddToScheme(scheme.Scheme))
	utilruntime.Must(topologyapi.AddToScheme(scheme.Scheme))
	cadvisorManager := cadvisor.NewCadvisorManager(cgroupDriver)
	exclusiveCPUSet := cm.DefaultExclusiveCPUSet
	if craneCpuSetManager := utilfeature.DefaultFeatureGate.Enabled(features.CraneCpuSetManager); craneCpuSetManager {
		cpuManager := cm.NewAdvancedCpuManager(podInformer, runtimeEndpoint, cadvisorManager)
		exclusiveCPUSet = cpuManager.GetExclusiveCpu
		managers = appendManagerIfNotNil(managers, cpuManager)
	}
	stateCollector := collector.NewStateCollector(nodeName, nodeQOSInformer.Lister(), podInformer.Lister(), nodeInformer.Lister(), ifaces, healthCheck, CollectInterval, exclusiveCPUSet, cadvisorManager)
	managers = appendManagerIfNotNil(managers, stateCollector)
	analyzerManager := analyzer.NewAnomalyAnalyzer(kubeClient, nodeName, podInformer, nodeInformer, nodeQOSInformer, podQOSInformer, actionInformer, stateCollector.AnalyzerChann, noticeCh)
	managers = appendManagerIfNotNil(managers, analyzerManager)
	avoidanceManager := executor.NewActionExecutor(kubeClient, nodeName, podInformer, nodeInformer, noticeCh, runtimeEndpoint, stateCollector.State, executeExcess)
	managers = appendManagerIfNotNil(managers, avoidanceManager)

	if nodeResource := utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource); nodeResource {
		tspName := agent.CreateNodeResourceTsp()
		nodeResourceManager, err := resource.NewNodeResourceManager(kubeClient, nodeName, nodeResourceReserved, tspName, nodeInformer, tspInformer, stateCollector.NodeResourceChann)
		if err != nil {
			return agent, err
		}
		managers = appendManagerIfNotNil(managers, nodeResourceManager)
	}

	if podResource := utilfeature.DefaultFeatureGate.Enabled(features.CranePodResource); podResource {
		podResourceManager := resource.NewPodResourceManager(kubeClient, nodeName, podInformer, runtimeEndpoint, stateCollector.PodResourceChann, stateCollector.GetCadvisorManager())
		managers = appendManagerIfNotNil(managers, podResourceManager)
	}

	_, err := agent.CreateNodeResourceTopology(sysPath)
	if err != nil {
		return agent, err
	}

	agent.managers = managers

	return agent, nil
}

func (a *Agent) Run(healthCheck *metrics.HealthCheck, enableProfiling bool, bindAddr string) {
	klog.Infof("Crane agent %s is starting", a.name)

	for _, m := range a.managers {
		m.Run(a.ctx.Done())
	}

	healthCheck.StartMonitoring()

	go func() {
		pathRecorderMux := mux.NewPathRecorderMux("crane-agent")
		defaultMetricsHandler := legacyregistry.Handler().ServeHTTP
		pathRecorderMux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
			defaultMetricsHandler(w, req)
		})

		pathRecorderMux.HandleFunc("/health-check", healthCheck.ServeHTTP)
		if enableProfiling {
			routes.Profiling{}.Install(pathRecorderMux)
		}
		err := http.ListenAndServe(bindAddr, pathRecorderMux)
		klog.Fatalf("Failed to start metrics: %v", err)
	}()

	<-a.ctx.Done()
}

func getAgentName(nodeName string) string {
	return nodeName + "." + string(uuid.NewUUID())
}

func (a *Agent) CreateNodeResourceTsp() string {
	foundTsp := true
	tsp, err := a.craneClient.PredictionV1alpha1().TimeSeriesPredictions(resource.TspNamespace).Get(context.TODO(), a.GenerateNodeResourceTspName(), metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to get noderesource tsp : %v", err)
			return ""
		}
		foundTsp = false
	}
	config, err := a.kubeClient.CoreV1().ConfigMaps(resource.TspNamespace).Get(context.TODO(), "noderesource-tsp-template", metav1.GetOptions{})

	if err != nil {
		klog.Errorf("Failed to get noderesource tsp configmap : %v", err)
	}

	if config == nil {
		klog.Errorf("Can't get noderesource tsp configmap noderesource-tsp-template")
	}

	n, err := a.kubeClient.CoreV1().Nodes().Get(context.TODO(), a.nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get node : %v", err)
		return ""
	}

	spec := predictionapi.TimeSeriesPredictionSpec{}
	tpl, err := template.New("").Parse(config.Data["spec"])
	if err != nil {
		klog.Errorf("Failed to convert spec template : %v", err)
		return ""
	}
	var buf bytes.Buffer
	//The k8s object is converted here to a json object in order to use lowercase letters in the template to take the node field,
	//just like {{ .metadata.name }}
	raw, _ := json.Marshal(n)
	var data interface{}
	_ = json.Unmarshal(raw, &data)

	err = tpl.Execute(&buf, data)
	if err != nil {
		klog.Errorf("Failed to convert spec template : %v", err)
		return ""
	}
	err = yaml.Unmarshal(buf.Bytes(), &spec)
	if err != nil {
		klog.Errorf("Failed to convert spec template : %v", err)
		return ""
	}

	gvk, _ := apiutil.GVKForObject(n, scheme.Scheme)
	spec.TargetRef = corev1.ObjectReference{
		Kind:       gvk.Kind,
		APIVersion: gvk.GroupVersion().String(),
		Name:       a.nodeName,
	}

	if foundTsp {
		klog.V(4).Infof("Discover the presence of old noderesource tsp and try to contrast the changes: %s", a.GenerateNodeResourceTspName())
		if reflect.DeepEqual(tsp.Spec, spec) {
			return a.GenerateNodeResourceTspName()
		}
		klog.V(4).Infof("Discover the presence of old noderesource tsp and the Tsp rules have been changed: %s", a.GenerateNodeResourceTspName())
		tsp.Spec = spec
		_, err := a.craneClient.PredictionV1alpha1().TimeSeriesPredictions(tsp.Namespace).Update(context.TODO(), tsp, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update noderesource tsp %s : %v", a.GenerateNodeResourceTspName(), err)
			return ""
		}
		klog.V(4).Infof("The noderesource tsp is updated successfully: %s", a.GenerateNodeResourceTspName())
	} else {
		klog.V(4).Infof("The noderesource tsp does not exist, try to create a new one: %s", a.GenerateNodeResourceTspName())
		tsp = &predictionapi.TimeSeriesPrediction{}
		tsp.Name = a.GenerateNodeResourceTspName()
		tsp.Namespace = resource.TspNamespace
		tsp.Spec = spec
		_ = controllerutil.SetControllerReference(n, tsp, scheme.Scheme)
		_, err = a.craneClient.PredictionV1alpha1().TimeSeriesPredictions(tsp.Namespace).Create(context.TODO(), tsp, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("Failed to create noderesource tsp %s : %v", a.GenerateNodeResourceTspName(), err)
			return ""
		}
		klog.V(4).Infof("The noderesource tsp is created successfully: %s", a.GenerateNodeResourceTspName())
	}

	return a.GenerateNodeResourceTspName()
}

func (a *Agent) CreateNodeResourceTopology(sysPath string) (*topologyapi.NodeResourceTopology, error) {
	topo, err := ghw.Topology(ghw.WithPathOverrides(ghw.PathOverrides{
		"/sys": sysPath,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to detect topology info by GHW: %v", err)
	}
	klog.InfoS("Get topology info from GHW finished", "info", topo.String())

	exist := true
	nrt, err := a.craneClient.TopologyV1alpha1().NodeResourceTopologies().Get(context.TODO(), a.nodeName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to get node resource topology: %v", err)
			return nil, err
		}
		exist = false
	}

	node, err := a.kubeClient.CoreV1().Nodes().Get(context.TODO(), a.nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return nil, err
	}

	kubeReserved, err := parseResourceList(a.kubeletConfig.KubeReserved)
	if err != nil {
		return nil, err
	}
	systemReserved, err := parseResourceList(a.kubeletConfig.SystemReserved)
	if err != nil {
		return nil, err
	}
	reserved := quotav1.Add(kubeReserved, systemReserved)

	cpuManagerPolicy := topologyapi.CPUManagerPolicyStatic
	// If kubelet cpumanager policy is static, we should set the agent cpu manager policy to none.
	if a.kubeletConfig.CPUManagerPolicy == string(kubeletcpumanager.PolicyStatic) {
		cpuManagerPolicy = topologyapi.CPUManagerPolicyNone
	}

	nrtBuilder := topology.NewNRTBuilder()
	nrtBuilder.WithNode(node)
	nrtBuilder.WithReserved(reserved)
	nrtBuilder.WithTopologyInfo(topo)
	nrtBuilder.WithCPUManagerPolicy(cpuManagerPolicy)
	newNrt := nrtBuilder.Build()
	_ = controllerutil.SetControllerReference(node, newNrt, scheme.Scheme)

	if exist {
		newNrt.TypeMeta = nrt.TypeMeta
		newNrt.ObjectMeta = nrt.ObjectMeta
		oldData, err := json.Marshal(nrt)
		if err != nil {
			return nil, err
		}
		newData, err := json.Marshal(newNrt)
		if err != nil {
			return nil, err
		}
		patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
		if err != nil {
			return nil, fmt.Errorf("failed to create merge patch: %v", err)
		}
		nrt, err = a.craneClient.TopologyV1alpha1().NodeResourceTopologies().Patch(context.TODO(), a.nodeName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			klog.Errorf("Failed to update node resource topology %s: %v", a.nodeName, err)
			return nil, err
		}
		return nrt, nil
	} else {
		nrt, err = a.craneClient.TopologyV1alpha1().NodeResourceTopologies().Create(context.TODO(), newNrt, metav1.CreateOptions{})
		if err != nil {
			klog.Errorf("Failed to create node resource topology %s: %v", a.nodeName, err)
			return nil, err
		}
		return nrt, nil
	}
}

func (a *Agent) DeleteNodeResourceTsp() error {
	err := a.craneClient.PredictionV1alpha1().TimeSeriesPredictions(resource.TspNamespace).Delete(context.TODO(), a.GenerateNodeResourceTspName(), metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) GenerateNodeResourceTspName() string {
	return fmt.Sprintf("noderesource-%s", a.nodeName)
}

func appendManagerIfNotNil(managers []manager.Manager, m manager.Manager) []manager.Manager {
	if m != nil {
		return append(managers, m)
	}
	return managers
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
