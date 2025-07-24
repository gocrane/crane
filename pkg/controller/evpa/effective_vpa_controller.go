package evpa

import (
	"context"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	vpatypes "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/autoscaling/estimator"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/oom"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/utils"
	"github.com/gocrane/crane/pkg/utils/target"
)

// EffectiveVPAController is responsible for scaling workload's replica based on EffectiveVerticalPodAutoscaler spec
type EffectiveVPAController struct {
	client.Client
	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	OOMRecorder      oom.Recorder
	EstimatorManager estimator.ResourceEstimatorManager
	lastScaleTime    map[string]metav1.Time
	Predictor        prediction.Interface
	TargetFetcher    target.SelectorFetcher
	mu               sync.Mutex
}

func (c *EffectiveVPAController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got evpa %s", req.NamespacedName)

	evpa := &autoscalingapi.EffectiveVerticalPodAutoscaler{}
	err := c.Get(ctx, req.NamespacedName, evpa)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return
			klog.V(3).Infof("EffectiveVPA %s has been deleted.", req)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	newStatus := evpa.Status.DeepCopy()

	err = defaultingEVPA(evpa)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeWarning, "FailedValidation", err.Error())
		msg := fmt.Sprintf("Validation EffectiveVerticalPodAutoscaler failed, evpa %s error %v", klog.KObj(evpa), err)
		klog.Error(msg)
		setCondition(newStatus, EffectiveVPAConditionTypeReady, metav1.ConditionFalse, "FailedValidation", msg)
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	podTemplate, err := utils.GetPodTemplate(ctx, evpa.Namespace, evpa.Spec.TargetRef.Name, evpa.Spec.TargetRef.Kind, evpa.Spec.TargetRef.APIVersion, c.Client)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeWarning, "FailedGetPodTemplate", err.Error())
		klog.Errorf("Failed to get pod template, evpa %s", klog.KObj(evpa))
		setCondition(newStatus, EffectiveVPAConditionTypeReady, metav1.ConditionFalse, "FailedGetPodTemplate", "Failed to get pod template")
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	estimators := c.EstimatorManager.GetEstimators(evpa)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeWarning, "FailedGetEstimators", err.Error())
		klog.Errorf("Failed to get estimators, evpa %s", klog.KObj(evpa))
		setCondition(newStatus, EffectiveVPAConditionTypeReady, metav1.ConditionFalse, "FailedGetEstimators", "Failed to get estimators")
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	if evpa.DeletionTimestamp != nil {
		c.EstimatorManager.DeleteEstimators(evpa) // release estimators
		c.CleanLastScaleTime(evpa)                // clean last scale time

		evpaCopy := evpa.DeepCopy()
		evpaCopy.Finalizers = utils.RemoveString(evpaCopy.Finalizers, known.AutoscalingFinalizer)
		err = c.Client.Update(ctx, evpaCopy)
		if err != nil {
			c.Recorder.Event(evpa, v1.EventTypeWarning, "FailedRemoveFinalizers", err.Error())
			klog.Errorf("Failed to remove finalizers, evpa %s", klog.KObj(evpa))
			setCondition(newStatus, EffectiveVPAConditionTypeReady, metav1.ConditionFalse, "FailedRemoveFinalizers", "Failed to remove finalizers")
			c.UpdateStatus(ctx, evpa, newStatus)
			return ctrl.Result{}, err
		}
		c.Recorder.Event(evpa, v1.EventTypeNormal, "RemoveFinalizers", "")
	} else if !utils.ContainsString(evpa.Finalizers, known.AutoscalingFinalizer) {
		evpa.Finalizers = append(evpa.Finalizers, known.AutoscalingFinalizer)
		err = c.Client.Update(ctx, evpa)
		if err != nil {
			c.Recorder.Event(evpa, v1.EventTypeWarning, "FailedAddFinalizers", err.Error())
			klog.Errorf("Failed to add finalizers, evpa %s", klog.KObj(evpa))
			setCondition(newStatus, EffectiveVPAConditionTypeReady, metav1.ConditionFalse, "FailedAddFinalizers", "Failed to add finalizers")
			c.UpdateStatus(ctx, evpa, newStatus)
			return ctrl.Result{}, err
		}
		c.Recorder.Event(evpa, v1.EventTypeNormal, "AddFinalizers", "Add finalizers successful.")
	}

	if evpa.Spec.ResourcePolicy == nil {
		return ctrl.Result{}, nil
	}

	currentEstimatorStatus, recommend, err := c.ReconcileContainerPolicies(evpa, podTemplate, estimators)
	if err != nil {
		c.Recorder.Event(evpa, v1.EventTypeWarning, "FailedReconcileContainerPolicies", err.Error())
		klog.Errorf("Failed to reconcile container policies, evpa %s", klog.KObj(evpa))
		setCondition(newStatus, EffectiveVPAConditionTypeReady, metav1.ConditionFalse, "FailedGetEstimators", "Failed to get estimators")
		c.UpdateStatus(ctx, evpa, newStatus)
		return ctrl.Result{}, err
	}

	newStatus.Recommendation = recommend
	newStatus.CurrentEstimators = currentEstimatorStatus

	recordMetric(evpa, newStatus, podTemplate)
	setCondition(newStatus, EffectiveVPAConditionTypeReady, metav1.ConditionTrue, "EffectiveVerticalPodAutoscaler", "EffectiveVerticalPodAutoscaler is ready")
	c.UpdateStatus(ctx, evpa, newStatus)

	return ctrl.Result{
		RequeueAfter: DefaultEVPARsyncPeriod,
	}, nil
}

