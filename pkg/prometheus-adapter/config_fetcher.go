package prometheus_adapter

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
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

type PrometheusAdapterConfigFetcher struct {
	client.Client
	Scheme               *runtime.Scheme
	RestMapper           meta.RESTMapper
	Recorder             record.EventRecorder
	AdapterConfigMapNS   string
	AdapterConfigMapName string
	AdapterConfigMapKey  string
	AdapterConfig        string
}

type PrometheusAdapterConfigChangedPredicate struct {
	predicate.Funcs
	Name      string
	Namespace string
}

func (pc *PrometheusAdapterConfigFetcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if req.NamespacedName.String() != pc.AdapterConfigMapNS+"/"+pc.AdapterConfigMapName {
		return ctrl.Result{}, fmt.Errorf("configmap %s/%s not matched", req.NamespacedName, req.NamespacedName.Name)
	}
	klog.V(4).Infof("Got prometheus adapter configmap %s", req.NamespacedName)

	//get configmap content
	cm := &corev1.ConfigMap{}
	err := pc.Client.Get(ctx, req.NamespacedName, cm)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cm == nil {
		return ctrl.Result{}, fmt.Errorf("get configmap %s/%s failed", req.NamespacedName.Namespace, req.NamespacedName.Name)
	}

	cfg, err := config.FromYAML([]byte(cm.Data[pc.AdapterConfigMapKey]))
	if err != nil {
		klog.Errorf("Got metricsDiscoveryConfig failed[%s] %v", pc.AdapterConfigMapName, err)
	}

	err = FlushRules(*cfg)
	if err != nil {
		klog.Errorf("Flush rules failed %v", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager creates a controller and register to controller manager.
func (pc *PrometheusAdapterConfigFetcher) SetupWithManager(mgr ctrl.Manager) error {
	var promAdapterConfigMapChangedPredicate = &PrometheusAdapterConfigChangedPredicate{
		Namespace: pc.AdapterConfigMapNS,
		Name:      pc.AdapterConfigMapName,
	}

	// Watch for changes to ConfigMap
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(promAdapterConfigMapChangedPredicate)).
		Complete(pc)
}

// fetched metricRule if configmap is updated
func (paCm *PrometheusAdapterConfigChangedPredicate) Update(e event.UpdateEvent) bool {
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

// if set promAdapterConfig, daemon reload by config fsnotify
func (pc *PrometheusAdapterConfigFetcher) Reload() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Error(err)
		return
	}
	defer watcher.Close()
	err = watcher.Add(pc.AdapterConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to watch", "file", pc.AdapterConfig)
		return
	}
	klog.Infof("Start watching %s for update.", pc.AdapterConfig)

	for {
		select {
		case event, ok := <-watcher.Events:
			klog.Infof("Watched an event: %v", event)
			if !ok {
				return
			}
			metricsDiscoveryConfig, err := config.FromFile(pc.AdapterConfig)
			if err != nil {
				klog.Errorf("Got metricsDiscoveryConfig failed[%s] %v", pc.AdapterConfig, err)
			} else {
				err = FlushRules(*metricsDiscoveryConfig)
				if err != nil {
					klog.Errorf("Flush rules failed %v", err)
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

func FlushRules(metricsDiscoveryConfig config.MetricsDiscoveryConfig) error {
	err := ParsingResourceRules(metricsDiscoveryConfig)
	if err != nil {
		return err
	}
	err = ParsingRules(metricsDiscoveryConfig)
	if err != nil {
		return err
	}
	err = ParsingExternalRules(metricsDiscoveryConfig)
	if err != nil {
		return err
	}

	return nil
}
