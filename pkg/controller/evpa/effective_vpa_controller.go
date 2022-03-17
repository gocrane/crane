package evpa

import (
	"context"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	vpatypes "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/autoscaling/estimator"
	"github.com/gocrane/crane/pkg/oom"
	"github.com/gocrane/crane/pkg/utils"
)

// EffectiveVPAController is responsible for scaling workload's replica based on EffectiveVerticalPodAutoscaler spec
type EffectiveVPAController struct {
	client.Client
	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	OOMRecorder      oom.Recorder
	EstimatorManager estimator.ResourceEstimatorManager
	lastScaleTime    map[string]metav1.Time
	mu               sync.Mutex
}

func (c *EffectiveVPAController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got evpa %s", req.NamespacedName)

	evpa := &autoscalingapi.EffectiveVerticalPodAutoscaler{}
	err := c.Client.Get(ctx, req.NamespacedName, evpa)
	if err != nil {
		return ctrl.Result{}, err
	}

	newStatus := evpa.Status.DeepCopy()

	err = defaultingEVPA(evpa)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeNormal, "FailedValidation", err.Error())
		msg := fmt.Sprintf("Validation EffectiveVerticalPodAutoscaler failed, evpa %s error %v", klog.KObj(evpa), err)
		klog.Error(msg)
		setCondition(newStatus, metav1.ConditionFalse, "FailedValidation", msg)
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	podTemplate, err := utils.GetPodTemplate(ctx, evpa.Namespace, evpa.Spec.TargetRef.Name, evpa.Spec.TargetRef.Kind, evpa.Spec.TargetRef.APIVersion, c.Client)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeNormal, "FailedGetPodTemplate", err.Error())
		klog.Errorf("Failed to get pod template, evpa %s", klog.KObj(evpa))
		setCondition(newStatus, metav1.ConditionFalse, "FailedGetPodTemplate", "Failed to get pod template")
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	estimators := c.EstimatorManager.GetEstimators(evpa)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeNormal, "FailedGetEstimators", err.Error())
		klog.Errorf("Failed to get estimators, evpa %s", klog.KObj(evpa))
		setCondition(newStatus, metav1.ConditionFalse, "FailedGetEstimators", "Failed to get estimators")
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	if evpa.Spec.ResourcePolicy == nil {
		return ctrl.Result{}, nil
	}

	currentEstimatorStatus, recommend, err := c.ReconcileContainerPolicies(ctx, evpa, *newStatus, podTemplate, estimators)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeNormal, "FailedReconcileContainerPolicies", err.Error())
		klog.Errorf("Failed to reconcile container policies, evpa %s", klog.KObj(evpa))
		setCondition(newStatus, metav1.ConditionFalse, "FailedGetEstimators", "Failed to get estimators")
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	newStatus.Recommendation = recommend
	newStatus.CurrentEstimators = currentEstimatorStatus
	setCondition(newStatus, metav1.ConditionTrue, "EffectiveVerticalPodAutoscalerReady", "Effective VPA is ready")
	c.UpdateStatus(ctx, evpa, newStatus)
	return ctrl.Result{}, nil
}

func (c *EffectiveVPAController) UpdateStatus(ctx context.Context, evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, newStatus *autoscalingapi.EffectiveVerticalPodAutoscalerStatus) {
	if !equality.Semantic.DeepEqual(&evpa.Status, newStatus) {
		klog.V(4).Infof("EffectiveVerticalPodAutoscaler status should be updated, currentStatus %v newStatus %v", &evpa.Status, newStatus)

		evpa.Status = *newStatus
		err := c.Status().Update(ctx, evpa)
		if err != nil {
			c.Recorder.Event(evpa, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, evpa %s error %v", klog.KObj(evpa), err)
			return
		}

		klog.Infof("Update EffectiveVerticalPodAutoscaler status successful, evpa %s", klog.KObj(evpa))
	}
}

func (c *EffectiveVPAController) SetupWithManager(mgr ctrl.Manager) error {
	estimatorManager := estimator.NewResourceEstimatorManager(mgr.GetClient(), c.OOMRecorder)
	c.EstimatorManager = estimatorManager
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.EffectiveVerticalPodAutoscaler{}).
		Complete(c)
}

