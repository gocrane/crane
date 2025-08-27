package timeseriesprediction

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	DefaultTSPTemplateConfigMapName      = "tsp-templates-configmap"
	DefaultTSPTemplateConfigMapNamespace = "crane-system"
)

// TSPTemplateConfigMapReconciler listens to ConfigMap and updates template store
type TSPTemplateConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	TemplateStore *TSPTemplateStore
	ConfigMapName string
	Namespace     string
}

// SetupWithManager sets up the Controller
func (r *TSPTemplateConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("tsp-template-configmap-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch the specified ConfigMap
	err = c.Watch(
		&source.Kind{Type: &corev1.ConfigMap{}},
		&handler.EnqueueRequestForObject{},
		predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return e.Object.GetNamespace() == r.Namespace && e.Object.GetName() == r.ConfigMapName
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectNew.GetNamespace() == r.Namespace && e.ObjectNew.GetName() == r.ConfigMapName
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return e.Object.GetNamespace() == r.Namespace && e.Object.GetName() == r.ConfigMapName
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return e.Object.GetNamespace() == r.Namespace && e.Object.GetName() == r.ConfigMapName
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// Reconcile handles ConfigMap changes
func (r *TSPTemplateConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).InfoS("Reconciling tsp template Configmap", "NamespacedName", req.NamespacedName)
	configMap := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, req.NamespacedName, configMap)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Warningf("ConfigMap %s/%s not found, using default templates", req.Namespace, req.Name)
			// Clear the store because ConfigMap does not exist
			r.TemplateStore.Lock()
			r.TemplateStore.templates = make(map[string]TSPTemplate)
			r.TemplateStore.Unlock()
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Update template store
	r.TemplateStore.UpdateFromConfigMap(configMap)
	return ctrl.Result{}, nil
}
