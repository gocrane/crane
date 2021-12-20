package analytics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	craneclient "github.com/gocrane/api/pkg/generated/clientset/versioned"
	analysisinformer "github.com/gocrane/api/pkg/generated/informers/externalversions"
	analysislister "github.com/gocrane/api/pkg/generated/listers/analysis/v1alpha1"
)

type Controller struct {
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

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Logger.V(4).Info("Got an analytics resource.", "analytics", req.NamespacedName)

	a := &analysisv1alph1.Analytics{}

	err := c.Client.Get(ctx, req.NamespacedName, a)
	if err != nil {
		return ctrl.Result{}, err
	}

	if a.DeletionTimestamp != nil {
		c.Logger.Info("Analytics resource is being deleted.", "name", req.NamespacedName)
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

		resList, err := c.discoveryClient.ServerResourcesForGroupVersion(rs.APIVersion)
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
			u, err := c.dynamicClient.Resource(gvr).Namespace(req.Namespace).Get(ctx, rs.Name, metav1.GetOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
			us = append(us, *u)
			ownerNames = append(ownerNames, rs.Name)
		} else {
			ul, err := c.dynamicClient.Resource(gvr).Namespace(req.Namespace).List(ctx, metav1.ListOptions{})
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
			k := objRefKey(rs.Kind, rs.APIVersion, ownerNames[i])
			if _, exists := identities[k]; !exists {
				identities[k] = ObjectIdentity{
					Namespace:  req.Namespace,
					Name:       ownerNames[i],
					Kind:       rs.Kind,
					APIVersion: rs.APIVersion,
				}
			}
		}
	}

	rs, err := c.recommLister.Recommendations(req.Namespace).List(labels.Everything())
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
				if err = c.Update(ctx, nr); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			if err = c.createRecommendation(ctx, a, id, &refs); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	na := a.DeepCopy()
	na.Status.Recommendations = refs
	t := metav1.Now()
	na.Status.LastSuccessfulTime = &t
	if err = c.Client.Update(ctx, na); err != nil {
		return ctrl.Result{}, err
	}

	if a.Spec.CompletionStrategy.CompletionStrategyType == analysisv1alph1.CompletionStrategyPeriodical {
		if a.Spec.CompletionStrategy.PeriodSeconds != nil {
			d := time.Second * time.Duration(*a.Spec.CompletionStrategy.PeriodSeconds)
			c.Logger.V(5).Info("Will re-sync", "after", d)
			return ctrl.Result{
				RequeueAfter: d,
			}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (ac *Controller) createRecommendation(ctx context.Context, a *analysisv1alph1.Analytics,
	id ObjectIdentity, refs *[]corev1.ObjectReference) error {

	r := &analysisv1alph1.Recommendation{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", a.Name, strings.ToLower(string(a.Spec.Type))),
			Namespace:    a.Namespace,
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

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
}
