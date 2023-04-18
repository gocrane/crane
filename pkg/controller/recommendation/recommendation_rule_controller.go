package recommendation

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
	recommender "github.com/gocrane/crane/pkg/recommendation"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

type RecommendationRuleController struct {
	client.Client
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	RestMapper      meta.RESTMapper
	ScaleClient     scale.ScalesGetter
	RecommenderMgr  recommender.RecommenderManager
	PredictorMgr    predictormgr.Manager
	kubeClient      kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	Provider        providers.History
}

func (c *RecommendationRuleController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).InfoS("Got a RecommendationRule resource.", "RecommendationRule", req.NamespacedName)

	recommendationRule := &analysisv1alph1.RecommendationRule{}

	err := c.Client.Get(ctx, req.NamespacedName, recommendationRule)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if recommendationRule.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	lastUpdateTime := recommendationRule.Status.LastUpdateTime
	if len(strings.TrimSpace(recommendationRule.Spec.RunInterval)) == 0 {
		if lastUpdateTime != nil {
			// This is a one-off recommendationRule task which has been completed.
			return ctrl.Result{}, nil
		}
	}

	interval, err := time.ParseDuration(recommendationRule.Spec.RunInterval)
	if err != nil {
		c.Recorder.Event(recommendationRule, corev1.EventTypeWarning, "FailedParseRunInterval", err.Error())
		klog.Errorf("Failed to parse RunInterval, recommendationRule %s", klog.KObj(recommendationRule))
		return ctrl.Result{}, err
	}

	if lastUpdateTime != nil {
		planingTime := lastUpdateTime.Add(interval)
		now := time.Now()
		if now.Before(planingTime) {
			return ctrl.Result{
				RequeueAfter: planingTime.Sub(now),
			}, nil
		}
	}

	finished := c.doReconcile(ctx, recommendationRule, interval)
	if finished && len(strings.TrimSpace(recommendationRule.Spec.RunInterval)) != 0 {
		klog.V(4).InfoS("Will re-sync", "after", interval)
		// Arrange for next round.
		return ctrl.Result{
			RequeueAfter: interval,
		}, nil
	}

	return ctrl.Result{RequeueAfter: time.Second * 1}, nil
}

