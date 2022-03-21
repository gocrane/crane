package agent

import (
	"context"
	"fmt"
	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"net/http"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
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
	v1alpha12 "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/cmd/crane-agent/app/options"
	"github.com/gocrane/crane/pkg/ensurance/analyzer"
	"github.com/gocrane/crane/pkg/ensurance/cm"
	"github.com/gocrane/crane/pkg/ensurance/collector"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/resource"
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
	nodeName, runtimeEndpoint string,
	kubeClient *kubernetes.Clientset,
	craneClient *craneclientset.Clientset,
	podInformer coreinformers.PodInformer,
	nodeInformer coreinformers.NodeInformer,
	nepInformer v1alpha1.NodeQOSEnsurancePolicyInformer,
	actionInformer v1alpha1.AvoidanceActionInformer,
	tspInformer predictionv1.TimeSeriesPredictionInformer,
	nodeResourceOptions options.NodeResourceOptions,
	ifaces []string,
	healthCheck *metrics.HealthCheck,
	CollectInterval time.Duration,
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

	utilruntime.Must(ensuranceapi.AddToScheme(scheme.Scheme))
	cadvisorManager := cadvisor.NewCadvisorManager()
	exclusiveCPUSet := cm.DefaultExclusiveCPUSet
	if craneCpuSetManager := utilfeature.DefaultFeatureGate.Enabled(features.CraneCpuSetManager); craneCpuSetManager {
		cpuManager := cm.NewAdvancedCpuManager(podInformer, runtimeEndpoint, cadvisorManager)
		exclusiveCPUSet = cpuManager.GetExclusiveCpu
		managers = appendManagerIfNotNil(managers, cpuManager)
	}
	stateCollector := collector.NewStateCollector(nodeName, nepInformer.Lister(), podInformer.Lister(), nodeInformer.Lister(), ifaces, healthCheck, CollectInterval, exclusiveCPUSet, cadvisorManager)
	managers = appendManagerIfNotNil(managers, stateCollector)
	analyzerManager := analyzer.NewAnormalyAnalyzer(kubeClient, nodeName, podInformer, nodeInformer, nepInformer, actionInformer, stateCollector.AnalyzerChann, noticeCh)
	managers = appendManagerIfNotNil(managers, analyzerManager)
	avoidanceManager := executor.NewActionExecutor(kubeClient, nodeName, podInformer, nodeInformer, noticeCh, runtimeEndpoint)
	managers = appendManagerIfNotNil(managers, avoidanceManager)

	if nodeResource := utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource); nodeResource {
		nodeResourceManager := resource.NewNodeResourceManager(kubeClient, nodeName, nodeResourceOptions.ReserveCpuPercentStr, nodeResourceOptions.ReserveMemoryPercentStr, agent.CreateNodeResourceTsp(), nodeInformer, tspInformer, stateCollector.NodeResourceChann)
		managers = append(managers, nodeResourceManager)
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
	tsp, err := a.craneClient.PredictionV1alpha1().TimeSeriesPredictions(resource.TspNamespace).Get(context.TODO(), a.GenerateNodeResourceTspName(), metav1.GetOptions{})
	if err == nil {
		klog.V(4).Infof("Found old tsp %s in namespace %s", a.GenerateNodeResourceTspName(), resource.TspNamespace)
		err := a.DeleteNodeResourceTsp()
		if err != nil {
			klog.Errorf("Delete old tsp %s with error: %v", a.GenerateNodeResourceTspName(), err)
			return a.GenerateNodeResourceTspName()
		}
	}
	config, err := a.kubeClient.CoreV1().ConfigMaps(resource.TspNamespace).Get(context.TODO(), "noderesource-tsp-template", metav1.GetOptions{})

	if err != nil {
		klog.Exitf("Get noderesource tsp configmap noderesource-tsp-template with error: %v", err)
	}

	if config == nil {
		klog.Exitf("Can't get noderesource tsp configmap noderesource-tsp-template")
	}

	spec := v1alpha12.TimeSeriesPredictionSpec{}
	err = yaml.Unmarshal([]byte(strings.Replace(config.Data["spec"], "{{nodename}}", a.nodeName, -1)), &spec)
	if err != nil {
		klog.Exitf("Convert spec template error: %v", err)
	}

	n, err := a.kubeClient.CoreV1().Nodes().Get(context.TODO(), a.nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Exitf("Get node error: %v", err)
	}

	tsp = &v1alpha12.TimeSeriesPrediction{}

	tsp.Name = a.GenerateNodeResourceTspName()
	tsp.Namespace = resource.TspNamespace
	gvk, _ := apiutil.GVKForObject(n, scheme.Scheme)
	spec.TargetRef = v1.ObjectReference{
		Kind:       gvk.Kind,
		APIVersion: gvk.GroupVersion().String(),
		Name:       a.nodeName,
	}
	tsp.Spec = spec
	_ = controllerutil.SetControllerReference(n, tsp, scheme.Scheme)
	_, err = a.craneClient.PredictionV1alpha1().TimeSeriesPredictions(tsp.Namespace).Create(context.TODO(), tsp, metav1.CreateOptions{})
	if err != nil {
		klog.Exitf("Create noderesource tsp %s with error: %v", a.GenerateNodeResourceTspName(), err)
	}
	return a.GenerateNodeResourceTspName()
}

func (a *Agent) DeleteNodeResourceTsp() error {
	err := a.craneClient.PredictionV1alpha1().TimeSeriesPredictions(resource.TspNamespace).Delete(context.TODO(), a.GenerateNodeResourceTspName(), metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) GenerateNodeResourceTspName() string {
	return fmt.Sprintf("noderesource-%s", a.name)
}

func appendManagerIfNotNil(managers []manager.Manager, m manager.Manager) []manager.Manager {
	if m != nil {
		return append(managers, m)
	}
	return managers
}
