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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/apiserver/pkg/server/routes"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/api/pkg/generated/informers/externalversions/ensurance/v1alpha1"
	predictionv1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	topologyinformer "github.com/gocrane/api/pkg/generated/informers/externalversions/topology/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	topologyapi "github.com/gocrane/api/topology/v1alpha1"

	"github.com/gocrane/crane/pkg/ensurance/analyzer"
	"github.com/gocrane/crane/pkg/ensurance/cm/cpumanager"
	"github.com/gocrane/crane/pkg/ensurance/collector"
	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"github.com/gocrane/crane/pkg/ensurance/collector/noderesourcetopology"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/ensurance/runtime"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/resource"
	"github.com/gocrane/crane/pkg/utils"
)

type Agent struct {
	ctx         context.Context
	name        string
	nodeName    string
	kubeClient  kubernetes.Interface
	craneClient craneclientset.Interface
	managers    []manager.Manager
}

func NewAgent(ctx context.Context,
	nodeName, runtimeEndpoint, cgroupDriver, sysPath, kubeletRootPath string,
	kubeClient kubernetes.Interface,
	craneClient craneclientset.Interface,
	podInformer coreinformers.PodInformer,
	nodeInformer coreinformers.NodeInformer,
	nodeQOSInformer v1alpha1.NodeQOSInformer,
	podQOSInformer v1alpha1.PodQOSInformer,
	actionInformer v1alpha1.AvoidanceActionInformer,
	tspInformer predictionv1.TimeSeriesPredictionInformer,
	nrtInformer topologyinformer.NodeResourceTopologyInformer,
	nodeResourceReserved map[string]string,
	ifaces []string,
	healthCheck *metrics.HealthCheck,
	collectInterval time.Duration,
	executeExcess string,
	cpuManagerReconcilePeriod time.Duration,
	defaultCPUPolicy string,
) (*Agent, error) {
	var managers []manager.Manager
	var noticeCh = make(chan executor.AvoidanceExecutor)
	agent := &Agent{
		ctx:         ctx,
		name:        getAgentName(nodeName),
		nodeName:    nodeName,
		kubeClient:  kubeClient,
		craneClient: craneClient,
	}

	runtimeService, err := runtime.GetCRIRuntimeService(runtimeEndpoint)
	if err != nil {
		return nil, err
	}

	utilruntime.Must(ensuranceapi.AddToScheme(scheme.Scheme))
	utilruntime.Must(topologyapi.AddToScheme(scheme.Scheme))
	cadvisorManager := cadvisor.NewCadvisorManager(cgroupDriver)
	exclusiveCPUSet := cpumanager.DefaultExclusiveCPUSet
	if utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResourceTopology) {
		if err := agent.CreateNodeResourceTopology(sysPath); err != nil {
			return nil, err
		}
		if utilfeature.DefaultFeatureGate.Enabled(features.CraneCPUManager) {
			cpuManager, err := cpumanager.NewCPUManager(nodeName, defaultCPUPolicy, cpuManagerReconcilePeriod, cadvisorManager, runtimeService, kubeletRootPath, podInformer, nrtInformer)
			if err != nil {
				return nil, fmt.Errorf("failed to new cpumanager: %v", err)
			}
			exclusiveCPUSet = cpuManager.GetExclusiveCPUSet
			managers = appendManagerIfNotNil(managers, cpuManager)
		}
	}

	stateCollector := collector.NewStateCollector(nodeName, sysPath, kubeClient, craneClient, nodeQOSInformer.Lister(), nrtInformer.Lister(), podInformer.Lister(), nodeInformer.Lister(), ifaces, healthCheck, collectInterval, exclusiveCPUSet, cadvisorManager)
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

func (a *Agent) CreateNodeResourceTopology(sysPath string) error {
	nrt, err := a.craneClient.TopologyV1alpha1().NodeResourceTopologies().Get(context.TODO(), a.nodeName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to get node resource topology: %v", err)
			return err
		}
		nrt = nil
	}

	node, err := a.kubeClient.CoreV1().Nodes().Get(context.TODO(), a.nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return err
	}

	kubeletConfig, err := utils.GetKubeletConfig(context.TODO(), a.kubeClient, a.nodeName)
	if err != nil {
		klog.Errorf("Failed to get kubelet config: %v", err)
		return err
	}

	newNrt, err := noderesourcetopology.BuildNodeResourceTopology(sysPath, kubeletConfig, node)
	if err != nil {
		klog.Errorf("Failed to build node resource topology: %v", err)
		return err
	}

	if err = noderesourcetopology.CreateOrUpdateNodeResourceTopology(a.craneClient, nrt, newNrt); err != nil {
		klog.Errorf("Failed to create or update node resource topology: %v", err)
		return err
	}
	return nil
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
