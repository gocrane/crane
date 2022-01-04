package recommendation

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/recommend"
)

// Controller is responsible for reconcile Recommendation
type Controller struct {
	client.Client
	ConfigSet   *analysisv1alph1.ConfigSet
	Log         logr.Logger
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	RestMapper  meta.RESTMapper
	ScaleClient scale.ScalesGetter
	Predictors  map[predictionapi.AlgorithmType]prediction.Interface
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "Recommendation", req.NamespacedName)

	r := &analysisv1alph1.Recommendation{}
	err := c.Client.Get(ctx, req.NamespacedName, r)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if r.DeletionTimestamp != nil {
		// todo stop prediction
		return ctrl.Result{}, nil
	}

	if r.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical &&
		r.Spec.CompletionStrategy.PeriodSeconds != nil && r.Status.LastSuccessfulTime != nil {
		d := time.Second * time.Duration(*r.Spec.CompletionStrategy.PeriodSeconds)
		if r.Status.LastSuccessfulTime.Add(d).After(time.Now()) {
			c.Log.V(5).Info("Retry recommendation", "after", d, "Recommendation", req.NamespacedName)
			return ctrl.Result{}, nil
		}
	}

	newStatus := r.Status.DeepCopy()

	recommender, err := recommend.NewRecommender(c.Client, c.RestMapper, c.ScaleClient, r, c.Predictors, c.Log, c.ConfigSet)

	if err != nil {
		c.Recorder.Event(r, v1.EventTypeNormal, "FailedCreateRecommender", err.Error())
		c.Log.Error(err, "Failed to create recommender", "recommendation", klog.KObj(r))
		setCondition(newStatus, "Ready", metav1.ConditionFalse, "FailedCreateRecommender", "Failed to create recommender")
		c.UpdateStatus(ctx, r, newStatus)
		return ctrl.Result{}, err
	}

	proposed, err := recommender.Offer()
	if err != nil {
		c.Recorder.Event(r, v1.EventTypeNormal, "FailedOfferRecommendation", err.Error())
		c.Log.Error(err, "Failed to offer recommend", "recommendation", klog.KObj(r))
		setCondition(newStatus, "Ready", metav1.ConditionFalse, "FailedOfferRecommend", "Failed to offer recommend")
		c.UpdateStatus(ctx, r, newStatus)
		return ctrl.Result{}, err
	}

	if proposed != nil {
		newStatus.ResourceRequest = proposed.ResourceRequest
		newStatus.EffectiveHPA = proposed.EffectiveHPA
	}

	setCondition(newStatus, "Ready", metav1.ConditionTrue, "RecommendationReady", "Recommendation is ready")
	c.UpdateStatus(ctx, r, newStatus)

	if r.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical {
		if r.Spec.CompletionStrategy.PeriodSeconds != nil {
			d := time.Second * time.Duration(*r.Spec.CompletionStrategy.PeriodSeconds)
			c.Log.V(5).Info("Will re-sync", "after", d)
			return ctrl.Result{
				RequeueAfter: d,
			}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (c *Controller) UpdateStatus(ctx context.Context, recommendation *analysisv1alph1.Recommendation, newStatus *analysisv1alph1.RecommendationStatus) {
	if !equality.Semantic.DeepEqual(&recommendation.Status, newStatus) {
		c.Log.V(4).Info("Recommendation status should be updated", "currentStatus", &recommendation.Status, "newStatus", newStatus)

		recommendation.Status = *newStatus
		recommendation.Status.LastUpdateTime = metav1.Now()

		var ready = false
		for _, cond := range newStatus.Conditions {
			if cond.Reason == "RecommendationReady" && cond.Status == metav1.ConditionTrue {
				ready = true
				break
			}
		}
		if ready {
			recommendation.Status.LastSuccessfulTime = &recommendation.Status.LastUpdateTime
		}

		err := c.Update(ctx, recommendation)
		if err != nil {
			c.Recorder.Event(recommendation, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			c.Log.Error(err, "Failed to update status", "Recommendation", klog.KObj(recommendation))
			return
		}

		c.Log.Info("Update Recommendation status successful", "recommendation", klog.KObj(recommendation))
	}
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Recommendation{}).
		Complete(c)
}

func setCondition(status *analysisv1alph1.RecommendationStatus, conditionType string, conditionStatus metav1.ConditionStatus, reason string, message string) {
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
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}
