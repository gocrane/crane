package agent

import (
	"context"
	"fmt"
	"github.com/gocrane/crane/pkg/ensurance/cm"
	"github.com/gocrane/crane/pkg/noderesource"
	"github.com/gocrane/crane/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	"github.com/gocrane/crane/pkg/metrics"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/apiserver/pkg/server/routes"
	"k8s.io/apimachinery/pkg/util/yaml"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"

	v1alpha12 "github.com/gocrane/api/prediction/v1alpha1"
	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	predictionv1alpha1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	"github.com/gocrane/api/pkg/generated/informers/externalversions/ensurance/v1alpha1"
	"github.com/gocrane/crane/cmd/crane-agent/app/options"
	"github.com/gocrane/crane/pkg/ensurance/analyzer"
	"github.com/gocrane/crane/pkg/ensurance/collector"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/ensurance/manager"
)

type Agent struct {
	ctx         context.Context
	name        string
	kubeClient  kubernetes.Interface
	craneClient craneclientset.Interface
	managers    []manager.Manager
	host         string
}

func NewAgent(ctx context.Context,
	nodeName, runtimeEndpoint string,
	kubeClient *kubernetes.Clientset,
	craneClient *craneclientset.Clientset,
	podInformer coreinformers.PodInformer,
	nodeInformer coreinformers.NodeInformer,
	nepInformer v1alpha1.NodeQOSEnsurancePolicyInformer,
	actionInformer v1alpha1.AvoidanceActionInformer,
	timeSeriesPredictionInformer predictionv1alpha1.TimeSeriesPredictionInformer,
	nodeResourceOptions options.NodeResourceOptions,
	ifaces []string,
	healthCheck *metrics.HealthCheck,
	CollectInterval time.Duration,
	useBt bool,
) (*Agent, error) {
	var managers []manager.Manager
	var noticeCh = make(chan executor.AvoidanceExecutor)
	agent := &Agent{
		ctx:          ctx,
		name:         getAgentName(nodeName),
		host:         nodeName,
		kubeClient:   kubeClient,
		craneClient:  craneClient,
	}
	cadvisorManager, err := utils.NewCadvisorManager()
	if err != nil {
		return nil, err
	}
	utilruntime.Must(ensuranceapi.AddToScheme(scheme.Scheme))

	stateCollector := collector.NewStateCollector(nodeName, nepInformer.Lister(), podInformer.Lister(), nodeInformer.Lister(), ifaces, healthCheck, CollectInterval)
	managers = append(managers, stateCollector)
	analyzerManager := analyzer.NewAnormalyAnalyzer(kubeClient, nodeName, podInformer, nodeInformer, nepInformer, actionInformer, stateCollector.StateChann, noticeCh)
	managers = append(managers, analyzerManager)
	avoidanceManager := executor.NewActionExecutor(kubeClient, nodeName, podInformer, nodeInformer, noticeCh, runtimeEndpoint)
	managers = append(managers, avoidanceManager)
	cpuManager := cm.NewAdvancedCpuManager(kubeClient, nodeName, podInformer, nodeInformer, runtimeEndpoint, cadvisorManager)
	managers = append(managers, cpuManager)
	if nodeResourceOptions.Enabled {
		nodeResourceManager := noderesource.NewNodeResource(nodeName, kubeClient, craneClient, nodeInformer, timeSeriesPredictionInformer, nodeResourceOptions.ReserveCpuPercentStr, nodeResourceOptions.ReserveMemoryPercentStr, nodeResourceOptions.CollectorNames, utils.NewCpuStateProvider(cadvisorManager, podInformer.Lister(), useBt, cpuManager.GetExclusiveCpu), agent.CreateNodeResourceTsp())
		managers = append(managers, nodeResourceManager)
	}

	return &Agent{
		ctx:         ctx,
		name:        getAgentName(nodeName),
		kubeClient:  kubeClient,
		craneClient: craneClient,
		managers:    managers,
	}, nil
}

func (a *Agent) Run(healthCheck *metrics.HealthCheck, opts *options.Options) {
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
		if opts.EnableProfiling {
			routes.Profiling{}.Install(pathRecorderMux)
		}
		err := http.ListenAndServe(opts.BindAddr, pathRecorderMux)
		klog.Fatalf("Failed to start metrics: %v", err)
	}()

	<-a.ctx.Done()
}

func getAgentName(nodeName string) string {
	return nodeName + "." + string(uuid.NewUUID())
}

func (a *Agent) CreateNodeResourceTsp() string {
	tsp, err := a.craneClient.PredictionV1alpha1().TimeSeriesPredictions("default").Get(context.TODO(), a.GenerateNodeResourceTspName(), metav1.GetOptions{})
	if err == nil {
		klog.V(4).Infof("Found old tsp %s in namespace default", a.GenerateNodeResourceTspName())
		err := a.DeleteNodeResourceTsp()
		if err != nil {
			klog.Errorf("Delete old tsp %s with error: %v", a.GenerateNodeResourceTspName(), err)
			return a.GenerateNodeResourceTspName()
		}
	}
	config, err := a.kubeClient.CoreV1().ConfigMaps("default").Get(context.TODO(), "noderesource-tsp-template", metav1.GetOptions{})

	if err != nil {
		klog.Exitf("Get noderesource tsp configmap noderesource-tsp-template with error: %v", err)
	}

	if config == nil {
		klog.Exitf("Can't get noderesource tsp configmap noderesource-tsp-template")
	}

	spec := v1alpha12.TimeSeriesPredictionSpec{}
	err = yaml.Unmarshal([]byte(strings.Replace(config.Data["spec"], "{{nodename}}", a.host, -1)), &spec)
	if err != nil {
		klog.Exitf("Convert spec template error: %v", err)
	}

	n, _ := a.kubeClient.CoreV1().Nodes().Get(context.TODO(), a.host, metav1.GetOptions{})

	tsp = &v1alpha12.TimeSeriesPrediction{}

	tsp.Name = a.GenerateNodeResourceTspName()
	tsp.Namespace = "default"
	gvk, _:= apiutil.GVKForObject(n, scheme.Scheme)
	spec.TargetRef = v1.ObjectReference{
		Kind:       gvk.Kind,
		APIVersion: gvk.GroupVersion().String(),
		Name:       a.host,
	}
	tsp.Spec = spec
	_ = controllerutil.SetControllerReference(n, tsp, scheme.Scheme)
	_, err = a.craneClient.PredictionV1alpha1().TimeSeriesPredictions("default").Create(context.TODO(), tsp, metav1.CreateOptions{})
	if err != nil {
		klog.Exitf("Create noderesource tsp %s with error: %v", a.GenerateNodeResourceTspName(), err)
	}
	return a.GenerateNodeResourceTspName()
}

func (a *Agent) DeleteNodeResourceTsp() error {
	err := a.craneClient.PredictionV1alpha1().TimeSeriesPredictions("default").Delete(context.TODO(), a.GenerateNodeResourceTspName(), metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) GenerateNodeResourceTspName() string {
	return fmt.Sprintf("noderesource-%s", a.name)
}