func (c *RecommendationRuleController) doReconcile(ctx context.Context, recommendationRule *analysisv1alph1.RecommendationRule, interval time.Duration) bool {
	newStatus := recommendationRule.Status.DeepCopy()

	identities, err := c.getIdentities(ctx, recommendationRule)
	if err != nil {
		c.Recorder.Event(recommendationRule, corev1.EventTypeWarning, "FailedSelectResource", err.Error())
		msg := fmt.Sprintf("Failed to get idenitities, RecommendationRule %s error %v", klog.KObj(recommendationRule), err)
		klog.Errorf(msg)
		updateRecommendationRuleStatus(ctx, c.Client, c.Recorder, recommendationRule, newStatus)
		return false
	}

	timeNow := metav1.Now()

	// if the first mission start time is last round, reset currMissions here
	currMissions := newStatus.Recommendations
	if currMissions != nil && len(currMissions) > 0 {
		firstMissionStartTime := currMissions[0].LastStartTime
		if firstMissionStartTime.IsZero() {
			currMissions = nil
		} else {
			planingTime := firstMissionStartTime.Add(interval)
			if time.Now().After(planingTime) {
				currMissions = nil // reset missions to trigger creation for missions
			}
		}
	}

	if currMissions == nil {
		// create recommendation rule missions for this round
		// every recommendation rule have multi recommender for one identity
		for _, id := range identities {
			for _, recommender := range recommendationRule.Spec.Recommenders {
				currMissions = append(currMissions, analysisv1alph1.RecommendationMission{
					TargetRef: id.GetObjectReference(),
					RecommenderRef: analysisv1alph1.Recommender{
						Name: recommender.Name,
					},
				})
			}
		}

		// +1 for runNumber
		newStatus.RunNumber = newStatus.RunNumber + 1
	}

	var currRecommendations analysisv1alph1.RecommendationList
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{known.RecommendationRuleUidLabel: string(recommendationRule.UID)}),
	}
	if convert, uid := IsConvertFromAnalytics(recommendationRule); convert {
		opts = []client.ListOption{
			client.MatchingLabels(map[string]string{known.AnalyticsUidLabel: uid}),
		}
	}

	err = c.Client.List(ctx, &currRecommendations, opts...)
	if err != nil {
		c.Recorder.Event(recommendationRule, corev1.EventTypeWarning, "FailedSelectResource", err.Error())
		msg := fmt.Sprintf("Failed to get recomendations, RecommendationRule %s error %v", klog.KObj(recommendationRule), err)
		klog.Errorf(msg)
		updateRecommendationRuleStatus(ctx, c.Client, c.Recorder, recommendationRule, newStatus)
		return false
	}

	if klog.V(6).Enabled() {
		// Print identities
		for k, id := range identities {
			klog.V(6).InfoS("identities", "RecommendationRule", klog.KObj(recommendationRule), "key", k, "apiVersion", id.APIVersion, "kind", id.Kind, "namespace", id.Namespace, "name", id.Name)
		}
	}

	maxConcurrency := 10
	executionIndex := -1
	var concurrency int
	for index, mission := range currMissions {
		if mission.LastStartTime != nil {
			continue
		}
		if executionIndex == -1 {
			executionIndex = index
		}
		if concurrency < maxConcurrency {
			concurrency++
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(concurrency)
	for index := executionIndex; index < len(currMissions) && index < concurrency+executionIndex; index++ {
		var existingRecommendation *analysisv1alph1.Recommendation
		for _, r := range currRecommendations.Items {
			if reflect.DeepEqual(currMissions[index].TargetRef, r.Spec.TargetRef) && string(r.Spec.Type) == currMissions[index].RecommenderRef.Name {
				existingRecommendation = &r
				break
			}
		}

		go executeMission(ctx, &wg, c.RecommenderMgr, c.Provider, c.PredictorMgr, recommendationRule, identities, &currMissions[index], existingRecommendation, c.Client, c.ScaleClient, timeNow, newStatus.RunNumber)
	}

	wg.Wait()

	finished := false
	if executionIndex+concurrency == len(currMissions) || len(currMissions) == 0 {
		finished = true
	}

	if finished {
		newStatus.LastUpdateTime = &timeNow
		// clean orphan recommendations
		for _, recommendation := range currRecommendations.Items {
			exist := false
			for _, mission := range currMissions {
				if recommendation.UID == mission.UID {
					exist = true
					break
				}
			}

			if !exist {
				err = c.Client.Delete(ctx, &recommendation)
				if err != nil {
					klog.ErrorS(err, "Failed to delete recommendation.", "recommendation", klog.KObj(&recommendation))
				} else {
					klog.Infof("Deleted orphan recommendation %v.", klog.KObj(&recommendation))
				}
			}
		}

	}

	newStatus.Recommendations = currMissions

	updateRecommendationRuleStatus(ctx, c.Client, c.Recorder, recommendationRule, newStatus)
	return finished
}

func (c *RecommendationRuleController) SetupWithManager(mgr ctrl.Manager) error {
	c.kubeClient = kubernetes.NewForConfigOrDie(mgr.GetConfig())
	c.discoveryClient = discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig())
	c.dynamicClient = dynamic.NewForConfigOrDie(mgr.GetConfig())

	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.RecommendationRule{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(c)
}

func (c *RecommendationRuleController) getIdentities(ctx context.Context, recommendationRule *analysisv1alph1.RecommendationRule) (map[string]ObjectIdentity, error) {
	identities := map[string]ObjectIdentity{}

	for _, rs := range recommendationRule.Spec.ResourceSelectors {
		if rs.Kind == "" {
			return nil, fmt.Errorf("empty kind")
		}

		gvr, err := utils.GetGroupVersionResource(c.discoveryClient, rs.APIVersion, rs.Kind)
		if err != nil {
			return nil, err
		}

		var unstructureds []unstructuredv1.Unstructured
		if recommendationRule.Spec.NamespaceSelector.Any {
			unstructuredList, err := c.dynamicClient.Resource(*gvr).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			unstructureds = append(unstructureds, unstructuredList.Items...)
		} else {
			for _, namespace := range recommendationRule.Spec.NamespaceSelector.MatchNames {
				unstructuredList, err := c.dynamicClient.Resource(*gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}

				unstructureds = append(unstructureds, unstructuredList.Items...)
			}
		}

		var filterdUnstructureds []unstructuredv1.Unstructured
		for _, unstructed := range unstructureds {
			if rs.Name != "" && unstructed.GetName() != rs.Name {
				// filter Name that not match
				continue
			}

			if match, _ := utils.LabelSelectorMatched(unstructed.GetLabels(), rs.LabelSelector); !match {
				// filter that not match labelSelector
				continue
			}
			filterdUnstructureds = append(filterdUnstructureds, unstructed)
		}

		for i := range filterdUnstructureds {
			k := objRefKey(rs.Kind, rs.APIVersion, filterdUnstructureds[i].GetNamespace(), filterdUnstructureds[i].GetName())
			if _, exists := identities[k]; !exists {
				identities[k] = ObjectIdentity{
					Namespace:  filterdUnstructureds[i].GetNamespace(),
					Name:       filterdUnstructureds[i].GetName(),
					Kind:       rs.Kind,
					APIVersion: rs.APIVersion,
					Labels:     filterdUnstructureds[i].GetLabels(),
					Object:     filterdUnstructureds[i],
				}
			}
		}
	}

	return identities, nil
}

func updateRecommendationRuleStatus(ctx context.Context, c client.Client, recorder record.EventRecorder, recommendationRule *analysisv1alph1.RecommendationRule, newStatus *analysisv1alph1.RecommendationRuleStatus) {
	if !equality.Semantic.DeepEqual(&recommendationRule.Status, newStatus) {
		klog.V(2).Infof("Updating RecommendationRule %s status", klog.KObj(recommendationRule))
		recommendationRuleCopy := recommendationRule.DeepCopy()
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			recommendationRuleCopy.Status = *newStatus
			err := c.Update(ctx, recommendationRuleCopy)
			if err == nil {
				return nil
			}

			updated := &analysisv1alph1.RecommendationRule{}
			errGet := c.Get(context.TODO(), types.NamespacedName{Namespace: recommendationRuleCopy.Namespace, Name: recommendationRuleCopy.Name}, updated)
			if errGet == nil {
				recommendationRuleCopy = updated
			}

			return err
		})

		if err != nil {
			recorder.Event(recommendationRule, corev1.EventTypeWarning, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, RecommendationRule %s error %v", klog.KObj(recommendationRule), err)
			return
		}

		klog.V(2).Infof("Update RecommendationRule status successful, RecommendationRule %s", klog.KObj(recommendationRule))
	}
}

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
	Labels     map[string]string
	Object     unstructuredv1.Unstructured
}

