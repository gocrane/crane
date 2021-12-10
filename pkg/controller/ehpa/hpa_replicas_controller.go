package ehpa

import (
	"context"

	"github.com/go-logr/logr"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/crane/pkg/metrics"
)

// HPAReplicasController is responsible for monitor and export replicas for hpa
type HPAReplicasController struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestMapper meta.RESTMapper
	Recorder   record.EventRecorder
}

func (c *HPAReplicasController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "hpa", req.NamespacedName)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := c.Client.Get(ctx, req.NamespacedName, hpa); err != nil {
		return ctrl.Result{}, err
	}

	labels := map[string]string{
		"identity": klog.KObj(hpa).String(),
	}
	metrics.HPAReplicas.With(labels).Set(float64(hpa.Status.DesiredReplicas))

	return ctrl.Result{}, nil
}

func (c *HPAReplicasController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(c)
}
