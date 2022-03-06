package recommendation

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommend"
)

// Controller is responsible for reconcile Recommendation
type Controller struct {
	client.Client
	ConfigSet   *analysisv1alph1.ConfigSet
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	RestMapper  meta.RESTMapper
	ScaleClient scale.ScalesGetter
	Predictors  map[predictionapi.AlgorithmType]prediction.Interface
	Provider    providers.Interface
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got Recommendation %s", req.NamespacedName)

	recommendation := &analysisv1alph1.Recommendation{}
	err := c.Client.Get(ctx, req.NamespacedName, recommendation)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if recommendation.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	shouldRecommend := c.ShouldRecommend(recommendation)
	if !shouldRecommend {
		klog.V(4).Infof("Nothing happens for Recommendation %s", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	c.DoRecommend(ctx, recommendation)

	if recommendation.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical {
		if recommendation.Spec.CompletionStrategy.PeriodSeconds != nil {
			d := time.Second * time.Duration(*recommendation.Spec.CompletionStrategy.PeriodSeconds)
			klog.V(4).InfoS("Will re-sync", "after", d)
			return ctrl.Result{
				RequeueAfter: d,
			}, nil
		}
	}
	return ctrl.Result{}, nil
}

// ShouldRecommend decide if we need do recommendation according to status
func (c *Controller) ShouldRecommend(recommendation *analysisv1alph1.Recommendation) bool {
	lastUpdateTime := recommendation.Status.LastUpdateTime

	if recommendation.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyOnce {
		if lastUpdateTime != nil {
			// already finish recommendation
			return false
		}
	} else {
		if lastUpdateTime != nil {
			planingTime := lastUpdateTime.Add(time.Duration(*recommendation.Spec.CompletionStrategy.PeriodSeconds) * time.Second)
			if time.Now().Before(planingTime) {
				return false
			}
		}
	}

	return true
}

func (c *Controller) DoRecommend(ctx context.Context, recommendation *analysisv1alph1.Recommendation) {
	klog.V(4).Info("Starting to process Recommendation %s", klog.KObj(recommendation))

	newStatus := recommendation.Status.DeepCopy()

	recommender, err := recommend.NewRecommender(c.Client, c.RestMapper, c.ScaleClient, recommendation, c.Predictors, c.Provider, c.ConfigSet)
	if err != nil {
		c.Recorder.Event(recommendation, v1.EventTypeNormal, "FailedCreateRecommender", err.Error())
		msg := fmt.Sprintf("Failed to create recommender, Recommendation %s error %v", klog.KObj(recommendation), err)
		klog.Errorf(msg)
		setReadyCondition(newStatus, metav1.ConditionFalse, "FailedCreateRecommender", msg)
		c.UpdateStatus(ctx, recommendation, newStatus)
		return
	}

	proposed, err := recommender.Offer()
	if err != nil {
		c.Recorder.Event(recommendation, v1.EventTypeNormal, "FailedOfferRecommendation", err.Error())
		msg := fmt.Sprintf("Failed to offer recommend, Recommendation %s: %v", klog.KObj(recommendation), err)
		klog.Errorf(msg)
		setReadyCondition(newStatus, metav1.ConditionFalse, "FailedOfferRecommend", msg)
		c.UpdateStatus(ctx, recommendation, newStatus)
		return
	}

	if proposed != nil {
		if proposed.ResourceRequest != nil {
			val, _ := yaml.Marshal(proposed.ResourceRequest)
			newStatus.RecommendedValue = string(val)
		} else if proposed.EffectiveHPA != nil {
			val, _ := yaml.Marshal(proposed.EffectiveHPA)
			newStatus.RecommendedValue = string(val)
		}
	}

	setReadyCondition(newStatus, metav1.ConditionTrue, "RecommendationReady", "Recommendation is ready")
	c.UpdateStatus(ctx, recommendation, newStatus)
}

func (c *Controller) UpdateStatus(ctx context.Context, recommendation *analysisv1alph1.Recommendation, newStatus *analysisv1alph1.RecommendationStatus) {
	if !equality.Semantic.DeepEqual(&recommendation.Status, newStatus) {
		klog.V(4).Infof("Recommendation status should be updated, currentStatus %v newStatus %v", &recommendation.Status, newStatus)

		recommendation.Status = *newStatus
		timeNow := metav1.Now()
		recommendation.Status.LastUpdateTime = &timeNow

		err := c.Update(ctx, recommendation)
		if err != nil {
			c.Recorder.Event(recommendation, v1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, Recommendation %s error %v", klog.KObj(recommendation), err)
			return
		}

		klog.Infof("Update Recommendation status successful, Recommendation %s", klog.KObj(recommendation))
	}
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Recommendation{}).
		Complete(c)
}

func setReadyCondition(status *analysisv1alph1.RecommendationStatus, conditionStatus metav1.ConditionStatus, reason string, message string) {
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
