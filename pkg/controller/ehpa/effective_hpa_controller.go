package ehpa

import (
	"context"

	"github.com/go-logr/logr"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/metrics"
)

// EffectiveHPAController is responsible for scaling workload's replica based on EffectiveHorizontalPodAutoscaler spec
type EffectiveHPAController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	scaleClient scale.ScalesGetter
	K8SVersion  *version.Version
}

func (c *EffectiveHPAController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "ehpa", req.NamespacedName)

	ehpa := &autoscalingapi.EffectiveHorizontalPodAutoscaler{}
	err := c.Client.Get(ctx, req.NamespacedName, ehpa)
	if err != nil {
		return ctrl.Result{}, err
	}

	// record expect replicas
	labels := map[string]string{
		"identity": klog.KObj(ehpa).String(),
		"strategy": string(ehpa.Spec.ScaleStrategy),
	}
	metrics.EHPAReplicas.With(labels).Set(float64(*ehpa.Status.ExpectReplicas))

	newStatus := ehpa.Status.DeepCopy()

	scale, mapping, err := GetScale(ctx, c.RestMapper, c.scaleClient, ehpa.Namespace, ehpa.Spec.ScaleTargetRef)
	if err != nil {
		c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedGetScale", err.Error())
		c.Log.Error(err, "Failed to get scale", "ehpa", klog.KObj(ehpa))
		setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedGetScale", "Failed to get scale")
		c.UpdateStatus(ctx, ehpa, newStatus)
		return ctrl.Result{}, err
	}

	if ehpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyPreview {
		substitute, err := c.ReconcileSubstitute(ctx, ehpa, scale)
		if err != nil {
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcileSubstitute", "Failed to reconcile substitute")
			c.UpdateStatus(ctx, ehpa, newStatus)
			return ctrl.Result{}, err
		}

		hpa, err := c.ReconcileHPA(ctx, ehpa, substitute)
		if err != nil {
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcileHPA", err.Error())
			c.UpdateStatus(ctx, ehpa, newStatus)
			return ctrl.Result{}, err
		}

		newStatus.ExpectReplicas = &hpa.Status.DesiredReplicas
		newStatus.CurrentReplicas = &scale.Status.Replicas

		// scale target to its specific replicas
		if ehpa.Spec.SpecificReplicas != nil && *ehpa.Spec.SpecificReplicas != scale.Status.Replicas {
			scale.Spec.Replicas = *ehpa.Spec.SpecificReplicas
			updatedScale, err := c.scaleClient.Scales(scale.Namespace).Update(ctx, mapping.Resource.GroupResource(), scale, metav1.UpdateOptions{})
			if err != nil {
				c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedManualScale", err.Error())
				c.Log.Error(err, "Failed to manual scale target to specific replicas", "ehpa", klog.KObj(ehpa), "replicas", ehpa.Spec.SpecificReplicas)
				setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedScale", "Failed to scale target manually")
				c.UpdateStatus(ctx, ehpa, newStatus)
				return ctrl.Result{}, err
			}

			c.Log.Info("Manual scale target to specific replicas", "ehpa", klog.KObj(ehpa), "replicas", ehpa.Spec.SpecificReplicas)
			now := metav1.Now()
			newStatus.LastScaleTime = &now
			newStatus.CurrentReplicas = &updatedScale.Status.Replicas
		}
	} else if ehpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyAuto {
		// reconcile prediction if enabled
		if IsPredictionEnabled(ehpa) {
			prediction, err := c.ReconcilePodPredication(ctx, ehpa)
			if err != nil {
				setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcilePrediction", err.Error())
				c.UpdateStatus(ctx, ehpa, newStatus)
				return ctrl.Result{}, err
			}
			setPredictionCondition(newStatus, prediction.Status.Conditions)
		}

		hpa, err := c.ReconcileHPA(ctx, ehpa, nil)
		if err != nil {
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcileHPA", err.Error())
			c.UpdateStatus(ctx, ehpa, newStatus)
			return ctrl.Result{}, err
		}

		newStatus.ExpectReplicas = &hpa.Status.DesiredReplicas
		newStatus.LastScaleTime = hpa.Status.LastScaleTime
		newStatus.CurrentReplicas = &hpa.Status.CurrentReplicas

		setHPACondition(newStatus, hpa.Status.Conditions)
	}

	setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionTrue, "EffectiveHorizontalPodAutoscalerReady", "Effective HPA is ready")
	c.UpdateStatus(ctx, ehpa, newStatus)
	return ctrl.Result{}, nil
}

func (c *EffectiveHPAController) UpdateStatus(ctx context.Context, ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler, newStatus *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus) {
	if !equality.Semantic.DeepEqual(&ehpa.Status, newStatus) {
		c.Log.V(4).Info("EffectiveHorizontalPodAutoscaler status should be updated", "currentStatus", &ehpa.Status, "newStatus", newStatus)

		ehpa.Status = *newStatus
		err := c.Status().Update(ctx, ehpa)
		if err != nil {
			c.Recorder.Event(ehpa, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			c.Log.Error(err, "Failed to update status", "ehpa", klog.KObj(ehpa))
			return
		}

		c.Log.Info("Update EffectiveHorizontalPodAutoscaler status successful", "ehpa", klog.KObj(ehpa))
	}
}

func (c *EffectiveHPAController) GetPodsFromScale(scale *autoscalingapiv1.Scale) ([]v1.Pod, error) {
	selector, err := labels.ConvertSelectorToLabelsMap(scale.Status.Selector)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{
		client.InNamespace(scale.GetNamespace()),
		client.MatchingLabels(selector),
	}

	podList := &v1.PodList{}
	err = c.Client.List(context.TODO(), podList, opts...)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func (c *EffectiveHPAController) SetupWithManager(mgr ctrl.Manager) error {
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
		Owns(&predictionapi.PodGroupPrediction{}).
		Complete(c)
}

func setCondition(status *autoscalingapi.EffectiveHorizontalPodAutoscalerStatus, conditionType autoscalingapi.ConditionType, conditionStatus metav1.ConditionStatus, reason string, message string) {
	for _, cond := range status.Conditions {
		if cond.Type == string(conditionType) {
			cond.Status = conditionStatus
			cond.Reason = reason
			cond.Message = message
			cond.LastTransitionTime = metav1.Now()
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
