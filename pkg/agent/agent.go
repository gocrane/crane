package agent

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	craneinformers "github.com/gocrane/api/pkg/generated/informers/externalversions"

	"github.com/gocrane/crane/cmd/crane-agent/app/options"
	"github.com/gocrane/crane/pkg/ensurance/analyzer"
	"github.com/gocrane/crane/pkg/ensurance/avoidance"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/ensurance/statestore"
)

const (
	nodeNameField      = "metadata.name"
	specNodeNameField  = "spec.nodeName"
	informerSyncPeriod = time.Minute
	DefaultWorkers     = 2
)

type Agent struct {
	ctx                  context.Context
	name                 string
	kubeClient           kubernetes.Interface
	craneClient          craneclientset.Interface
	podInformerFactory   informers.SharedInformerFactory
	nodeInformerFactory  informers.SharedInformerFactory
	craneInformerFactory craneinformers.SharedInformerFactory
	recorder             record.EventRecorder

	nodeQOSController *statestore.Controller
	managers          []manager.Manager
}

func NewAgent(ctx context.Context, opts *options.Options) (*Agent, error) {
	nodeName, _ := os.Hostname()

	if os.Getenv("NODE_NAME") != "" {
		nodeName = os.Getenv("NODE_NAME")
	}

	if len(opts.HostnameOverride) != 0 {
		nodeName = opts.HostnameOverride
	}

	name := nodeName + "_" + string(uuid.NewUUID())

	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Error(err, "Failed to get InClusterConfig.")
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err, "Failed to new kubernetes client.")
		return nil, err
	}

	craneClient, err := craneclientset.NewForConfig(config)
	if err != nil {
		klog.Error(err, "Failed to new crane client.")
		return nil, err
	}

	podInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, informerSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector(specNodeNameField, nodeName).String()
		}),
	)

	nodeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, informerSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector(nodeNameField, nodeName).String()
		}),
	)

	craneInformerFactory := craneinformers.NewSharedInformerFactory(craneClient, informerSyncPeriod)

	utilruntime.Must(ensuranceapi.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "crane-agent"})

	nodeQOSController := statestore.NewController(kubeClient, craneClient, craneInformerFactory, craneInformerFactory.Ensurance().V1alpha1().NodeQOSEnsurancePolicies(), recorder)

	var managers []manager.Manager

	var noticeCh = make(chan executor.AvoidanceExecutor)

	stateStoreManager := statestore.NewStateStoreManager(craneInformerFactory.Ensurance().V1alpha1().NodeQOSEnsurancePolicies().Informer())
	managers = append(managers, stateStoreManager)

	// init analyzer manager
	analyzerManager := analyzer.NewAnalyzerManager(nodeName, podInformerFactory.Core().V1().Pods(), nodeInformerFactory.Core().V1().Nodes(), craneInformerFactory, stateStoreManager, recorder, noticeCh)
	managers = append(managers, analyzerManager)

	// init avoidance manager
	avoidanceManager := avoidance.NewAvoidanceManager(kubeClient, nodeName, podInformerFactory.Core().V1().Pods(), nodeInformerFactory.Core().V1().Nodes(), noticeCh)
	managers = append(managers, avoidanceManager)

	return &Agent{
		ctx:                  ctx,
		name:                 name,
		kubeClient:           kubeClient,
		craneClient:          craneClient,
		podInformerFactory:   podInformerFactory,
		nodeInformerFactory:  nodeInformerFactory,
		craneInformerFactory: craneInformerFactory,
		recorder:             recorder,
		nodeQOSController:    nodeQOSController,
		managers:             managers,
	}, nil
}

func (a *Agent) Run() {
	klog.Infof("Crane agent %s is starting", a.name)

	a.podInformerFactory.Start(a.ctx.Done())
	a.nodeInformerFactory.Start(a.ctx.Done())
	a.craneInformerFactory.Start(a.ctx.Done())

	a.podInformerFactory.WaitForCacheSync(a.ctx.Done())
	a.nodeInformerFactory.WaitForCacheSync(a.ctx.Done())
	a.craneInformerFactory.WaitForCacheSync(a.ctx.Done())

	go a.nodeQOSController.Run(DefaultWorkers, a.ctx.Done())

	for _, m := range a.managers {
		m.Run(a.ctx.Done())
	}

	<-a.ctx.Done()

}