func setCondition(status *autoscalingapi.EffectiveVerticalPodAutoscalerStatus, conditionStatus metav1.ConditionStatus, reason string, message string) {
	for i := range status.Conditions {
		if status.Conditions[i].Type == "Ready" {
			status.Conditions[i].Status = conditionStatus
			status.Conditions[i].Reason = reason
			status.Conditions[i].Message = message
			status.Conditions[i].LastTransitionTime = metav1.Now()
			return
		}
	}

	status.Conditions = append(status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

func defaultingEVPA(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) error {
	if evpa.Spec.ResourceEstimators == nil {
		evpa.Spec.ResourceEstimators = defaultEstimators
	} else {
		for _, estimator := range defaultEstimators {
			isFound := false
			for _, estimatorCurr := range evpa.Spec.ResourceEstimators {
				if estimator.Type == estimatorCurr.Type {
					isFound = true
					break
				}
			}

			if !isFound {
				evpa.Spec.ResourceEstimators = append(evpa.Spec.ResourceEstimators, estimator)
			}
		}
	}

	for _, estimatorCurr := range evpa.Spec.ResourceEstimators {
		if estimatorCurr.Type == "" {
			return fmt.Errorf("Estimator type cannot be empty. ")
		}
	}

	if evpa.Spec.ResourcePolicy == nil || len(evpa.Spec.ResourcePolicy.ContainerPolicies) == 0 {
		return fmt.Errorf("Resource policy or container policy cannot be empty. ")
	}
	for index, containerPolicy := range evpa.Spec.ResourcePolicy.ContainerPolicies {
		if containerPolicy.ContainerName == "" {
			return fmt.Errorf("Container name cannot be empty. ")
		}

		// scale up
		defaultScaleUpMode := vpatypes.ContainerScalingModeAuto
		defaultComponentScaleUpStabWindowSeconds := DefaultComponentScaleUpStabWindowSeconds
		defaultScaleUpCPUUtilPercentageThreshold := DefaultScaleUpCPUUtilPercentageThreshold
		defaultScaleUpMemoryUtilPercentageThreshold := DefaultScaleUpMemoryUtilPercentageThreshold
		defaultScaleUpThresholds := &autoscalingapi.ResourceMetricList{
			autoscalingapi.ResourceName("cpu"): autoscalingapi.ResourceMetric{
				Utilization: &defaultScaleUpCPUUtilPercentageThreshold,
			},
			autoscalingapi.ResourceName("memory"): autoscalingapi.ResourceMetric{
				Utilization: &defaultScaleUpMemoryUtilPercentageThreshold,
			},
		}
		defaultScaleUpPolicy := autoscalingapi.ContainerScalingPolicy{
			ScaleMode:                  &defaultScaleUpMode,
			StabilizationWindowSeconds: &defaultComponentScaleUpStabWindowSeconds,
			MetricThresholds:           defaultScaleUpThresholds,
		}
		if containerPolicy.ScaleUpPolicy == nil {
			containerPolicy.ScaleUpPolicy = &defaultScaleUpPolicy
		} else {
			if containerPolicy.ScaleUpPolicy.ScaleMode == nil {
				containerPolicy.ScaleUpPolicy.ScaleMode = &defaultScaleUpMode
			}
			if containerPolicy.ScaleUpPolicy.StabilizationWindowSeconds == nil {
				containerPolicy.ScaleUpPolicy.StabilizationWindowSeconds = &defaultComponentScaleUpStabWindowSeconds
			}
			if containerPolicy.ScaleUpPolicy.MetricThresholds == nil {
				containerPolicy.ScaleUpPolicy.MetricThresholds = defaultScaleUpThresholds
			}
		}

		// scale down
		defaultScaleDownpMode := vpatypes.ContainerScalingModeAuto
		defaultComponentScaleDownStabWindowSeconds := DefaultComponentScaleDownStabWindowSeconds
		defaultScaleDownCPUUtilPercentageThreshold := DefaultScaleDownCPUUtilPercentageThreshold
		defaultScaleDownMemoryUtilPercentageThreshold := DefaultScaleDownMemoryUtilPercentageThreshold
		defaultScaleDownThresholds := &autoscalingapi.ResourceMetricList{
			autoscalingapi.ResourceName("cpu"): autoscalingapi.ResourceMetric{
				Utilization: &defaultScaleDownCPUUtilPercentageThreshold,
			},
			autoscalingapi.ResourceName("memory"): autoscalingapi.ResourceMetric{
				Utilization: &defaultScaleDownMemoryUtilPercentageThreshold,
			},
		}
		defaultScaleDownPolicy := autoscalingapi.ContainerScalingPolicy{
			ScaleMode:                  &defaultScaleDownpMode,
			StabilizationWindowSeconds: &defaultComponentScaleDownStabWindowSeconds,
			MetricThresholds:           defaultScaleDownThresholds,
		}
		if containerPolicy.ScaleDownPolicy == nil {
			containerPolicy.ScaleDownPolicy = &defaultScaleDownPolicy
		} else {
			if containerPolicy.ScaleDownPolicy.ScaleMode == nil {
				containerPolicy.ScaleDownPolicy.ScaleMode = &defaultScaleUpMode
			}
			if containerPolicy.ScaleDownPolicy.StabilizationWindowSeconds == nil {
				containerPolicy.ScaleDownPolicy.StabilizationWindowSeconds = &defaultComponentScaleDownStabWindowSeconds
			}
			if containerPolicy.ScaleDownPolicy.MetricThresholds == nil {
				containerPolicy.ScaleDownPolicy.MetricThresholds = defaultScaleDownThresholds
			}
		}

		if containerPolicy.ControlledResources == nil {
			containerPolicy.ControlledResources = &DefaultControlledResources
		}

		evpa.Spec.ResourcePolicy.ContainerPolicies[index] = containerPolicy
	}

	return nil
}
