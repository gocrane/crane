package recommendation

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	analysisv1alpha1 "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/oom"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
	recommender "github.com/gocrane/crane/pkg/recommendation"
	"github.com/gocrane/crane/pkg/utils"
)

// RecommendationTriggerController is responsible for trigger a recommendation
type RecommendationTriggerController struct {
	client.Client
	Recorder        record.EventRecorder
	RecommenderMgr  recommender.RecommenderManager
	ScaleClient     scale.ScalesGetter
	OOMRecorder     oom.Recorder
	discoveryClient discovery.DiscoveryInterface
	dynamicClient   dynamic.Interface
	PredictorMgr    predictormgr.Manager
	Provider        providers.History
}

func (c *RecommendationTriggerController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Got Recommendation %s to be triggered", req.NamespacedName)

	recommendation := &analysisv1alpha1.Recommendation{}
	err := c.Client.Get(ctx, req.NamespacedName, recommendation)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if recommendation.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	recommendationRuleRef := utils.GetRecommendationRuleOwnerReference(recommendation)
	if recommendationRuleRef == nil {
		klog.Warning("cannot found referred recommendation rule")
		return ctrl.Result{}, nil
	}

	recommendationRule := &analysisv1alpha1.RecommendationRule{}
	err = c.Client.Get(ctx, types.NamespacedName{Name: recommendationRuleRef.Name}, recommendationRule)
	if err != nil {
		klog.Warningf("cannot found recommendation rule %s/%s", recommendation.Namespace, recommendationRuleRef.Name)
		return ctrl.Result{}, nil
	}

	if recommendation.Spec.TargetRef.Kind == "" {
		return ctrl.Result{}, fmt.Errorf(" recommendation %s has empty target kind", klog.KObj(recommendation))
	}

	gvr, err := utils.GetGroupVersionResource(c.discoveryClient, recommendation.Spec.TargetRef.APIVersion, recommendation.Spec.TargetRef.Kind)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get gvr for recommendation %s failed: %v", klog.KObj(recommendation), err)
	}

	object, err := c.dynamicClient.Resource(*gvr).Namespace(recommendation.Spec.TargetRef.Namespace).Get(ctx, recommendation.Spec.TargetRef.Name, metav1.GetOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get target object for recommendation %s failed: %v", klog.KObj(recommendation), err)
	}

	id := ObjectIdentity{
		Namespace:      object.GetNamespace(),
		Name:           object.GetName(),
		Kind:           object.GetKind(),
		APIVersion:     object.GetAPIVersion(),
		Labels:         object.GetLabels(),
		Recommender:    string(recommendation.Spec.Type),
		Object:         *object,
		Recommendation: recommendation,
	}

	newStatus := recommendationRule.Status.DeepCopy()
	currentMissionIndex := -1
	for index, mission := range newStatus.Recommendations {
		if mission.UID == recommendation.UID {
			currentMissionIndex = index
			break
		}
	}

	if currentMissionIndex == -1 {
		klog.Warningf("cannot found recommendation mission %s", recommendationRuleRef.Name)
		return ctrl.Result{}, nil
	}

	executeIdentity(context.TODO(), nil, c.RecommenderMgr, c.Provider, c.PredictorMgr, recommendationRule, id, c.Client, c.ScaleClient, c.OOMRecorder, metav1.Now(), newStatus.RunNumber)
	if newStatus.Recommendations[currentMissionIndex].Message != "Success" {
		err = c.Client.Delete(context.TODO(), recommendation)
		if err != nil {
			klog.Errorf("Failed to delete recommendation %s when checking: %v", klog.KObj(recommendation), err)
		}
	}

	updateRecommendationRuleStatus(context.TODO(), c.Client, c.Recorder, recommendationRule, newStatus)

	return ctrl.Result{}, nil
}

func (c *RecommendationTriggerController) SetupWithManager(mgr ctrl.Manager) error {
	c.discoveryClient = discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig())
	c.dynamicClient = dynamic.NewForConfigOrDie(mgr.GetConfig())

	controller, err := controller.New("recommendation-trigger-controller", mgr, controller.Options{
		Reconciler: c})
	if err != nil {
		return err
	}

	// Watch for changes to Recommendation that runNumber decrease
	return controller.Watch(&source.Kind{Type: &analysisv1alpha1.Recommendation{}}, &recommendationEventHandler{
		enqueueHandler: handler.EnqueueRequestForObject{},
	})
}

type recommendationEventHandler struct {
	enqueueHandler handler.EnqueueRequestForObject
}

func (h *recommendationEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
}

func (h *recommendationEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
}

func (h *recommendationEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	newRecommendation := evt.ObjectNew.(*analysisv1alpha1.Recommendation)
	oldRecommendation := evt.ObjectOld.(*analysisv1alpha1.Recommendation)
	klog.V(6).Infof("recommendation %s OnUpdate", klog.KObj(newRecommendation))

	oldRunNumber, _ := utils.GetRunNumber(oldRecommendation)
	newRunNumber, _ := utils.GetRunNumber(newRecommendation)
	if oldRunNumber > newRunNumber {
		// only handle this condition: the new runNumber is lower than old runNumber
		h.enqueueHandler.Update(evt, q)
	}
}

func (h *recommendationEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}
