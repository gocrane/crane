package analytics

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/yaml"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	craneclient "github.com/gocrane/api/pkg/generated/clientset/versioned"
	analysisinformer "github.com/gocrane/api/pkg/generated/informers/externalversions"
	analysislister "github.com/gocrane/api/pkg/generated/listers/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommend"
)

type Controller struct {
	client.Client
	Scheme          *runtime.Scheme
	RestMapper      meta.RESTMapper
	Recorder        record.EventRecorder
	kubeClient      kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	recommLister    analysislister.RecommendationLister
	K8SVersion      *version.Version
	ScaleClient     scale.ScalesGetter
	PredictorMgr    predictormgr.Manager
	Provider        providers.History
	ConfigSetFile   string
	configSet       *analysisv1alph1.ConfigSet
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).InfoS("Got an analytics resource.", "analytics", req.NamespacedName)

	analytics := &analysisv1alph1.Analytics{}

	err := c.Client.Get(ctx, req.NamespacedName, analytics)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if analytics.DeletionTimestamp != nil {
		klog.InfoS("Analytics resource is being deleted.", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	lastUpdateTime := analytics.Status.LastUpdateTime
	if analytics.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyOnce {
		if lastUpdateTime != nil {
			// This is a one-off analytics task which has been completed.
			return ctrl.Result{}, nil
		}
	} else {
		if lastUpdateTime != nil {
			planingTime := lastUpdateTime.Add(time.Duration(*analytics.Spec.CompletionStrategy.PeriodSeconds) * time.Second)
			now := time.Now()
			if now.Before(planingTime) {
				return ctrl.Result{
					RequeueAfter: planingTime.Sub(now),
				}, nil
			}
		}
	}

	finished := c.analyze(ctx, analytics)

	if finished && analytics.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical {
		if analytics.Spec.CompletionStrategy.PeriodSeconds != nil {
			d := time.Second * time.Duration(*analytics.Spec.CompletionStrategy.PeriodSeconds)
			klog.V(4).InfoS("Will re-sync", "after", d)
			// Arrange for next round.
			return ctrl.Result{
				RequeueAfter: d,
			}, nil
		}
	}

	klog.V(6).Infof("Analytics not finished, continue to do it.")
	return ctrl.Result{RequeueAfter: time.Second * 1}, nil
}

func (c *Controller) analyze(ctx context.Context, analytics *analysisv1alph1.Analytics) bool {
	newStatus := analytics.Status.DeepCopy()

	identities, err := c.getIdentities(ctx, analytics)
	if err != nil {
		c.Recorder.Event(analytics, corev1.EventTypeNormal, "FailedSelectResource", err.Error())
		msg := fmt.Sprintf("Failed to get idenitities, Analytics %s error %v", klog.KObj(analytics), err)
		klog.Errorf(msg)
		setReadyCondition(newStatus, metav1.ConditionFalse, "FailedSelectResource", msg)
		c.UpdateStatus(ctx, analytics, newStatus)
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
			planingTime := firstMissionStartTime.Add(time.Duration(*analytics.Spec.CompletionStrategy.PeriodSeconds) * time.Second)
			if time.Now().After(planingTime) {
				currMissions = nil // reset missions to trigger creation for missions
			}
		}
	}

	if currMissions == nil {
		// create recommendation missions for this round
		for _, id := range identities {
			currMissions = append(currMissions, analysisv1alph1.RecommendationMission{
				TargetRef: corev1.ObjectReference{Kind: id.Kind, APIVersion: id.APIVersion, Namespace: id.Namespace, Name: id.Name},
			})
		}
	}

	var currRecommendations []*analysisv1alph1.Recommendation
	labelSet := labels.Set{}
	labelSet[known.AnalyticsUidLabel] = string(analytics.UID)
	klog.V(4).Infof("List current recommendations, name %s ns %s selector %s.", analytics.Name, analytics.Namespace, labelSet.String())
	currRecommendations, err = c.recommLister.Recommendations(analytics.Namespace).List(labels.SelectorFromSet(labelSet))
	klog.V(4).Infof("List current recommendations result length %d, error %v", len(currRecommendations), err)
	if err != nil {
		c.Recorder.Event(analytics, corev1.EventTypeNormal, "FailedSelectResource", err.Error())
		msg := fmt.Sprintf("Failed to get recomendations, Analytics %s error %v", klog.KObj(analytics), err)
		klog.Errorf(msg)
		setReadyCondition(newStatus, metav1.ConditionFalse, "FailedSelectResource", msg)
		c.UpdateStatus(ctx, analytics, newStatus)
		return false
	}

	if klog.V(6).Enabled() {
		// Print identities
		for k, id := range identities {
			klog.V(6).InfoS("identities", "analytics", klog.KObj(analytics), "key", k, "apiVersion", id.APIVersion, "kind", id.Kind, "namespace", id.Namespace, "name", id.Name)
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
		for _, r := range currRecommendations {
			if reflect.DeepEqual(currMissions[index].TargetRef, r.Spec.TargetRef) {
				existingRecommendation = r
				break
			}
		}

		go c.executeMission(ctx, &wg, analytics, identities, &currMissions[index], existingRecommendation, timeNow)
	}

	wg.Wait()

	finished := false
	if executionIndex+concurrency == len(currMissions) || len(currMissions) == 0 {
		finished = true
	}

	if finished {
		newStatus.LastUpdateTime = &timeNow

		// clean orphan recommendations
		for _, recommendation := range currRecommendations {
			exist := false
			for _, mission := range currMissions {
				if recommendation.UID == mission.UID {
					exist = true
					break
				}
			}

			if !exist {
				err = c.Client.Delete(ctx, recommendation)
				if err != nil {
					klog.ErrorS(err, "Failed to delete recommendation.", "recommendation", klog.KObj(recommendation))
				} else {
					klog.Infof("Deleted orphan recommendation %v.", klog.KObj(recommendation))
				}
			}
		}

	}

	newStatus.Recommendations = currMissions
	setReadyCondition(newStatus, metav1.ConditionTrue, "AnalyticsReady", "Analytics is ready")

	c.UpdateStatus(ctx, analytics, newStatus)
	return finished
}

func (c *Controller) CreateRecommendationObject(ctx context.Context, analytics *analysisv1alph1.Analytics,
	target corev1.ObjectReference, id ObjectIdentity) *analysisv1alph1.Recommendation {

	recommendation := &analysisv1alph1.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", analytics.Name, strings.ToLower(string(analytics.Spec.Type))),
			Namespace:    analytics.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*newOwnerRef(analytics),
			},
			Labels: id.Labels,
		},
		Spec: analysisv1alph1.RecommendationSpec{
			TargetRef: target,
			Type:      analytics.Spec.Type,
		},
	}

	labels := map[string]string{}
	labels[known.AnalyticsNameLabel] = analytics.Name
	labels[known.AnalyticsUidLabel] = string(analytics.UID)
	labels[known.AnalyticsTypeLabel] = string(analytics.Spec.Type)
	for k, v := range id.Labels {
		labels[k] = v
	}

	recommendation.Labels = labels

	return recommendation
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	c.kubeClient = kubernetes.NewForConfigOrDie(mgr.GetConfig())

	c.discoveryClient = discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig())

	c.dynamicClient = dynamic.NewForConfigOrDie(mgr.GetConfig())

	cli := craneclient.NewForConfigOrDie(mgr.GetConfig())
	fact := analysisinformer.NewSharedInformerFactory(cli, time.Second*30)
	c.recommLister = fact.Analysis().V1alpha1().Recommendations().Lister()

	fact.Start(nil)
	if ok := cache.WaitForCacheSync(nil, fact.Analysis().V1alpha1().Recommendations().Informer().HasSynced); !ok {
		return fmt.Errorf("failed to sync")
	}

	serverVersion, err := c.discoveryClient.ServerVersion()
	if err != nil {
		return err
	}
	c.K8SVersion = version.MustParseGeneric(serverVersion.GitVersion)

	if err = c.loadConfigSetFile(); err != nil {
		return err
	}

	go c.watchConfigSetFile()

	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Analytics{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(c)
}

