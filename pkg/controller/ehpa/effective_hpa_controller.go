package ehpa

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

// EffectiveHPAController is responsible for scaling workload's replica based on EffectiveHorizontalPodAutoscaler spec
type EffectiveHPAController struct {
	client.Client
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	ScaleClient scale.ScalesGetter
	K8SVersion  *version.Version
}

func (c *EffectiveHPAController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got ehpa %s", req.NamespacedName)

	ehpa := &autoscalingapi.EffectiveHorizontalPodAutoscaler{}
	err := c.Client.Get(ctx, req.NamespacedName, ehpa)
	if err != nil {
		return ctrl.Result{}, err
	}

	RecordMetrics(ehpa)

	newStatus := ehpa.Status.DeepCopy()

	scale, mapping, err := utils.GetScale(ctx, c.RestMapper, c.ScaleClient, ehpa.Namespace, ehpa.Spec.ScaleTargetRef)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetScale", err.Error())
		klog.Errorf("Failed to get scale, ehpa %s", klog.KObj(ehpa))
		setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedGetScale", "Failed to get scale")
		c.UpdateStatus(ctx, ehpa, newStatus)
		return ctrl.Result{}, err
	}

	var substitute *autoscalingapi.Substitute
	if ehpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyPreview {
		substitute, err = c.ReconcileSubstitute(ctx, ehpa, scale)
		if err != nil {
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcileSubstitute", "Failed to reconcile substitute")
			c.UpdateStatus(ctx, ehpa, newStatus)
			return ctrl.Result{}, err
		}
	}

	// reconcile prediction if enabled
	if IsPredictionEnabled(ehpa) {
		prediction, err := c.ReconcilePredication(ctx, ehpa)
		if err != nil {
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcilePrediction", err.Error())
			c.UpdateStatus(ctx, ehpa, newStatus)
			return ctrl.Result{}, err
		}
		setPredictionCondition(newStatus, prediction.Status.Conditions)
	}

	hpa, err := c.ReconcileHPA(ctx, ehpa, substitute)
	if err != nil {
		setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcileHPA", err.Error())
		c.UpdateStatus(ctx, ehpa, newStatus)
		return ctrl.Result{}, err
	}

	newStatus.ExpectReplicas = &hpa.Status.DesiredReplicas
	newStatus.CurrentReplicas = &hpa.Status.CurrentReplicas

	if hpa.Status.LastScaleTime != nil && newStatus.LastScaleTime != nil && hpa.Status.LastScaleTime.After(newStatus.LastScaleTime.Time) {
		newStatus.LastScaleTime = hpa.Status.LastScaleTime
	}

	setHPACondition(newStatus, hpa.Status.Conditions)

	// scale target to its specific replicas for Preview strategy
	if ehpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyPreview && ehpa.Spec.SpecificReplicas != nil && *ehpa.Spec.SpecificReplicas != scale.Status.Replicas {
		scale.Spec.Replicas = *ehpa.Spec.SpecificReplicas
		updatedScale, err := c.ScaleClient.Scales(scale.Namespace).Update(ctx, mapping.Resource.GroupResource(), scale, metav1.UpdateOptions{})
		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedManualScale", err.Error())
			msg := fmt.Sprintf("Failed to manual scale target to specific replicas, ehpa %s replicas %d", klog.KObj(ehpa), ehpa.Spec.SpecificReplicas)
			klog.Error(err, msg)
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedScale", msg)
			c.UpdateStatus(ctx, ehpa, newStatus)
			return ctrl.Result{}, err
		}

		klog.Infof("Manual scale target to specific replicas, ehpa %s replicas %d", klog.KObj(ehpa), ehpa.Spec.SpecificReplicas)
		now := metav1.Now()
		newStatus.LastScaleTime = &now
		newStatus.CurrentReplicas = &updatedScale.Status.Replicas
	}

	setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionTrue, "EffectiveHorizontalPodAutoscalerReady", "Effective HPA is ready")
	c.UpdateStatus(ctx, ehpa, newStatus)
	return ctrl.Result{}, nil
}

func (c *EffectiveHPAController) UpdateStatus(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, newStatus *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus) {
	if !equality.Semantic.DeepEqual(&ehpa.Status, newStatus) {
		klog.V(4).Infof("EffectiveHorizontalPodAutoscaler status should be updated, currentStatus %v newStatus %v", &ehpa.Status, newStatus)

		ehpa.Status = *newStatus
		err := c.Status().Update(ctx, ehpa)
		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, ehpa %s error %v", klog.KObj(ehpa), err)
			return
		}

		klog.Infof("Update EffectiveHorizontalPodAutoscaler status successful, ehpa %s", klog.KObj(ehpa))
	}
}

func (c *EffectiveHPAController) SetupWithManager(mgr ctrl.Manager) error {
	discoveryClientSet, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	serverVersion, err := discoveryClientSet.ServerVersion()
	if err != nil {
		return err
	}
	K8SVersion, err := version.ParseGeneric(serverVersion.GitVersion)
	if err != nil {
		return err
	}
	c.K8SVersion = K8SVersion
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.EffectiveHorizontalPodAutoscaler{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Owns(&predictionapi.TimeSeriesPrediction{}).
		Complete(c)
}

func setCondition(status *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus, conditionType autoscalingapi.ConditionType, conditionStatus metav1.ConditionStatus, reason string, message string) {
	for i := range status.Conditions {
		if status.Conditions[i].Type == string(conditionType) {
			status.Conditions[i].Status = conditionStatus
			status.Conditions[i].Reason = reason
			status.Conditions[i].Message = message
			status.Conditions[i].LastTransitionTime = metav1.Now()
			return
		}
	}

	status.Conditions = append(status.Conditions, metav1.Condition{
		Type:               string(conditionType),
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func RecordMetrics(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) {
	if ehpa.Status.ExpectReplicas != nil {
		labels := map[string]string{
			"identity": klog.KObj(ehpa).String(),
			"strategy": string(ehpa.Spec.ScaleStrategy),
		}
		metrics.EHPAReplicas.With(labels).Set(float64(*ehpa.Status.ExpectReplicas))
	}
}
