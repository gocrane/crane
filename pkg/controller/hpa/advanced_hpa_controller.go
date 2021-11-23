package hpa

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

// AdvancedHPAController is responsible for scaling workload's replica based on AdvancedHorizontalPodAutoscaler spec
type AdvancedHPAController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	scaleClient scale.ScalesGetter
	K8SVersion  *version.Version
}

func (p *AdvancedHPAController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	p.Log.Info("got", "advanced-hpa", req.NamespacedName)

	ahpa := &autoscalingapi.AdvancedHorizontalPodAutoscaler{}
	err := p.Client.Get(ctx, req.NamespacedName, ahpa)
	if err != nil {
		return ctrl.Result{}, err
	}
	// todo: check scaleTarget

	// reconcile prediction if enabled
	if IsPredictionEnabled(ahpa) {
		err = p.ReconcilePodPredication(ctx, ahpa)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err = p.ReconcileHPA(ctx, ahpa)
	if err != nil {
		return ctrl.Result{}, err
	}

	// todo: update status
	// autoscalingStatusOrigin := autoscaler.Status.DeepCopy()

	return ctrl.Result{}, nil
}

func (p *AdvancedHPAController) GetScale(ctx context.Context, ahpa *autoscalingapi.AdvancedHorizontalPodAutoscaler) (*autoscalingapiv1.Scale, *meta.RESTMapping, error) {
	targetGV, err := schema.ParseGroupVersion(ahpa.Spec.ScaleTargetRef.APIVersion)
	if err != nil {
		return nil, nil, err
	}

	targetGK := schema.GroupKind{
		Group: targetGV.Group,
		Kind:  ahpa.Spec.ScaleTargetRef.Kind,
	}

	mappings, err := p.RestMapper.RESTMappings(targetGK)
	if err != nil {
		return nil, nil, err
	}

	for _, mapping := range mappings {
		scale, err := p.scaleClient.Scales(ahpa.Namespace).Get(ctx, mapping.Resource.GroupResource(), ahpa.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
		if err == nil {
			return scale, mapping, nil
		}
	}

	return nil, nil, fmt.Errorf("unrecognized resource")
}

func (p *AdvancedHPAController) GetPodsFromScale(scale *autoscalingapiv1.Scale) ([]v1.Pod, error) {
	selector, err := labels.ConvertSelectorToLabelsMap(scale.Status.Selector)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{
		client.InNamespace(scale.GetNamespace()),
		client.MatchingLabels(selector),
	}

	podList := &v1.PodList{}
	err = p.Client.List(context.TODO(), podList, opts...)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func (p *AdvancedHPAController) SetupWithManager(mgr ctrl.Manager) error {
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
	p.scaleClient = scaleClient
	serverVersion, err := discoveryClientSet.ServerVersion()
	if err != nil {
		return err
	}
	K8SVersion, err := version.ParseGeneric(serverVersion.GitVersion)
	if err != nil {
		return err
	}
	p.K8SVersion = K8SVersion
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingapi.AdvancedHorizontalPodAutoscaler{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(p)
}