func (c *Controller) getIdentities(ctx context.Context, analytics *analysisv1alph1.Analytics) (map[string]ObjectIdentity, error) {
	identities := map[string]ObjectIdentity{}

	for _, rs := range analytics.Spec.ResourceSelectors {
		if rs.Kind == "" {
			return nil, fmt.Errorf("empty kind")
		}

		resList, err := c.discoveryClient.ServerResourcesForGroupVersion(rs.APIVersion)
		if err != nil {
			return nil, err
		}

		var resName string
		for _, res := range resList.APIResources {
			if rs.Kind == res.Kind {
				resName = res.Name
				break
			}
		}
		if resName == "" {
			return nil, fmt.Errorf("invalid kind %s", rs.Kind)
		}

		gv, err := schema.ParseGroupVersion(rs.APIVersion)
		if err != nil {
			return nil, err
		}
		gvr := gv.WithResource(resName)

		var unstructureds []unstructured.Unstructured

		if rs.Name != "" {
			unstructured, err := c.dynamicClient.Resource(gvr).Namespace(analytics.Namespace).Get(ctx, rs.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			unstructureds = append(unstructureds, *unstructured)
		} else {
			var unstructuredList *unstructured.UnstructuredList
			var err error

			if analytics.Namespace == known.CraneSystemNamespace {
				unstructuredList, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
			} else {
				unstructuredList, err = c.dynamicClient.Resource(gvr).Namespace(analytics.Namespace).List(ctx, metav1.ListOptions{})
			}
			if err != nil {
				return nil, err
			}

			for _, item := range unstructuredList.Items {
				// todo: rename rs.LabelSelector to rs.matchLabelSelector ?
				m, ok, err := unstructured.NestedStringMap(item.Object, "spec", "selector", "matchLabels")
				if !ok || err != nil {
					return nil, fmt.Errorf("%s not supported", gvr.String())
				}
				matchLabels := map[string]string{}
				for k, v := range m {
					matchLabels[k] = v
				}
				if match(rs.LabelSelector, matchLabels) {
					unstructureds = append(unstructureds, item)
				}
			}
		}

		for i := range unstructureds {
			k := objRefKey(rs.Kind, rs.APIVersion, unstructureds[i].GetNamespace(), unstructureds[i].GetName(), string(analytics.Spec.Type))
			if _, exists := identities[k]; !exists {
				identities[k] = ObjectIdentity{
					Namespace:  unstructureds[i].GetNamespace(),
					Name:       unstructureds[i].GetName(),
					Kind:       rs.Kind,
					APIVersion: rs.APIVersion,
					Labels:     unstructureds[i].GetLabels(),
				}
			}
		}
	}

	return identities, nil
}

func (c *Controller) executeMission(ctx context.Context, wg *sync.WaitGroup, analytics *analysisv1alph1.Analytics, identities map[string]ObjectIdentity, mission *analysisv1alph1.RecommendationMission, existingRecommendation *analysisv1alph1.Recommendation, timeNow metav1.Time) {
	defer func() {
		mission.LastStartTime = &timeNow
		klog.Infof("Mission message: %s", mission.Message)

		wg.Done()
	}()

	k := objRefKey(mission.TargetRef.Kind, mission.TargetRef.APIVersion, mission.TargetRef.Namespace, mission.TargetRef.Name, string(analytics.Spec.Type))
	if id, exist := identities[k]; !exist {
		mission.Message = fmt.Sprintf("Failed to get identity, key %s. ", k)
		return
	} else {
		recommendation := existingRecommendation
		if recommendation == nil {
			recommendation = c.CreateRecommendationObject(ctx, analytics, mission.TargetRef, id)
		}
		// do recommendation
		recommender, err := recommend.NewRecommender(c.Client, c.RestMapper, c.ScaleClient, recommendation, c.PredictorMgr, c.Provider, c.configSet, analytics.Spec.Config)
		if err != nil {
			mission.Message = fmt.Sprintf("Failed to create recommender, Recommendation %s error %v", klog.KObj(recommendation), err)
			return
		}

		proposed, err := recommender.Offer()
		if err != nil {
			mission.Message = fmt.Sprintf("Failed to offer recommendation: %s", err.Error())
			return
		}

		var value string
		valueBytes, err := yaml.Marshal(proposed)
		if err != nil {
			mission.Message = err.Error()
			return
		}
		value = string(valueBytes)

		recommendation.Status.RecommendedValue = value
		recommendation.Status.LastUpdateTime = &timeNow
		if existingRecommendation != nil {
			klog.Infof("Update recommendation %s", klog.KObj(recommendation))
			if err := c.Update(ctx, recommendation); err != nil {
				mission.Message = fmt.Sprintf("Failed to create recommendation %s: %v", klog.KObj(recommendation), err)
				return
			}

			klog.Infof("Successfully to update Recommendation %s", klog.KObj(recommendation))
		} else {
			klog.Infof("Create recommendation %s", klog.KObj(recommendation))
			if err := c.Create(ctx, recommendation); err != nil {
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

func (c *Controller) UpdateStatus(ctx context.Context, analytics *analysisv1alph1.Analytics, newStatus *analysisv1alph1.AnalyticsStatus) {
	if !equality.Semantic.DeepEqual(&analytics.Status, newStatus) {
		analytics.Status = *newStatus
		err := c.Update(ctx, analytics)
		if err != nil {
			c.Recorder.Event(analytics, corev1.EventTypeNormal, "FailedUpdateStatus", err.Error())
			klog.Errorf("Failed to update status, Analytics %s error %v", klog.KObj(analytics), err)
			return
		}

		klog.Infof("Update Analytics status successful, Analytics %s", klog.KObj(analytics))
	}
}

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
	Labels     map[string]string
}

func newOwnerRef(a *analysisv1alph1.Analytics) *metav1.OwnerReference {
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

func objRefKey(kind, apiVersion, namespace, name, recommendType string) string {
	return fmt.Sprintf("%s#%s#%s#%s#%s", kind, apiVersion, namespace, name, recommendType)
}

func match(labelSelector metav1.LabelSelector, matchLabels map[string]string) bool {
	for k, v := range labelSelector.MatchLabels {
		if matchLabels[k] != v {
			return false
		}
	}

	for _, expr := range labelSelector.MatchExpressions {
		switch expr.Operator {
		case metav1.LabelSelectorOpExists:
			if _, exists := matchLabels[expr.Key]; !exists {
				return false
			}
		case metav1.LabelSelectorOpDoesNotExist:
			if _, exists := matchLabels[expr.Key]; exists {
				return false
			}
		case metav1.LabelSelectorOpIn:
			if v, exists := matchLabels[expr.Key]; !exists {
				return false
			} else {
				var found bool
				for i := range expr.Values {
					if expr.Values[i] == v {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		case metav1.LabelSelectorOpNotIn:
			if v, exists := matchLabels[expr.Key]; exists {
				for i := range expr.Values {
					if expr.Values[i] == v {
						return false
					}
				}
			}
		}
	}

	return true
}

func setReadyCondition(status *analysisv1alph1.AnalyticsStatus, conditionStatus metav1.ConditionStatus, reason string, message string) {
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
