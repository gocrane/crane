package ehpa

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

// SubstituteController is responsible for sync labelSelector to Substitute
type SubstituteController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	scaleClient scale.ScalesGetter
}

func (c *SubstituteController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "Substitute", req.NamespacedName)

	substitute := &autoscalingapi.Substitute{}
	err := c.Client.Get(ctx, req.NamespacedName, substitute)
	if err != nil {
		return ctrl.Result{}, err
	}

	scale, _, err := GetScale(ctx, c.RestMapper, c.scaleClient, substitute.Namespace, substitute.Spec.SubstituteTargetRef)
	if err != nil {
		c.Recorder.Event(substitute, v1.EventTypeNormal, "FailedGetScale", err.Error())
		c.Log.Error(err, "Failed to get scale", "Substitute", klog.KObj(substitute))
		return ctrl.Result{}, err
	}

	if substitute.Status.LabelSelector != scale.Status.Selector || substitute.Status.Replicas != *substitute.Spec.Replicas {
		c.Log.V(4).Info("Substitute labelSelector should be updated", "current", substitute.Status.LabelSelector, "new", scale.Status.Selector)

		// Update substitute labelSelector based on target scale
		substitute.Status.LabelSelector = scale.Status.Selector
		substitute.Status.Replicas = *substitute.Spec.Replicas
		err := c.Status().Update(ctx, substitute)
		if err != nil {
			c.Recorder.Event(substitute, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			c.Log.Error(err, "Failed to update status", "Substitute", klog.KObj(substitute))
			return ctrl.Result{}, err
		}

		c.Log.Info("Update Substitute status successful", "Substitute", klog.KObj(substitute))
	}

	// Rsync every 1 minute
	return ctrl.Result{
		RequeueAfter: time.Minute,
	}, nil
}

func (c *SubstituteController) SetupWithManager(mgr ctrl.Manager) error {
	discoveryClientSet, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discoveryClientSet)
	scaleClient := scale.New(
		discoveryClientSet.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scaleKindResolver,
	)
	c.scaleClient = scaleClient
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.Substitute{}).
		Complete(c)
}
