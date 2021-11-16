package informer

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
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
	// kubernetes client to communication with kubernetes api-server
	kubeClient clientset.Interface
	// kubernetes node resource factory
	nodeFactory informers.SharedInformerFactory
	// kubernetes pod resource factory
	podFactory informers.SharedInformerFactory
}

func (c *Context) ContextInit() error {
	if c.kubeClient != nil {
		return nil
	}

	clientConfig, err := clientcmd.BuildConfigFromFlags(c.master, c.kubeConfig)
	if err != nil {
		klog.Errorf("BuildConfigFromFlags failed %s", err.Error())
		return err
	}

	c.kubeClient = clientset.NewForConfigOrDie(clientConfig)

	klog.Infof("ContextInit kubernetes client succeed")

	return nil
}

// GetKubeClient return kubernetes client
func (c *Context) GetKubeClient() clientset.Interface {
	c.ContextInit()
	return c.kubeClient
}

// Run starts k8s informers
func (c *Context) Run(stop <-chan struct{}) {
	if c.podFactory != nil {
		c.podFactory.Start(stop)
		c.podFactory.WaitForCacheSync(stop)
	}

	if c.nodeFactory != nil {
		c.nodeFactory.Start(stop)
		c.nodeFactory.WaitForCacheSync(stop)
	}
}

// GetPodFactory returns pod resource factory
func (c *Context) GetPodFactory() informers.SharedInformerFactory {
	if c.podFactory == nil {

		var fieldSelector string
		if c.nodeName != "" {
			fieldSelector = fields.AndSelectors(fields.OneTermEqualSelector(specNodeNameField, c.nodeName),
				fields.OneTermNotEqualSelector(statusPhaseFiled, "Succeeded"),
				fields.OneTermNotEqualSelector(statusPhaseFiled, "Failed")).String()
		} else {
			fieldSelector = fields.AndSelectors(fields.OneTermNotEqualSelector(statusPhaseFiled, "Succeeded"),
				fields.OneTermNotEqualSelector(statusPhaseFiled, "Failed")).String()
		}

		c.podFactory = informers.NewSharedInformerFactoryWithOptions(c.GetKubeClient(), informerSyncPeriod,
			informers.WithTweakListOptions(func(options *metav1.ListOptions) {
				options.FieldSelector = fieldSelector
			}))
	}

	return c.podFactory
}

// GetNodeFactory returns node resource factory
func (c *Context) GetNodeFactory() informers.SharedInformerFactory {
	if c.nodeFactory == nil {

		var fieldSelector string
		if c.nodeName != "" {
			fieldSelector = fields.OneTermEqualSelector(nodeNameField, c.nodeName).String()
		}

		c.nodeFactory = informers.NewSharedInformerFactoryWithOptions(c.GetKubeClient(), informerSyncPeriod,
			informers.WithTweakListOptions(func(options *metav1.ListOptions) {
				options.FieldSelector = fieldSelector
			}))
	}
	return c.nodeFactory
}
