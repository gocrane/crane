package ehpa

import (
	"context"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/gocrane/crane/pkg/metrics"
)

// HPAObserverController is responsible for observer metrics for hpa
type HPAObserverController struct {
	client.Client
	Scheme     *runtime.Scheme
	RestMapper meta.RESTMapper
	Recorder   record.EventRecorder
}

func (c *HPAObserverController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(8).Infof("Got hpa %s", req.NamespacedName)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := c.Client.Get(ctx, req.NamespacedName, hpa); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labels := map[string]string{
		"namespace": hpa.Namespace,
		"name":      hpa.Name,
	}
	metrics.HPAReplicas.With(labels).Set(float64(hpa.Status.DesiredReplicas))

	return ctrl.Result{}, nil
}

func (c *HPAObserverController) SetupWithManager(mgr ctrl.Manager) error {
	// Create a new controller
	controller, err := controller.New("hpa-observer-controller", mgr, controller.Options{
		Reconciler: c})
	if err != nil {
		return err
	}

	// Watch for changes to HPA
	return controller.Watch(&source.Kind{Type: &autoscalingv2.HorizontalPodAutoscaler{}}, &hpaEventHandler{
		enqueueHandler: handler.EnqueueRequestForObject{},
	})
}
