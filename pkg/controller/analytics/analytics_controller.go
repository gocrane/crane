package analytics

import (
	"context"
	"fmt"
	"strings"
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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	craneclient "github.com/gocrane/api/pkg/generated/clientset/versioned"
	analysisinformer "github.com/gocrane/api/pkg/generated/informers/externalversions"
	analysislister "github.com/gocrane/api/pkg/generated/listers/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
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

	shouldAnalytics := c.ShouldAnalytics(analytics)
	if !shouldAnalytics {
		klog.V(4).Infof("Nothing happens for Analytics %s", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	c.DoAnalytics(ctx, analytics)

	if analytics.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical {
		if analytics.Spec.CompletionStrategy.PeriodSeconds != nil {
			d := time.Second * time.Duration(*analytics.Spec.CompletionStrategy.PeriodSeconds)
			klog.V(4).InfoS("Will re-sync", "after", d)
			return ctrl.Result{
				RequeueAfter: d,
			}, nil
		}
	}

	return ctrl.Result{}, nil
}

// ShouldAnalytics decide if we need do analytics according to status
func (c *Controller) ShouldAnalytics(analytics *analysisv1alph1.Analytics) bool {
	lastUpdateTime := analytics.Status.LastUpdateTime

	if analytics.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyOnce {
		if lastUpdateTime != nil {
			// already finish analytics
			return false
		}
	} else {
		if lastUpdateTime != nil {
			planingTime := lastUpdateTime.Add(time.Duration(*analytics.Spec.CompletionStrategy.PeriodSeconds) * time.Second)
			if time.Now().Before(planingTime) {
				return false
			}
		}
	}

	return true
}

func (c *Controller) DoAnalytics(ctx context.Context, analytics *analysisv1alph1.Analytics) {
	newStatus := analytics.Status.DeepCopy()

	identities, err := c.GetIdentities(ctx, analytics)
	if err != nil {
		c.Recorder.Event(analytics, corev1.EventTypeNormal, "FailedSelectResource", err.Error())
		msg := fmt.Sprintf("Failed to get idenitities, Analytics %s error %v", klog.KObj(analytics), err)
		klog.Errorf(msg)
		setReadyCondition(newStatus, metav1.ConditionFalse, "FailedSelectResource", msg)
		c.UpdateStatus(ctx, analytics, newStatus)
		return
	}

	var recommendations []*analysisv1alph1.Recommendation
	if analytics.Namespace == known.CraneSystemNamespace {
		recommendations, err = c.recommLister.List(labels.Everything())
	} else {
		recommendations, err = c.recommLister.Recommendations(analytics.Namespace).List(labels.Everything())
	}
	if err != nil {
		c.Recorder.Event(analytics, corev1.EventTypeNormal, "FailedSelectResource", err.Error())
		msg := fmt.Sprintf("Failed to get recomendations, Analytics %s error %v", klog.KObj(analytics), err)
		klog.Errorf(msg)
		setReadyCondition(newStatus, metav1.ConditionFalse, "FailedSelectResource", msg)
		c.UpdateStatus(ctx, analytics, newStatus)
		return
	}

	recommendationMap := map[string]*analysisv1alph1.Recommendation{}
	for _, r := range recommendations {
		k := objRefKey(r.Spec.TargetRef.Kind, r.Spec.TargetRef.APIVersion, r.Spec.TargetRef.Namespace, r.Spec.TargetRef.Name, string(r.Spec.Type))
		recommendationMap[k] = r.DeepCopy()
	}

	if klog.V(6).Enabled() {
		// Print recommendations
		for k, r := range recommendationMap {
			klog.V(6).InfoS("recommendations", "analytics", klog.KObj(analytics), "key", k, "namespace", r.Namespace, "name", r.Name)
		}
		// Print identities
		for k, id := range identities {
			klog.V(6).InfoS("identities", "analytics", klog.KObj(analytics), "key", k, "apiVersion", id.APIVersion, "kind", id.Kind, "namespace", id.Namespace, "name", id.Name)
		}
	}

	var refs []analysisv1alph1.RecommendationReference

	for k, id := range identities {
		if r, exists := recommendationMap[k]; exists {
			refs = append(refs, analysisv1alph1.RecommendationReference{
				ObjectReference: corev1.ObjectReference{
					Kind:       recommendationMap[k].Kind,
					Name:       recommendationMap[k].Name,
					Namespace:  recommendationMap[k].Namespace,
					APIVersion: recommendationMap[k].APIVersion,
					UID:        recommendationMap[k].UID,
				},
				TargetRef: recommendationMap[k].Spec.TargetRef,
			})
			found := false
			for _, or := range r.OwnerReferences {
				if or.Name == analytics.Name && or.Kind == analytics.Kind && or.APIVersion == analytics.APIVersion {
					found = true
					break
				}
			}
			if !found {
				rCopy := r.DeepCopy()
				rCopy.OwnerReferences = append(rCopy.OwnerReferences, *newOwnerRef(analytics))
				if err = c.Update(ctx, rCopy); err != nil {
					c.Recorder.Event(analytics, corev1.EventTypeNormal, "FailedUpdateRecommendation", err.Error())
					msg := fmt.Sprintf("Failed to update ownerReferences for recommendation %s, Analytics %s error %v", klog.KObj(rCopy), klog.KObj(analytics), err)
					klog.Errorf(msg)
					setReadyCondition(newStatus, metav1.ConditionFalse, "FailedUpdateRecommendation", msg)
					c.UpdateStatus(ctx, analytics, newStatus)
					return
				}
				klog.InfoS("Successful to update ownerReferences", "Recommendation", rCopy, "Analytics", analytics)
			}
		} else {
			if err = c.CreateRecommendation(ctx, analytics, id, &refs); err != nil {
				c.Recorder.Event(analytics, corev1.EventTypeNormal, "FailedCreateRecommendation", err.Error())
				msg := fmt.Sprintf("Failed to create recommendation, Analytics %s error %v", klog.KObj(analytics), err)
				klog.Errorf(msg)
				setReadyCondition(newStatus, metav1.ConditionFalse, "FailedCreateRecommendation", msg)
				c.UpdateStatus(ctx, analytics, newStatus)
				return
			}
		}
	}
	newStatus.Recommendations = refs
	timeNow := metav1.Now()
	newStatus.LastUpdateTime = &timeNow
	setReadyCondition(newStatus, metav1.ConditionTrue, "AnalyticsReady", "Analytics is ready")

	c.UpdateStatus(ctx, analytics, newStatus)
}

func (c *Controller) CreateRecommendation(ctx context.Context, analytics *analysisv1alph1.Analytics,
	id ObjectIdentity, refs *[]analysisv1alph1.RecommendationReference) error {

	targetRef := corev1.ObjectReference{Kind: id.Kind, APIVersion: id.APIVersion, Namespace: id.Namespace, Name: id.Name}
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
			TargetRef:          targetRef,
			Type:               analytics.Spec.Type,
			CompletionStrategy: analytics.Spec.CompletionStrategy,
		},
	}

	if err := c.Create(ctx, recommendation); err != nil {
		klog.Error(err, "Failed to create Recommendation")
		return err
	}

	klog.InfoS("Successful to create", "Recommendation", klog.KObj(recommendation), "Analytics", klog.KObj(analytics))

	*refs = append(*refs, analysisv1alph1.RecommendationReference{
		ObjectReference: corev1.ObjectReference{
			Kind:       recommendation.Kind,
			Name:       recommendation.Name,
			Namespace:  recommendation.Namespace,
			APIVersion: recommendation.APIVersion,
			UID:        recommendation.UID,
		},
		TargetRef: targetRef,
	})

	return nil
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Analytics{}).
		Complete(c)
}

func (c *Controller) GetIdentities(ctx context.Context, analytics *analysisv1alph1.Analytics) (map[string]ObjectIdentity, error) {
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
