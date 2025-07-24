package ehpa

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if substitute.DeletionTimestamp != nil {
		return ctrl.Result{}, err
	}

	scale, _, err := utils.GetScale(ctx, c.RestMapper, c.ScaleClient, substitute.Namespace, substitute.Spec.SubstituteTargetRef)
	if err != nil {
		c.Recorder.Event(substitute, v1.EventTypeWarning, "FailedGetScale", err.Error())
		klog.Errorf("Failed to get scale, Substitute %s error %v", klog.KObj(substitute), err)
		return ctrl.Result{}, err
	}

	newStatus := autoscalingapi.SubstituteStatus{
		LabelSelector: scale.Status.Selector,
		Replicas:      substitute.Spec.Replicas,
	}

	if substitute.Spec.Replicas != scale.Status.Replicas {
		substitute.Spec.Replicas = scale.Status.Replicas

		err := c.Update(ctx, substitute)
		if err != nil {
			c.Recorder.Event(substitute, v1.EventTypeWarning, "FailedUpdateSubstitute", err.Error())
			klog.Errorf("Failed to update Substitute %s, error %v", klog.KObj(substitute), err)
			return ctrl.Result{}, err
		}

		klog.Infof("Update Substitute successful, Substitute %s", klog.KObj(substitute))
	}

	if !equality.Semantic.DeepEqual(&substitute.Status, &newStatus) {
		substituteCopy := substitute.DeepCopy()
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			substituteCopy.Status = newStatus
			err := c.Status().Update(ctx, substituteCopy)
			if err == nil {
				return nil
			}

			updated := &autoscalingapi.Substitute{}
			errGet := c.Get(context.TODO(), types.NamespacedName{Namespace: substituteCopy.Namespace, Name: substituteCopy.Name}, updated)
			if errGet == nil {
				substituteCopy = updated
			}

			return err
		})

		if err != nil {
			c.Recorder.Event(substitute, v1.EventTypeWarning, "FailedUpdateStatus", err.Error())
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
		For(&autoscalingapi.Substitute{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(c)
}