func (id ObjectIdentity) GetObjectReference() corev1.ObjectReference {
	return corev1.ObjectReference{Kind: id.Kind, APIVersion: id.APIVersion, Namespace: id.Namespace, Name: id.Name}
}

func newOwnerRef(a *analysisv1alph1.RecommendationRule) *metav1.OwnerReference {
	blockOwnerDeletion, isController := false, false
	return &metav1.OwnerReference{
		APIVersion:         a.APIVersion,
		Kind:               a.Kind,
		Name:               a.GetName(),
		UID:                a.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}

func objRefKey(kind, apiVersion, namespace, name string) string {
	return fmt.Sprintf("%s#%s#%s#%s", kind, apiVersion, namespace, name)
}

func CreateRecommendationObject(recommendationRule *analysisv1alph1.RecommendationRule,
	target corev1.ObjectReference, id ObjectIdentity, recommenderName string) *analysisv1alph1.Recommendation {

	namespace := known.CraneSystemNamespace
	if id.Namespace != "" {
		namespace = id.Namespace
	}
	if convert, _ := IsConvertFromAnalytics(recommendationRule); convert {
		namespace = known.CraneSystemNamespace
	}

	recommendation := &analysisv1alph1.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", recommendationRule.Name, strings.ToLower(recommenderName)),
			Namespace:    namespace,
			OwnerReferences: []metav1.OwnerReference{
				*newOwnerRef(recommendationRule),
			},
		},
		Spec: analysisv1alph1.RecommendationSpec{
			TargetRef: target,
			Type:      analysisv1alph1.AnalysisType(recommenderName),
		},
	}

	labels := map[string]string{}
	labels[known.RecommendationRuleNameLabel] = recommendationRule.Name
	labels[known.RecommendationRuleUidLabel] = string(recommendationRule.UID)
	if convert, uid := IsConvertFromAnalytics(recommendationRule); convert {
		labels[known.AnalyticsUidLabel] = uid
	}
	labels[known.RecommendationRuleRecommenderLabel] = recommenderName
	labels[known.RecommendationRuleTargetKindLabel] = target.Kind
	labels[known.RecommendationRuleTargetVersionLabel] = target.GroupVersionKind().Version
	labels[known.RecommendationRuleTargetNameLabel] = target.Name
	for k, v := range id.Labels {
		labels[k] = v
	}

	recommendation.Labels = labels
	return recommendation
}

