package hpa

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
)

// AdvancedHPAController is responsible for scaling workload's replica based on AdvancedHorizontalPodAutoscaler spec
type AdvancedHPAController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	scaleClient scale.ScalesGetter
	K8SVersion  *version.Version
}

func (p *AdvancedHPAController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	p.Log.Info("got", "advanced-hpa", req.NamespacedName)

	ahpa := &autoscalingapi.AdvancedHorizontalPodAutoscaler{}
	err := p.Client.Get(ctx, req.NamespacedName, ahpa)
	if err != nil {
		return ctrl.Result{}, err
	}

	newStatus := ahpa.Status.DeepCopy()

	scale, mapping, err := p.GetScale(ctx, ahpa)
	if err != nil {
		p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedGetScale", err.Error())
		p.Log.Error(err, "Failed to get scale", "advanced-hpa", klog.KObj(ahpa))
		setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedGetScale", "Failed to get scale")
		p.UpdateStatus(ctx, ahpa, newStatus)
		return ctrl.Result{}, err
	}

	if ahpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyManual {
		err = p.DisableHPA(ctx, ahpa)
		if err != nil {
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedDisableHPA", "Failed to disable hpa")
			p.UpdateStatus(ctx, ahpa, newStatus)
			return ctrl.Result{}, err
		}

		newStatus.ExpectReplicas = ahpa.Spec.SpecificReplicas
		newStatus.CurrentReplicas = &scale.Status.Replicas

		// todo: validation SpecificReplicas is between minReplicas and maxReplicas in webhook
		// scale target to its specific replicas
		if ahpa.Spec.SpecificReplicas != nil && *ahpa.Spec.SpecificReplicas != scale.Status.Replicas {
			scale.Spec.Replicas = *ahpa.Spec.SpecificReplicas
			updatedScale, err := p.scaleClient.Scales(scale.Namespace).Update(ctx, mapping.Resource.GroupResource(), scale, metav1.UpdateOptions{})
			if err != nil {
				p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedManualScale", err.Error())
				p.Log.Error(err, "Failed to manual scale target to specific replicas", "advanced-hpa", klog.KObj(ahpa), "replicas", ahpa.Spec.SpecificReplicas)
				setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedScale", "Failed to scale target manually")
				p.UpdateStatus(ctx, ahpa, newStatus)
				return ctrl.Result{}, err
			}

			p.Log.Info("Manual scale target to specific replicas", "advanced-hpa", klog.KObj(ahpa), "replicas", ahpa.Spec.SpecificReplicas)
			now := metav1.Now()
			newStatus.LastScaleTime = &now
			newStatus.CurrentReplicas = &updatedScale.Status.Replicas
		}
	} else if ahpa.Spec.ScaleStrategy == autoscalingapi.ScaleStrategyAuto {
		// reconcile prediction if enabled
		if IsPredictionEnabled(ahpa) {
			prediction, err := p.ReconcilePodPredication(ctx, ahpa)
			if err != nil {
				setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcilePrediction", err.Error())
				p.UpdateStatus(ctx, ahpa, newStatus)
				return ctrl.Result{}, err
			}
			setPredictionCondition(newStatus, prediction.Status.Conditions)
		}

		hpa, err := p.ReconcileHPA(ctx, ahpa)
		if err != nil {
			setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionFalse, "FailedReconcileHPA", err.Error())
			p.UpdateStatus(ctx, ahpa, newStatus)
			return ctrl.Result{}, err
		}

		newStatus.ExpectReplicas = &hpa.Status.DesiredReplicas
		newStatus.LastScaleTime = hpa.Status.LastScaleTime
		newStatus.CurrentReplicas = &hpa.Status.CurrentReplicas

		setHPACondition(newStatus, hpa.Status.Conditions)
	}

	setCondition(newStatus, autoscalingapi.Ready, metav1.ConditionTrue, "AdvancedHorizontalPodAutoscalerReady", "Advanced HPA is ready")
	return ctrl.Result{}, p.UpdateStatus(ctx, ahpa, newStatus)
}

func (p *AdvancedHPAController) UpdateStatus(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler, newStatus *autoscalingapi.AdvancedHorizontalPodAutoscalerStatus) error {
	if !equality.Semantic.DeepEqual(&ahpa.Status, newStatus) {
		p.Log.V(4).Info("AdvancedHorizontalPodAutoscaler status should be updated", "currentStatus", &ahpa.Status, "newStatus", newStatus)

		ahpa.Status = *newStatus
		err := p.Status().Update(ctx, ahpa)
		if err != nil {
			p.Recorder.Event(ahpa, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			p.Log.Error(err, "Failed to update status", "advanced-hpa", klog.KObj(ahpa))
			return err
		}

		p.Log.Info("Update AdvancedHorizontalPodAutoscaler status successful", "advanced-hpa", klog.KObj(ahpa))
	}

	return nil
}

func (p *AdvancedHPAController) GetScale(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*autoscalingapiv1.Scale, *meta.RESTMapping, error) {
	targetGV, err := schema.ParseGroupVersion(ahpa.Spec.ScaleTargetRef.APIVersion)
	if err != nil {
		return nil, nil, err
	}

	targetGK := schema.GroupKind{
		Group: targetGV.Group,
		Kind:  ahpa.Spec.ScaleTargetRef.Kind,
	}

	mappings, err := p.RestMapper.RESTMappings(targetGK)
	if err != nil {
		return nil, nil, err
	}

	for _, mapping := range mappings {
		scale, err := p.scaleClient.Scales(ahpa.Namespace).Get(ctx, mapping.Resource.GroupResource(), ahpa.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
		if err == nil {
			return scale, mapping, nil
		}
	}

	return nil, nil, fmt.Errorf("unrecognized resource")
}

func (p *AdvancedHPAController) GetPodsFromScale(scale *autoscalingapiv1.Scale) ([]v1.Pod, error) {
	selector, err := labels.ConvertSelectorToLabelsMap(scale.Status.Selector)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{
		client.InNamespace(scale.GetNamespace()),
		client.MatchingLabels(selector),
	}

	podList := &v1.PodList{}
	err = p.Client.List(context.TODO(), podList, opts...)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func (p *AdvancedHPAController) SetupWithManager(mgr ctrl.Manager) error {
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
	p.scaleClient = scaleClient
	serverVersion, err := discoveryClientSet.ServerVersion()
	if err != nil {
		return err
	}
	K8SVersion, err := version.ParseGeneric(serverVersion.GitVersion)
	if err != nil {
		return err
	}
	p.K8SVersion = K8SVersion
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.AdvancedHorizontalPodAutoscaler{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Owns(&predictionapi.PodGroupPrediction{}).
		Complete(p)
}

func setCondition(status *autoscalingapi.AdvancedHorizontalPodAutoscalerStatus, conditionType autoscalingapi.ConditionType, conditionStatus metav1.ConditionStatus, reason string, message string) {
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
