package analysis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	craneclient "github.com/gocrane/api/pkg/generated/clientset/versioned"
	analysisinformer "github.com/gocrane/api/pkg/generated/informers/externalversions"
	analysislister "github.com/gocrane/api/pkg/generated/listers/analysis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AnalyticsController struct {
	client.Client
	Logger          logr.Logger
	Scheme          *runtime.Scheme
	RestMapper      meta.RESTMapper
	Recorder        record.EventRecorder
	kubeClient      kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	recommLister    analysislister.RecommendationLister
	K8SVersion      *version.Version
}

func (ac *AnalyticsController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ac.Logger.V(4).Info("Got an analytics resource.", "analytics", req.NamespacedName)

	a := &analysisv1alph1.Analytics{}

	err := ac.Client.Get(ctx, req.NamespacedName, a)
	if err != nil {
		return ctrl.Result{}, err
	}

	if a.DeletionTimestamp != nil {
		ac.Logger.Info("Analytics resource is being deleted.", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	if a.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical &&
		a.Spec.CompletionStrategy.PeriodSeconds != nil && a.Status.LastSuccessfulTime != nil {
		d := time.Second * time.Duration(*a.Spec.CompletionStrategy.PeriodSeconds)
		if a.Status.LastSuccessfulTime.Add(d).After(time.Now()) {
			return ctrl.Result{}, nil
		}
	}

	identities := map[string]ObjectIdentity{}

	for _, rs := range a.Spec.ResourceSelectors {
		if rs.Kind == "" {
			return ctrl.Result{}, fmt.Errorf("empty kind")
		}

		resList, err := ac.discoveryClient.ServerResourcesForGroupVersion(rs.APIVersion)
		if err != nil {
			return ctrl.Result{}, err
		}

		var resName string
		for _, res := range resList.APIResources {
			if rs.Kind == res.Kind {
				resName = res.Name
				break
			}
		}
		if resName == "" {
			return ctrl.Result{}, fmt.Errorf("invalid kind %s", rs.Kind)
		}

		gv, err := schema.ParseGroupVersion(rs.APIVersion)
		if err != nil {
			return ctrl.Result{}, err
		}
		gvr := gv.WithResource(resName)

		var us []unstructured.Unstructured
		var ownerNames []string

		if rs.Name != "" {
			u, err := ac.dynamicClient.Resource(gvr).Namespace(req.Namespace).Get(ctx, rs.Name, metav1.GetOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
			us = append(us, *u)
			ownerNames = append(ownerNames, rs.Name)
		} else {
			ul, err := ac.dynamicClient.Resource(gvr).Namespace(req.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
			for _, u := range ul.Items {
				m, ok, err := unstructured.NestedStringMap(u.Object, "spec", "selector", "matchLabels")
				if !ok || err != nil {
					return ctrl.Result{}, fmt.Errorf("%s not supported", gvr.String())
				}
				matchLabels := map[string]string{}
				for k, v := range m {
					matchLabels[k] = v
				}
				if match(rs.LabelSelector, matchLabels) {
					us = append(us, u)
					ownerNames = append(ownerNames, u.GetName())
				}
			}
		}

		for i := range us {
			//m, ok, err := unstructured.NestedStringMap(us[i].Object, "spec", "selector", "matchLabels")
			//if !ok || err != nil {
			//	return ctrl.Result{}, fmt.Errorf("%s not supported", gvr.String())
			//}
			//
			//ls := labels.NewSelector()
			//for k, v := range m {
			//	r, err := labels.NewRequirement(k, selection.Equals, []string{v})
			//	if err != nil {
			//		return ctrl.Result{}, err
			//	}
			//	ls = ls.Add(*r)
			//}
			//
			//opts := metav1.ListOptions{
			//	LabelSelector: ls.String(),
			//	Limit:         1,
			//}
			//
			//podList, err := ac.kubeClient.CoreV1().Pods(req.Namespace).List(ctx, opts)
			//if err != nil {
			//	return ctrl.Result{}, err
			//}
			//
			//if len(podList.Items) != 1 {
			//	return ctrl.Result{}, fmt.Errorf("pod not found for %s", gvr.String())
			//}

			k := objRefKey(rs.Kind, rs.APIVersion, ownerNames[i])

			//pod := podList.Items[0]

			if _, exists := identities[k]; !exists {
				//var cs []string
				//for _, c := range pod.Spec.Containers {
				//	cs = append(cs, c.Name)
				//}
				identities[k] = ObjectIdentity{
					Namespace:  req.Namespace,
					Name:       ownerNames[i],
					Kind:       rs.Kind,
					APIVersion: rs.APIVersion,
					//Containers: cs,
				}
			}
		}
	}

	rs, err := ac.recommLister.Recommendations(req.Namespace).List(labels.Everything())
	if err != nil {
		return ctrl.Result{}, err
	}

	rm := map[string]*analysisv1alph1.Recommendation{}
	for _, r := range rs {
		k := objRefKey(r.Spec.TargetRef.Kind, r.Spec.TargetRef.APIVersion, r.Spec.TargetRef.Name)
		rm[k] = r.DeepCopy()
	}

	var refs []corev1.ObjectReference

	for k, id := range identities {
		if r, exists := rm[k]; exists {
			refs = append(refs, corev1.ObjectReference{
				Kind:       rm[k].Kind,
				Name:       rm[k].Name,
				APIVersion: rm[k].APIVersion,
				UID:        rm[k].UID,
			})
			found := false
			for _, or := range r.OwnerReferences {
				if or.Name == id.Name && or.Kind == id.Kind && or.APIVersion == id.APIVersion {
					found = true
					break
				}
			}
			if !found {
				nr := r.DeepCopy()
				nr.OwnerReferences = append(nr.OwnerReferences,
					*metav1.NewControllerRef(a, schema.GroupVersionKind{Version: a.APIVersion, Kind: a.Kind}))
				if err = ac.Update(ctx, nr); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			if err = ac.createRecommendation(ctx, a, id, &refs); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	na := a.DeepCopy()
	na.Status.Recommendations = refs
	t := metav1.Now()
	na.Status.LastSuccessfulTime = &t
	if err = ac.Client.Update(ctx, na); err != nil {
		return ctrl.Result{}, err
	}

	if a.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical {
		if a.Spec.CompletionStrategy.PeriodSeconds != nil {
			d := time.Second * time.Duration(*a.Spec.CompletionStrategy.PeriodSeconds)
			ac.Logger.V(5).Info("Will re-sync", "after", d)
			return ctrl.Result{
				RequeueAfter: d,
			}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (ac *AnalyticsController) createRecommendation(ctx context.Context, a *analysisv1alph1.Analytics,
	id ObjectIdentity, refs *[]corev1.ObjectReference) error {

	r := &analysisv1alph1.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-%s-%s",
				a.Name, strings.ToLower(id.Kind), strings.ReplaceAll(id.APIVersion, "/", "-"), id.Name),
			Namespace: a.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(a, schema.GroupVersionKind{Version: a.APIVersion, Kind: a.Kind}),
			},
		},
		Spec: analysisv1alph1.RecommendationSpec{
			TargetRef:          corev1.ObjectReference{Kind: id.Kind, APIVersion: id.APIVersion, Name: id.Name},
			Type:               a.Spec.Type,
			CompletionStrategy: a.Spec.CompletionStrategy,
		},
	}

	if err := ac.Create(ctx, r); err != nil {
		ac.Logger.Error(err, "Failed to create Recommendation")
		return err
	}

	if err := ac.Get(ctx, types.NamespacedName{Namespace: r.Namespace, Name: r.Name}, r); err != nil {
		ac.Logger.Error(err, "Failed to get Recommendation")
		return err
	}
	*refs = append(*refs, corev1.ObjectReference{
		Kind:       r.Kind,
		Name:       r.Name,
		APIVersion: r.APIVersion,
		UID:        r.UID,
	})

	return nil
}

func objRefKey(kind, apiVersion, name string) string {
	return fmt.Sprintf("%s#%s#%s", kind, apiVersion, name)
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

func (ac *AnalyticsController) SetupWithManager(mgr ctrl.Manager) error {
	ac.kubeClient = kubernetes.NewForConfigOrDie(mgr.GetConfig())

	ac.discoveryClient = discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig())

	ac.dynamicClient = dynamic.NewForConfigOrDie(mgr.GetConfig())

	cli := craneclient.NewForConfigOrDie(mgr.GetConfig())
	fact := analysisinformer.NewSharedInformerFactory(cli, time.Second*30)
	ac.recommLister = fact.Analysis().V1alpha1().Recommendations().Lister()

	fact.Start(nil)
	if ok := cache.WaitForCacheSync(nil, fact.Analysis().V1alpha1().Recommendations().Informer().HasSynced); !ok {
		return fmt.Errorf("failed to sync")
	}

	serverVersion, err := ac.discoveryClient.ServerVersion()
	if err != nil {
		return err
	}
	ac.K8SVersion = version.MustParseGeneric(serverVersion.GitVersion)

	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Analytics{}).
		Complete(ac)
}

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
	//Containers []string
}
