package analysis

import (
	"context"
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AnalyticsController struct {
	client.Client
	Logger         logr.Logger
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	scaleClient scale.ScalesGetter
	K8SVersion  *version.Version
}

func (ctl *AnalyticsController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctl.Logger.Info("Got an analytics res", "analytics", req.NamespacedName)

	a := &analysisv1alph1.Analytics{}

	err := ctl.Client.Get(ctx, req.NamespacedName, a)
	if err != nil {
		return ctrl.Result{}, err
	}

	// todo: update status
	// autoscalingStatusOrigin := autoscaler.Status.DeepCopy()

	return ctrl.Result{}, nil
}
