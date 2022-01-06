package ehpa

import (
	"context"
	"time"

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

const (
	RsyncPeriod = 15 * time.Second
)

// SubstituteController is responsible for sync labelSelector to Substitute
type SubstituteController struct {
	client.Client
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	ScaleClient scale.ScalesGetter
}

func (c *SubstituteController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("Got Substitute %s", req.NamespacedName)

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
		klog.Errorf("Failed to get scale, Substitute %s error %v", klog.KObj(substitute), err)
		return ctrl.Result{}, err
	}

	newStatus := autoscalingapi.SubstituteStatus{
		LabelSelector: scale.Status.Selector,
		Replicas:      substitute.Spec.Replicas,
	}

	if !equality.Semantic.DeepEqual(&substitute.Status, &newStatus) {
		klog.V(4).Infof("Substitute status should be updated", "current %v new %v", substitute.Status, newStatus)

		substitute.Status = newStatus
		err := c.Status().Update(ctx, substitute)
		if err != nil {
			c.Recorder.Event(substitute, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, Substitute %s error %v", klog.KObj(substitute), err)
			return ctrl.Result{}, err
		}

		klog.Infof("Update Substitute status successful, Substitute %s", klog.KObj(substitute))
	}

	// Rsync every 15 seconds
	return ctrl.Result{
		RequeueAfter: RsyncPeriod,
	}, nil
}

func (c *SubstituteController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.Substitute{}).
		Complete(c)
}