func (c *EffectiveVPAController) UpdateStatus(ctx context.Context, evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, newStatus *autoscalingapi.EffectiveVerticalPodAutoscalerStatus) {
	if !equality.Semantic.DeepEqual(&evpa.Status, newStatus) {
		klog.V(4).Infof("EffectiveVerticalPodAutoscaler status should be updated, currentStatus %v newStatus %v", &evpa.Status, newStatus)

		evpa.Status = *newStatus
		err := c.Status().Update(ctx, evpa)
		if err != nil {
			c.Recorder.Event(evpa, v1.EventTypeWarning, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, evpa %s error %v", klog.KObj(evpa), err)
			return
		}

		klog.Infof("Update EffectiveVerticalPodAutoscaler status successful, evpa %s", klog.KObj(evpa))
	}
}

func (c *EffectiveVPAController) SetupWithManager(mgr ctrl.Manager) error {
	estimatorManager := estimator.NewResourceEstimatorManager(mgr.GetClient(), c.TargetFetcher, c.OOMRecorder, c.Predictor)
	c.EstimatorManager = estimatorManager
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.EffectiveVerticalPodAutoscaler{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(c)
}

func recordResourceRecommendation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, containerPolicy autoscalingapi.ContainerResourcePolicy, resourceList v1.ResourceList) {
	for resourceName, resource := range resourceList {
		labels := map[string]string{
			"apiversion": evpa.Spec.TargetRef.APIVersion,
			"owner_kind": evpa.Spec.TargetRef.Kind,
			"namespace":  evpa.Namespace,
			"owner_name": evpa.Spec.TargetRef.Name,
			"container":  containerPolicy.ContainerName,
			"resource":   resourceName.String(),
		}
		switch resourceName {
		case v1.ResourceCPU:
			metrics.EVPAResourceRecommendation.With(labels).Set(float64(resource.MilliValue()) / 1000.)
		case v1.ResourceMemory:
			metrics.EVPAResourceRecommendation.With(labels).Set(float64(resource.Value()))
		}
	}
}

func recordMetric(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, status *autoscalingapi.EffectiveVerticalPodAutoscalerStatus, podTemplate *v1.PodTemplateSpec) {

	if status.Recommendation == nil {
		return
	}
	for _, container := range status.Recommendation.ContainerRecommendations {
		labels := map[string]string{
			"apiversion": evpa.Spec.TargetRef.APIVersion,
			"owner_kind": evpa.Spec.TargetRef.Kind,
			"namespace":  evpa.Namespace,
			"owner_name": evpa.Spec.TargetRef.Name,
			"container":  container.ContainerName,
		}
		resourceRequirement, found := utils.GetResourceByPodTemplate(podTemplate, container.ContainerName)
		if !found {
			klog.Warningf("ContainerName %s not found", container.ContainerName)
			continue
		}

		recommendCpu := container.Target[v1.ResourceCPU]
		currentCpu := resourceRequirement.Requests[v1.ResourceCPU]
		labels["resource"] = v1.ResourceCPU.String()
		if currentCpu.Cmp(recommendCpu) > 0 {
			// scale down
			currCopy := currentCpu.DeepCopy()
			currCopy.Sub(recommendCpu)
			metrics.EVPACpuScaleDown.With(labels).Set(float64(currCopy.MilliValue()) / 1000.)
		} else if currentCpu.Cmp(recommendCpu) < 0 {
			// scale up
			recommendCopy := recommendCpu.DeepCopy()
			recommendCopy.Sub(currentCpu)
			metrics.EVPACpuScaleUp.With(labels).Set(float64(recommendCopy.MilliValue()) / 1000.)
		}

		recommendMem := container.Target[v1.ResourceMemory]
		currentMem := resourceRequirement.Requests[v1.ResourceMemory]
		labels["resource"] = v1.ResourceMemory.String()
		if currentMem.Cmp(recommendMem) > 0 {
			// scale down
			currCopy := currentMem.DeepCopy()
			currCopy.Sub(recommendMem)
			metrics.EVPAMemoryScaleDown.With(labels).Set(float64(currCopy.Value()))
		} else if currentMem.Cmp(recommendMem) < 0 {
			// scale up
			recommendCopy := recommendMem.DeepCopy()
			recommendCopy.Sub(currentMem)
			metrics.EVPAMemoryScaleUp.With(labels).Set(float64(recommendCopy.Value()))
		}
	}
}

// nolint:unparam
func setCondition(status *autoscalingapi.EffectiveVerticalPodAutoscalerStatus, conditionType string, conditionStatus metav1.ConditionStatus, reason string, message string) {
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
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
			return fmt.Errorf("estimator type cannot be empty. ")
		}
	}

	if evpa.Spec.ResourcePolicy == nil || len(evpa.Spec.ResourcePolicy.ContainerPolicies) == 0 {
		return fmt.Errorf("resource policy or container policy cannot be empty. ")
	}
	for index, containerPolicy := range evpa.Spec.ResourcePolicy.ContainerPolicies {
		if containerPolicy.ContainerName == "" {
			return fmt.Errorf("container name cannot be empty. ")
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
		defaultScaleDownMode := vpatypes.ContainerScalingModeAuto
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
			ScaleMode:                  &defaultScaleDownMode,
			StabilizationWindowSeconds: &defaultComponentScaleDownStabWindowSeconds,
			MetricThresholds:           defaultScaleDownThresholds,
		}
		if containerPolicy.ScaleDownPolicy == nil {
			containerPolicy.ScaleDownPolicy = &defaultScaleDownPolicy
		} else {
			if containerPolicy.ScaleDownPolicy.ScaleMode == nil {
				containerPolicy.ScaleDownPolicy.ScaleMode = &defaultScaleDownMode
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