func executeMission(ctx context.Context, wg *sync.WaitGroup, recommenderMgr recommender.RecommenderManager, provider providers.History, predictorMgr predictormgr.Manager,
	recommendationRule *analysisv1alph1.RecommendationRule, identities map[string]ObjectIdentity, mission *analysisv1alph1.RecommendationMission,
	existingRecommendation *analysisv1alph1.Recommendation, client client.Client, scaleClient scale.ScalesGetter, timeNow metav1.Time, currentRunNumber int32) {
	defer func() {
		mission.LastStartTime = &timeNow
		klog.Infof("Mission message: %s", mission.Message)
		if wg != nil {
			wg.Done()
		}
	}()

	k := objRefKey(mission.TargetRef.Kind, mission.TargetRef.APIVersion, mission.TargetRef.Namespace, mission.TargetRef.Name)
	if id, exist := identities[k]; !exist {
		mission.Message = fmt.Sprintf("Failed to get identity, key %s. ", k)
		return
	} else {
		recommendation := existingRecommendation
		if recommendation == nil {
			recommendation = CreateRecommendationObject(recommendationRule, mission.TargetRef, id, mission.RecommenderRef.Name)
		}

		r, err := recommenderMgr.GetRecommenderWithRule(mission.RecommenderRef.Name, *recommendationRule)
		if err != nil {
			mission.Message = fmt.Sprintf("get recommender %s failed, %v", mission.RecommenderRef.Name, err)
			return
		}
		p := make(map[providers.DataSourceType]providers.History)
		p[providers.PrometheusDataSource] = provider
		identity := framework.ObjectIdentity{
			Namespace:  identities[k].Namespace,
			Name:       identities[k].Name,
			Kind:       identities[k].Kind,
			APIVersion: identities[k].APIVersion,
			Labels:     identities[k].Labels,
			Object:     identities[k].Object,
		}
		recommendationContext := framework.NewRecommendationContext(ctx, identity, recommendationRule, predictorMgr, p, recommendation, client, scaleClient)
		err = recommender.Run(&recommendationContext, r)
		if err != nil {
			mission.Message = fmt.Sprintf("Failed to run recommendation flow in recommender %s: %s", r.Name(), err.Error())
			return
		}

		recommendation.Status.LastUpdateTime = &timeNow
		if recommendation.Annotations == nil {
			recommendation.Annotations = map[string]string{}
		}
		recommendation.Annotations[known.RunNumberAnnotation] = strconv.Itoa(int(currentRunNumber))

		if existingRecommendation != nil {
			klog.Infof("Update recommendation %s", klog.KObj(recommendation))
			if err := client.Update(ctx, recommendation); err != nil {
				mission.Message = fmt.Sprintf("Failed to create recommendation %s: %v", klog.KObj(recommendation), err)
				return
			}

			klog.Infof("Successfully to update Recommendation %s", klog.KObj(recommendation))
		} else {
			klog.Infof("Create recommendation %s", klog.KObj(recommendation))
			if err := client.Create(ctx, recommendation); err != nil {
				mission.Message = fmt.Sprintf("Failed to create recommendation %s: %v", klog.KObj(recommendation), err)
				return
			}

			klog.Infof("Successfully to create Recommendation %s", klog.KObj(recommendation))
		}

		mission.Message = "Success"
		mission.UID = recommendation.UID
		mission.Name = recommendation.Name
		mission.Namespace = recommendation.Namespace
		mission.Kind = recommendation.Kind
		mission.APIVersion = recommendation.APIVersion
	}
}

func IsConvertFromAnalytics(recommendationRule *analysisv1alph1.RecommendationRule) (bool, string) {
	if recommendationRule.Annotations != nil {
		if uid, exist := recommendationRule.Annotations[known.AnalyticsConversionAnnotation]; exist {
			return true, uid
		}
	}

	return false, ""
}
