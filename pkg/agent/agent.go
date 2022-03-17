package agent

import (
	"context"
	"net/http"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/apiserver/pkg/server/routes"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/api/pkg/generated/informers/externalversions/ensurance/v1alpha1"
	predictionv1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/ensurance/analyzer"
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
	ifaces []string,
	healthCheck *metrics.HealthCheck,
	CollectInterval time.Duration,
) (*Agent, error) {
	var managers []manager.Manager
	var noticeCh = make(chan executor.AvoidanceExecutor)

	utilruntime.Must(ensuranceapi.AddToScheme(scheme.Scheme))

	stateCollector := collector.NewStateCollector(nodeName, nepInformer.Lister(), podInformer.Lister(), nodeInformer.Lister(), ifaces, healthCheck, CollectInterval)
	managers = append(managers, stateCollector)
	analyzerManager := analyzer.NewAnormalyAnalyzer(kubeClient, nodeName, podInformer, nodeInformer, nepInformer, actionInformer, stateCollector.AnalyzerChann, noticeCh)
	managers = append(managers, analyzerManager)
	avoidanceManager := executor.NewActionExecutor(kubeClient, nodeName, podInformer, nodeInformer, noticeCh, runtimeEndpoint)
	managers = append(managers, avoidanceManager)

	if nodeResource := utilfeature.DefaultFeatureGate.Enabled(features.CraneNodeResource); nodeResource {
		nodeResourceManager := resource.NewNodeResourceManager(kubeClient, nodeName, podInformer, nodeInformer, tspInformer, runtimeEndpoint, stateCollector.NodeResourceChann)
		managers = append(managers, nodeResourceManager)
	}

	if podResource := utilfeature.DefaultFeatureGate.Enabled(features.CranePodResource); podResource {
		podResourceManager := resource.NewPodResourceManager(kubeClient, nodeName, podInformer, runtimeEndpoint, stateCollector.PodResourceChann, stateCollector.GetCadvisorManager())
		managers = append(managers, podResourceManager)
	}

	return &Agent{
		ctx:         ctx,
		name:        getAgentName(nodeName),
		kubeClient:  kubeClient,
		craneClient: craneClient,
		managers:    managers,
	}, nil
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
	return nodeName + "_" + string(uuid.NewUUID())
}
