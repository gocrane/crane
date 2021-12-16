package recommendation

import (
	"context"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

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
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/recommend"
)

// RecommendationController is responsible for reconcile Recommendation
type RecommendationController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	RestMapper  meta.RESTMapper
	ScaleClient scale.ScalesGetter
	Predictors  map[predictionapi.AlgorithmType]prediction.Interface
}

func (c *RecommendationController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "Recommendation", req.NamespacedName)

	recommendation := &analysisv1alph1.Recommendation{}
	err := c.Client.Get(ctx, req.NamespacedName, recommendation)
	if err != nil {
		return ctrl.Result{}, err
	}

	if recommendation.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	newStatus := recommendation.Status.DeepCopy()

	recommender, err := recommend.NewRecommender(c.Client, c.RestMapper, c.ScaleClient, recommendation, c.Predictors)
	if err != nil {
		c.Recorder.Event(recommendation, v1.EventTypeNormal, "FailedCreateRecommender", err.Error())
		c.Log.Error(err, "Failed to create recommender", "recommendation", klog.KObj(recommendation))
		setCondition(newStatus, "Ready", metav1.ConditionFalse, "FailedCreateRecommender", "Failed to create recommender")
		c.UpdateStatus(ctx, recommendation, newStatus)
		return ctrl.Result{}, err
	}

	proposed, err := recommender.Offer()
	if err != nil {
		c.Recorder.Event(recommendation, v1.EventTypeNormal, "FailedOfferRecommend", err.Error())
		c.Log.Error(err, "Failed to offer recommend", "recommendation", klog.KObj(recommendation))
		setCondition(newStatus, "Ready", metav1.ConditionFalse, "FailedOfferRecommend", "Failed to offer recommend")
		c.UpdateStatus(ctx, recommendation, newStatus)
		return ctrl.Result{}, err
	}

	newStatus.ResourceRequest = proposed.ResourceRequest
	newStatus.EffectiveHPA = proposed.EffectiveHPA

	setCondition(newStatus, "Ready", metav1.ConditionTrue, "RecommendationReady", "Recommendation is ready")
	c.UpdateStatus(ctx, recommendation, newStatus)

	return ctrl.Result{}, nil
}

func (c *RecommendationController) UpdateStatus(ctx context.Context, recommendation *analysisv1alph1.Recommendation, newStatus *analysisv1alph1.RecommendationStatus) {
	if !equality.Semantic.DeepEqual(&recommendation.Status, newStatus) {
		c.Log.V(4).Info("Recommendation status should be updated", "currentStatus", &recommendation.Status, "newStatus", newStatus)

		recommendation.Status = *newStatus
		recommendation.Status.LastUpdateTime = metav1.Now()
		err := c.Status().Update(ctx, recommendation)
		if err != nil {
			c.Recorder.Event(recommendation, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			c.Log.Error(err, "Failed to update status", "Recommendation", klog.KObj(recommendation))
			return
		}

		c.Log.Info("Update Recommendation status successful", "ehpa", klog.KObj(recommendation))
	}
}

func (c *RecommendationController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Recommendation{}).
		Complete(c)
}

func setCondition(status *analysisv1alph1.RecommendationStatus, conditionType string, conditionStatus metav1.ConditionStatus, reason string, message string) {
	for _, cond := range status.Conditions {
		if cond.Type == conditionType {
			cond.Status = conditionStatus
			cond.Reason = reason
			cond.Message = message
			cond.LastTransitionTime = metav1.Now()
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
