package analysis

import (
	"context"
	"fmt"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

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
	K8SVersion      *version.Version
}

func (ac *AnalyticsController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ac.Logger.Info("Got an analytics res", "analytics", req.NamespacedName)

	a := &analysisv1alph1.Analytics{}

	err := ac.Client.Get(ctx, req.NamespacedName, a)
	if err != nil {
		return ctrl.Result{}, err
	}

	//var fingerPrints []string

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
		if rs.Name != "" {
			u, err := ac.dynamicClient.Resource(gvr).Namespace(req.Namespace).Get(ctx, rs.Name, metav1.GetOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
			us = append(us, *u)
		} else {
			ul, err := ac.dynamicClient.Resource(gvr).Namespace(req.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return ctrl.Result{}, err
			}
			for _, u := range ul.Items {
				m, ok, err := unstructured.NestedMap(u.Object, "spec", "selector", "matchLabels")
				if !ok || err != nil {
					return ctrl.Result{}, fmt.Errorf("%s not supported", gvr.String())
				}
				matchLabels := map[string]string{}
				for k, v := range m {
					matchLabels[k] = v.(string)
				}
				if match(rs.LabelSelector, matchLabels) {
					us = append(us, u)
				}
			}
		}

		for _, u := range us {
			m, ok, err := unstructured.NestedMap(u.Object, "spec", "selector", "matchLabels")
			if !ok || err != nil {
				return ctrl.Result{}, fmt.Errorf("%s not supported", gvr.String())
			}
			ls := labels.NewSelector()
			for k, v := range m {
				r, err := labels.NewRequirement(k, selection.Equals, []string{v.(string)})
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

			//pod := podList.Items[0]
			//for _, c := range pod.Spec.Containers {
			//	fp := fmt.Sprintf("%s/%s", pod.Namespace, )
			//}
		}

	}

	return ctrl.Result{}, nil
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
