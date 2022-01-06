package statestore

import (
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	craneinformerfactory "github.com/gocrane/api/pkg/generated/informers/externalversions"
	craneinformers "github.com/gocrane/api/pkg/generated/informers/externalversions/ensurance/v1alpha1"
	ensurancelisters "github.com/gocrane/api/pkg/generated/listers/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
)

// Controller is the controller implementation for NodeQosEnsurance resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for crane API group
	craneClient craneclientset.Interface

	nodeQOSLister ensurancelisters.NodeQOSEnsurancePolicyLister
	nodeOOSSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	craneClient craneclientset.Interface,
	craneInformerFactory craneinformerfactory.SharedInformerFactory,
	nodeQOSInformer craneinformers.NodeQOSEnsurancePolicyInformer,
	recorder record.EventRecorder,
) *Controller {

	controller := &Controller{
		kubeclientset: kubeclientset,
		craneClient:   craneClient,
		nodeQOSLister: craneInformerFactory.Ensurance().V1alpha1().NodeQOSEnsurancePolicies().Lister(),
		nodeOOSSynced: craneInformerFactory.Ensurance().V1alpha1().NodeQOSEnsurancePolicies().Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NodeOOS"),
		recorder:      recorder,
	}

	klog.Infof("Setting up event handlers")
	// Set up an event handler for when NodeQOS resources change
	nodeQOSInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNodeQOS,
		UpdateFunc: func(old, new interface{}) {
			oldQOS := old.(*ensuranceapi.NodeQOSEnsurancePolicy)
			newQOS := new.(*ensuranceapi.NodeQOSEnsurancePolicy)

			if newQOS.DeletionTimestamp != nil {
				controller.enqueueNodeQOS(newQOS)
				return
			}

			if reflect.DeepEqual(oldQOS.Spec, newQOS.Spec) {
				klog.V(4).Infof("No changes for %s happens for spec, skip updating", klog.KObj(oldQOS))
				return
			}

			klog.V(6).Infof("Updating NodeQOSEnsurance %q", klog.KObj(oldQOS))
			controller.enqueueNodeQOS(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Infof("Starting NodeQOS controller")

	// Wait for the caches to be synced before starting workers
	klog.Infof("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.nodeOOSSynced); !ok {
		return
	}

	klog.Infof("Starting workers")
	// Launch two workers to process NodeQOS resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Infof("Started workers")
	<-stopCh
	klog.Infof("Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// NodeQOS resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the NodeQOS resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {

	// Get the NodeQOS resource with this key
	nodeQOS, err := c.nodeQOSLister.Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("NodeQOSEnsurance '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	klog.Infof("Reconcile NodeQOSEnsurance %s", klog.KObj(nodeQOS))

	// todo: your logic here

	return nil
}

// enqueueNodeQOS takes a NodeQOS resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than NodeQOS.
func (c *Controller) enqueueNodeQOS(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	nodeQOS := obj.(*ensuranceapi.NodeQOSEnsurancePolicy)

	klog.V(4).Infof("Adding NodeQOSEnsurance %q", klog.KObj(nodeQOS))
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the NodeQOS resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that NodeQOS resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Deleting object: %s", object.GetName())
	c.enqueueNodeQOS(obj)
}

func (s *Controller) List() map[string][]common.TimeSeries {
	var maps = make(map[string][]common.TimeSeries)

	// todo: your logic here

	return maps
}
