package analysis

import (
	"context"
	"fmt"
	"time"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	craneclient "github.com/gocrane/api/pkg/generated/clientset/versioned"
	analysisinformer "github.com/gocrane/api/pkg/generated/informers/externalversions"
	analysislister "github.com/gocrane/api/pkg/generated/listers/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
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
	Prediction      prediction.Interface
	kubeClient      kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	recommLister    analysislister.RecommendationLister
	K8SVersion      *version.Version
}

func (ac *AnalyticsController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ac.Logger.Info("Got an analytics res", "analytics", req.NamespacedName)

	a := &analysisv1alph1.Analytics{}

	err := ac.Client.Get(ctx, req.NamespacedName, a)
	if err != nil {
		return ctrl.Result{}, err
	}

	identities := map[autoscalingv2.CrossVersionObjectReference]ObjectIdentity{}

	for _, rs := range a.Spec.ResourceSelectors {
		if rs.Kind == "" {
			return ctrl.Result{}, fmt.Errorf("emtpy kind")
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
			m, ok, err := unstructured.NestedStringMap(us[i].Object, "spec", "selector", "matchLabels")
			if !ok || err != nil {
				return ctrl.Result{}, fmt.Errorf("%s not supported", gvr.String())
			}

			ls := labels.NewSelector()
			for k, v := range m {
				r, err := labels.NewRequirement(k, selection.Equals, []string{v})
				if err != nil {
					return ctrl.Result{}, err
				}
				ls.Add(*r)
			}

			opts := metav1.ListOptions{
				LabelSelector: ls.String(),
				Limit:         1,
			}

			podList, err := ac.kubeClient.CoreV1().Pods(req.Namespace).List(ctx, opts)
			if err != nil {
				return ctrl.Result{}, err
			}

			if len(podList.Items) != 1 {
				return ctrl.Result{}, fmt.Errorf("pod not found for %s", gvr.String())
			}

			pod := podList.Items[0]
			objRef := autoscalingv2.CrossVersionObjectReference{
				Kind:       rs.Kind,
				Name:       ownerNames[i],
				APIVersion: rs.APIVersion,
			}
			if _, exists := identities[objRef]; !exists {
				var cs []string
				for _, c := range pod.Spec.Containers {
					cs = append(cs, c.Name)
				}
				identities[objRef] = ObjectIdentity{
					Namespace:  pod.Namespace,
					Name:       objRef.Name,
					Kind:       objRef.Kind,
					APIVersion: objRef.APIVersion,
					Containers: cs,
				}
			}
		}
	}

	rs, err := ac.recommLister.Recommendations(req.Namespace).List(labels.Everything())
	if err != nil {
		return ctrl.Result{}, err
	}

	rm := map[autoscalingv2.CrossVersionObjectReference]*analysisv1alph1.Recommendation{}
	for _, r := range rs {
		rm[r.Spec.TargetRef] = r
	}

	var refs []corev1.ObjectReference

	for ref, id := range identities {
		if r, exists := rm[ref]; exists {
			refs = append(refs, corev1.ObjectReference{
				Kind:       id.Kind,
				Name:       id.Name,
				APIVersion: id.APIVersion,
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
				nr.OwnerReferences = append(nr.OwnerReferences, metav1.OwnerReference{
					APIVersion: id.APIVersion,
					Kind:       id.Kind,
					Name:       id.Name,
				})
				if err = ac.Client.Update(ctx, nr); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			if err = ac.createRecommendations(ctx, a, identities, refs); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	na := a.DeepCopy()
	na.Status.Recommendations = refs
	if err = ac.Client.Update(ctx, na); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (ac *AnalyticsController) createRecommendations(ctx context.Context, a *analysisv1alph1.Analytics,
	identities map[autoscalingv2.CrossVersionObjectReference]ObjectIdentity, refs []corev1.ObjectReference) error {
	for ref := range identities {
		r := &analysisv1alph1.Recommendation{
			Spec: analysisv1alph1.RecommendationSpec{
				TargetRef:       ref,
				Type:            a.Spec.Type,
				IntervalSeconds: a.Spec.IntervalSeconds,
			},
		}
		if err := ac.Create(ctx, r); err != nil {
			return err
		}
		refs = append(refs, corev1.ObjectReference{
			Kind:       ref.Kind,
			Name:       ref.Name,
			APIVersion: ref.APIVersion,
		})
	}
	return nil
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
		Owns(&analysisv1alph1.Recommendation{}).
		Complete(ac)
}

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
	Containers []string
}
