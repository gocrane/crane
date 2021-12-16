package informer

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	ensuaranceset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/api/pkg/generated/informers/externalversions"
)

const (
	nodeNameField      = "metadata.name"
	specNodeNameField  = "spec.nodeName"
	statusPhaseField   = "status.phase"
	informerSyncPeriod = time.Minute
	defaultRetryTimes  = 3
)

// Context stores k8s client and factory,which generate the resource informers
type Context struct {
	nodeName string
	// stop channel
	stop <-chan struct{}
	// kubernetes client to communication with kubernetes api-server
	kubeClient clientset.Interface
	// ensurance client
	ensuranceClient ensuaranceset.Interface
	// kubernetes node resource factory
	nodeFactory informers.SharedInformerFactory
	// kubernetes node resource informer
	nodeInformer cache.SharedIndexInformer
	// kubernetes pod resource factory
	podFactory informers.SharedInformerFactory
	// kubernetes pod resource informer
	podInformer cache.SharedIndexInformer
	// avoidance action resource factory
	avoidanceFactory externalversions.SharedInformerFactory
	// avoidance action resource informer
	avoidanceInformer cache.SharedIndexInformer
	// node qos ensurance policy resource factory
	nepFactory externalversions.SharedInformerFactory
	// node qos ensurance policy resource informer
	nepInformer cache.SharedIndexInformer
	// recorder is an event recorder for recording Event
	recorder record.EventRecorder
}

func NewContextWithClient(client clientset.Interface, ensuranceClient ensuaranceset.Interface, nodeName string, stop <-chan struct{}) *Context {

	var ctx = &Context{kubeClient: client, stop: stop, nodeName: nodeName}

	var fieldPodSelector string
	if nodeName != "" {
		fieldPodSelector = fields.AndSelectors(fields.OneTermEqualSelector(specNodeNameField, nodeName),
			fields.OneTermNotEqualSelector(statusPhaseField, "Succeeded"),
			fields.OneTermNotEqualSelector(statusPhaseField, "Failed")).String()
	} else {
		fieldPodSelector = fields.AndSelectors(fields.OneTermNotEqualSelector(statusPhaseField, "Succeeded"),
			fields.OneTermNotEqualSelector(statusPhaseField, "Failed")).String()
	}

	ctx.podFactory = informers.NewSharedInformerFactoryWithOptions(client, informerSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fieldPodSelector
		}))

	var fieldNodeSelector string
	if nodeName != "" {
		fieldNodeSelector = fields.OneTermEqualSelector(nodeNameField, nodeName).String()
	}

	ctx.nodeFactory = informers.NewSharedInformerFactoryWithOptions(client, informerSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fieldNodeSelector
		}))

	ctx.avoidanceFactory = externalversions.NewSharedInformerFactory(ensuranceClient, informerSyncPeriod)
	ctx.nepFactory = externalversions.NewSharedInformerFactory(ensuranceClient, informerSyncPeriod)

	ctx.nodeInformer = ctx.nodeFactory.Core().V1().Nodes().Informer()
	ctx.podInformer = ctx.podFactory.Core().V1().Pods().Informer()
	ctx.avoidanceInformer = ctx.avoidanceFactory.Ensurance().V1alpha1().AvoidanceActions().Informer()
	ctx.nepInformer = ctx.nepFactory.Ensurance().V1alpha1().NodeQOSEnsurancePolicies().Informer()

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	ctx.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "crane-agent"})

	return ctx
}

// GetKubeClient return kubernetes client
func (c *Context) GetKubeClient() clientset.Interface {
	return c.kubeClient
}

// GetEnsuaranceClient return ensuarance client
func (c *Context) GetEnsuaranceClient() ensuaranceset.Interface {
	return c.ensuranceClient
}

// GetPodFactory returns pod resource factory
func (c *Context) GetPodFactory() informers.SharedInformerFactory {
	return c.podFactory
}

func (c *Context) GetPodInformer() cache.SharedIndexInformer {
	return c.podInformer
}

// GetNodeFactory returns node resource factory
func (c *Context) GetNodeFactory() informers.SharedInformerFactory {
	return c.nodeFactory
}

func (c *Context) GetNodeInformer() cache.SharedIndexInformer {
	return c.nodeInformer
}

// GetAvoidanceFactory returns AvoidanceAction resource factory
func (c *Context) GetAvoidanceFactory() externalversions.SharedInformerFactory {
	return c.avoidanceFactory
}

func (c *Context) GetAvoidanceInformer() cache.SharedIndexInformer {
	return c.avoidanceInformer
}

// GetNepFactory returns nodeqosensuarancepolicy resource factory
func (c *Context) GetNepFactory() externalversions.SharedInformerFactory {
	return c.nepFactory
}

func (c *Context) GetNepInformer() cache.SharedIndexInformer {
	return c.nepInformer
}

func (c *Context) GetRecorder() record.EventRecorder {
	return c.recorder
}

func (c *Context) GetStop() <-chan struct{} {
	return c.stop
}

// Run starts k8s informers
func (c *Context) Run() {
	if c.podFactory != nil {
		c.podFactory.Start(c.stop)
		c.podFactory.WaitForCacheSync(c.stop)
	}

	if c.nodeFactory != nil {
		c.nodeFactory.Start(c.stop)
		c.nodeFactory.WaitForCacheSync(c.stop)
	}

	if c.avoidanceFactory != nil {
		c.avoidanceFactory.Start(c.stop)
		c.avoidanceFactory.WaitForCacheSync(c.stop)
	}

	if c.nepFactory != nil {
		c.nepFactory.Start(c.stop)
		c.nepFactory.WaitForCacheSync(c.stop)
	}
}
