package ehpa

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	"github.com/gocrane/crane/pkg/utils"
)

// SubstituteController is responsible for sync labelSelector to Substitute
type SubstituteController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	ScaleClient scale.ScalesGetter
}

func (c *SubstituteController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "Substitute", req.NamespacedName)

	substitute := &autoscalingapi.Substitute{}
	err := c.Client.Get(ctx, req.NamespacedName, substitute)
	if err != nil {
		return ctrl.Result{}, err
	}

	if substitute.DeletionTimestamp != nil {
		return ctrl.Result{}, err
	}

	scale, _, err := utils.GetScale(ctx, c.RestMapper, c.ScaleClient, substitute.Namespace, substitute.Spec.SubstituteTargetRef)
	if err != nil {
		c.Recorder.Event(substitute, v1.EventTypeNormal, "FailedGetScale", err.Error())
		c.Log.Error(err, "Failed to get scale", "Substitute", klog.KObj(substitute))
		return ctrl.Result{}, err
	}

	newStatus := autoscalingapi.SubstituteStatus{
		LabelSelector: scale.Status.Selector,
		Replicas: substitute.Spec.Replicas,
	}

	if !equality.Semantic.DeepEqual(&substitute.Status, &newStatus) {
		c.Log.V(4).Info("Substitute status should be updated", "current", substitute.Status, "new", newStatus)

		substitute.Status = newStatus
		err := c.Status().Update(ctx, substitute)
		if err != nil {
			c.Recorder.Event(substitute, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			c.Log.Error(err, "Failed to update status", "Substitute", klog.KObj(substitute))
			return ctrl.Result{}, err
		}

		c.Log.Info("Update Substitute status successful", "Substitute", klog.KObj(substitute))
	}

	// Rsync every 15 seconds
	return ctrl.Result{
		RequeueAfter: time.Second * 15,
	}, nil
}

func (c *SubstituteController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.Substitute{}).
		Complete(c)
}
