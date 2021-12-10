package informer

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	ensuaranceset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/api/pkg/generated/informers/externalversions"
	"github.com/gocrane/crane/pkg/utils/log"
)

const (
	nodeNameField      = "metadata.name"
	specNodeNameField  = "spec.nodeName"
	statusPhaseFiled   = "status.phase"
	informerSyncPeriod = time.Minute
	defaultRetryTimes  = 3
)

// Context stores k8s client and factory,which generate the resource informers
type Context struct {
	// kubernetes master address be used to connect the kubernetes api-server
	master string
	// kubernetes config used to access the kubernetes api-server
	kubeConfig string
	// nodeName for filter, if nodeName is empty not to filer
	nodeName string
	// stop channel
	stop chan struct{}
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
}

func (c *Context) ContextInit() error {
	if c.kubeClient != nil {
		return nil
	}

	clientConfig, err := clientcmd.BuildConfigFromFlags(c.master, c.kubeConfig)
	if err != nil {
		log.Logger().Error(err, "BuildConfigFromFlags failed")
		return err
	}

	c.kubeClient = clientset.NewForConfigOrDie(clientConfig)

	log.Logger().V(2).Info("ContextInit kubernetes client succeed")

	return nil
}

func NewContextInitWithClient(client clientset.Interface, ensuranceClient ensuaranceset.Interface, nodeName string) *Context {

	var ctx = &Context{kubeClient: client, stop: make(chan struct{}), nodeName: nodeName}

	var fieldPodSelector string
	if nodeName != "" {
		fieldPodSelector = fields.AndSelectors(fields.OneTermEqualSelector(specNodeNameField, nodeName),
			fields.OneTermNotEqualSelector(statusPhaseFiled, "Succeeded"),
			fields.OneTermNotEqualSelector(statusPhaseFiled, "Failed")).String()
	} else {
		fieldPodSelector = fields.AndSelectors(fields.OneTermNotEqualSelector(statusPhaseFiled, "Succeeded"),
			fields.OneTermNotEqualSelector(statusPhaseFiled, "Failed")).String()
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

func (c *Context) GetStopChannel() chan struct{} {
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
