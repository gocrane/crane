package prometheus_adapter

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
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
type PromAdapterConfigMapFetcher struct {
	client.Client
	Scheme     *runtime.Scheme
	RestMapper meta.RESTMapper
	Recorder   record.EventRecorder
	ConfigMap  string
	Config     string
}

type PromAdapterConfigMapChangedPredicate struct {
	predicate.Funcs
	Name      string
	Namespace string
}

func (pc *PromAdapterConfigMapFetcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	//FlushRules
	err = FlushResourceRules(*cfg, pc.RestMapper)
	if err != nil {
		klog.Errorf("FlushResourceRules failed %v", err)
	}
	err = FlushRules(*cfg, pc.RestMapper)
	if err != nil {
		klog.Errorf("FlushRules failed %v", err)
	}
	err = FlushExternalRules(*cfg, pc.RestMapper)
	if err != nil {
		klog.Errorf("FlushExternalRules failed %v", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager creates a controller and register to controller manager.
func (pc *PromAdapterConfigMapFetcher) SetupWithManager(mgr ctrl.Manager) error {
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

// if set promAdapterConfig, daemon reload by config's md5
func (pc *PromAdapterConfigMapFetcher) PromAdapterConfigDaemonReload() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Error(err)
		return
	}
	defer watcher.Close()
	err = watcher.Add(pc.Config)
	if err != nil {
		klog.ErrorS(err, "Failed to watch", "file", pc.Config)
		return
	}
	klog.Infof("Start watching %s for update.", pc.Config)

	for {
		select {
		case event, ok := <-watcher.Events:
			klog.Infof("Watched an event: %v", event)
			if !ok {
				return
			}
			metricsDiscoveryConfig, err := config.FromFile(pc.Config)
			if err != nil {
				klog.Errorf("Got metricsDiscoveryConfig failed[%s] %v", pc.Config, err)
			} else {
				err = FlushResourceRules(*metricsDiscoveryConfig, pc.RestMapper)
				if err != nil {
					klog.Errorf("FlushResourceRules failed %v", err)
				}
				err = FlushRules(*metricsDiscoveryConfig, pc.RestMapper)
				if err != nil {
					klog.Errorf("FlushRules failed %v", err)
				}
				err = FlushExternalRules(*metricsDiscoveryConfig, pc.RestMapper)
				if err != nil {
					klog.Errorf("FlushExternalRules failed %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			klog.Error(err)
		}
	}
}
