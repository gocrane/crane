package recommendation

import (
	"context"
	"fmt"
	"github.com/gocrane/crane/pkg/metrics"
	"sort"
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
	"github.com/gocrane/crane/pkg/oom"
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
	OOMRecorder     oom.Recorder
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

	var identitiesArray []ObjectIdentity
	keys := make([]string, 0, len(identities))
	for k := range identities {
		keys = append(keys, k)
	}
	sort.Strings(keys) // sort key to get a certain order
	for _, key := range keys {
		id := identities[key]
		id.Recommendation = GetRecommendationFromIdentity(identities[key], currRecommendations)
		identitiesArray = append(identitiesArray, id)
	}

	timeNow := metav1.Now()
	newRound := false
	if len(identitiesArray) > 0 {
		firstRecommendation := identitiesArray[0].Recommendation
		firstMissionStartTime, err := utils.GetLastStartTime(firstRecommendation)
		if err != nil {
			newRound = true
		} else {
			planingTime := firstMissionStartTime.Add(interval)
			now := utils.NowUTC()
			if now.After(planingTime) {
				newRound = true
			}
		}
	}

	if newRound {
		// +1 for runNumber
		newStatus.RunNumber = newStatus.RunNumber + 1
	}

	maxConcurrency := 10
	executionIndex := -1
	var concurrency int
	for index, identity := range identitiesArray {
		if identity.Recommendation != nil {
			runNumber, _ := utils.GetRunNumber(identity.Recommendation)
			if runNumber >= newStatus.RunNumber {
				continue
			}
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
	for index := executionIndex; index < len(identitiesArray) && index < concurrency+executionIndex; index++ {
		if klog.V(6).Enabled() {
			klog.V(6).InfoS("execute identities", "RecommendationRule", klog.KObj(recommendationRule), "target", identitiesArray[index].GetObjectReference())
		}
		go executeIdentity(ctx, &wg, c.RecommenderMgr, c.Provider, c.PredictorMgr, recommendationRule, identitiesArray[index], c.Client, c.ScaleClient, c.OOMRecorder, timeNow, newStatus.RunNumber)
	}

	wg.Wait()

	finished := false
	if executionIndex+concurrency == len(identitiesArray) || len(identitiesArray) == 0 {
		finished = true
	}

	if finished {
		newStatus.LastUpdateTime = &timeNow
		// clean orphan recommendations
		for _, recommendation := range currRecommendations.Items {
			exist := false
			for _, id := range identitiesArray {
				if recommendation.UID == id.Recommendation.UID {
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
			for _, recommender := range recommendationRule.Spec.Recommenders {
				k := objRefKey(rs.Kind, rs.APIVersion, filterdUnstructureds[i].GetNamespace(), filterdUnstructureds[i].GetName(), recommender.Name)
				if _, exists := identities[k]; !exists {
					identities[k] = ObjectIdentity{
						Namespace:   filterdUnstructureds[i].GetNamespace(),
						Name:        filterdUnstructureds[i].GetName(),
						Kind:        rs.Kind,
						APIVersion:  rs.APIVersion,
						Labels:      filterdUnstructureds[i].GetLabels(),
						Object:      filterdUnstructureds[i],
						Recommender: recommender.Name,
					}
				}
			}
		}
	}

	for _, id := range identities {
		metrics.SelectTargets.With(map[string]string{
			"type":       id.Recommender,
			"apiversion": id.APIVersion,
			"owner_kind": id.Kind,
			"namespace":  id.Namespace,
			"owner_name": id.Name,
		}).Set(1)
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
	Namespace      string
	APIVersion     string
	Kind           string
	Name           string
	Labels         map[string]string
	Recommender    string
	Object         unstructuredv1.Unstructured
	Recommendation *analysisv1alph1.Recommendation
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

func objRefKey(kind, apiVersion, namespace, name, recommender string) string {
	return fmt.Sprintf("%s#%s#%s#%s#%s", kind, apiVersion, namespace, name, recommender)
}

func GetRecommendationFromIdentity(id ObjectIdentity, currRecommendations analysisv1alph1.RecommendationList) *analysisv1alph1.Recommendation {
	for _, r := range currRecommendations.Items {
		if id.Kind == r.Spec.TargetRef.Kind &&
			id.APIVersion == r.Spec.TargetRef.APIVersion &&
			id.Namespace == r.Spec.TargetRef.Namespace &&
			id.Name == r.Spec.TargetRef.Name &&
			id.Recommender == string(r.Spec.Type) {
			return &r
		}
	}

	return nil
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

	recommendation.Labels = generateRecommendationLabels(recommendationRule, target, id, recommenderName)
	return recommendation
}

func generateRecommendationLabels(recommendationRule *analysisv1alph1.RecommendationRule, target corev1.ObjectReference, id ObjectIdentity, recommenderName string) map[string]string {
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
	labels[known.RecommendationRuleTargetNamespaceLabel] = target.Namespace
	for k, v := range id.Labels {
		labels[k] = v
	}

	return labels
}

func executeIdentity(ctx context.Context, wg *sync.WaitGroup, recommenderMgr recommender.RecommenderManager, provider providers.History, predictorMgr predictormgr.Manager,
	recommendationRule *analysisv1alph1.RecommendationRule, id ObjectIdentity, client client.Client, scaleClient scale.ScalesGetter, oomRecorder oom.Recorder, timeNow metav1.Time, currentRunNumber int32) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()
	var message string

	recommendation := id.Recommendation
	if recommendation == nil {
		recommendation = CreateRecommendationObject(recommendationRule, id.GetObjectReference(), id, id.Recommender)
	} else {
		// update existing recommendation's labels
		for k, v := range generateRecommendationLabels(recommendationRule, id.GetObjectReference(), id, id.Recommender) {
			recommendation.Labels[k] = v
		}
	}

	r, err := recommenderMgr.GetRecommenderWithRule(id.Recommender, *recommendationRule)
	if err != nil {
		message = fmt.Sprintf("get recommender %s failed, %v", id.Recommender, err)
	} else {
		p := make(map[providers.DataSourceType]providers.History)
		p[providers.PrometheusDataSource] = provider
		identity := framework.ObjectIdentity{
			Namespace:  id.Namespace,
			Name:       id.Name,
			Kind:       id.Kind,
			APIVersion: id.APIVersion,
			Labels:     id.Labels,
			Object:     id.Object,
		}
		recommendationContext := framework.NewRecommendationContext(ctx, identity, recommendationRule, predictorMgr, p, recommendation, client, scaleClient, oomRecorder)
		err = recommender.Run(&recommendationContext, r)
		if err != nil {
			message = fmt.Sprintf("Failed to run recommendation flow in recommender %s: %s", r.Name(), err.Error())
		}
	}

	if len(message) == 0 {
		message = "Success"
	}

	recommendation.Status.LastUpdateTime = &timeNow
	if recommendation.Annotations == nil {
		recommendation.Annotations = map[string]string{}
	}
	recommendation.Annotations[known.RunNumberAnnotation] = strconv.Itoa(int(currentRunNumber))
	recommendation.Annotations[known.MessageAnnotation] = message
	utils.SetLastStartTime(recommendation)

	if id.Recommendation != nil {
		klog.Infof("Update recommendation %s", klog.KObj(recommendation))
		if err := client.Update(ctx, recommendation); err != nil {
			klog.Errorf("Failed to create recommendation %s: %v", klog.KObj(recommendation), err)
			return
		}

		klog.Infof("Successfully to update Recommendation %s", klog.KObj(recommendation))
	} else {
		klog.Infof("Create recommendation %s", klog.KObj(recommendation))
		if err := client.Create(ctx, recommendation); err != nil {
			klog.Errorf("Failed to create recommendation %s: %v", klog.KObj(recommendation), err)
			return
		}

		klog.Infof("Successfully to create Recommendation %s", klog.KObj(recommendation))
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
