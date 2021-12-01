package analysis

import (
	"context"
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

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

func (ac *AnalyticsController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ac.Logger.Info("Got an analytics res", "analytics", req.NamespacedName)

	a := &analysisv1alph1.Analytics{}

	err := ac.Client.Get(ctx, req.NamespacedName, a)
	if err != nil {
		return ctrl.Result{}, err
	}



	return ctrl.Result{}, nil
}

func (ac *AnalyticsController) SetupWithManager(mgr ctrl.Manager) error {
	discoveryClientSet, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discoveryClientSet)
	scaleClient := scale.New(
		discoveryClientSet.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scaleKindResolver,
	)
	ac.scaleClient = scaleClient
	serverVersion, err := discoveryClientSet.ServerVersion()
	if err != nil {
		return err
	}
	K8SVersion, err := version.ParseGeneric(serverVersion.GitVersion)
	if err != nil {
		return err
	}
	ac.K8SVersion = K8SVersion
	return ctrl.NewControllerManagedBy(mgr).
		For(&analysisv1alph1.Analytics{}).
		Owns(&analysisv1alph1.Recommendation{}).
		Complete(ac)
}
