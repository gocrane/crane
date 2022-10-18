package ehpa

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/prometheus-adapter/pkg/config"
)

// controller for configMap of prometheus-adapter
type PromAdapterConfigMapController struct {
	client.Client
	Scheme         *runtime.Scheme
	RestMapper     meta.RESTMapper
	Recorder       record.EventRecorder
	ConfigMap      string
	EhpaController *EffectiveHPAController
}

type PromAdapterConfigMapChangedPredicate struct {
	predicate.Funcs
	Name      string
	Namespace string
}

func (pc *PromAdapterConfigMapController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cmArray = strings.Split(pc.ConfigMap, "/")

	if len(cmArray) != 3 {
		return ctrl.Result{}, fmt.Errorf("configmap %s set error", req.NamespacedName)
	}
	cmNamespace := cmArray[0]
	cmName := cmArray[1]
	cmKey := cmArray[2]

	if req.NamespacedName.String() != cmNamespace+"/"+cmName {
		return ctrl.Result{}, fmt.Errorf("configmap %s not matched", req.NamespacedName)
	}
	klog.V(4).Infof("Got prometheus adapter configmap %s", req.NamespacedName)

	//get configmap content
	cm := &corev1.ConfigMap{}
	err := pc.Client.Get(ctx, req.NamespacedName, cm)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cm == nil {
		return ctrl.Result{}, fmt.Errorf("get configmap %s failed", req.NamespacedName)
	}

	cfg, err := config.FromYAML([]byte(cm.Data[cmKey]))
	if err != nil {
		klog.Errorf("Got metricsDiscoveryConfig failed[%s] %v", pc.ConfigMap, err)
	}
	pc.EhpaController.UpdateMetricRules(*cfg)

	return ctrl.Result{}, nil
}

// SetupWithManager creates a controller and register to controller manager.
func (pc *PromAdapterConfigMapController) SetupWithManager(mgr ctrl.Manager) error {
	var metaConfigmap = strings.Split(pc.ConfigMap, "/")

	if len(metaConfigmap) < 1 {
		return fmt.Errorf("prometheus adapter configmap set error")
	}
	namespace := metaConfigmap[0]
	name := metaConfigmap[1]

	var promAdapterConfigMapChangedPredicate = &PromAdapterConfigMapChangedPredicate{
		Namespace: namespace,
		Name:      name,
	}

	// Watch for changes to ConfigMap
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(promAdapterConfigMapChangedPredicate)).
		Complete(pc)
}

func (paCm *PromAdapterConfigMapChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}
	if e.ObjectNew == nil {
		return false
	}

	if e.ObjectNew.GetName() == paCm.Name && e.ObjectNew.GetNamespace() == paCm.Namespace {
		return e.ObjectNew.GetResourceVersion() != e.ObjectOld.GetResourceVersion()
	}

	return false
}

func GetMatchLabels(extensionLabels map[string]string, metricLabels map[string]string) map[string]string {
	var matchLabels = make(map[string]string)

	for k := range extensionLabels {
		matchLabels[k] = extensionLabels[k]
	}

	for k := range metricLabels {
		matchLabels[k] = metricLabels[k]
	}

	return matchLabels
}
