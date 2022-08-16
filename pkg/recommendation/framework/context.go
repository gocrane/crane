package framework

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction/config"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/utils"
)

type RecommendationContext struct {
	Context context.Context
	// The kubernetes resource object reference of recommendation flow.
	Identity ObjectIdentity
	// Target Object
	Object client.Object
	// Time series data from data source.
	InputValues []*common.TimeSeries
	// Result series from prediction
	ResultValues []*common.TimeSeries
	// DataProviders contains data source of your recommendation flow.
	DataProviders map[providers.DataSourceType]providers.History
	// Recommendation store result of recommendation flow.
	Recommendation *v1alpha1.Recommendation
	// When cancel channel accept signal indicates that the context has been canceled. The recommendation should stop executing as soon as possible.
	// CancelCh <-chan struct{}
	// RecommendationRule for the context
	RecommendationRule v1alpha1.RecommendationRule
	// metrics namer for datasource provider
	MetricNamer metricnaming.MetricNamer
	// Algorithm Config
	AlgorithmConfig *config.Config
	// Manager of predict algorithm
	PredictorMgr predictormgr.Manager
	// Pod template
	PodTemplate *corev1.PodTemplateSpec
	// Client
	Client client.Client
	// RestMapper
	RestMapper meta.RESTMapper
	// ScalesGetter
	ScaleClient scale.ScalesGetter
	// Scale
	Scale *autoscalingapiv1.Scale
	// Pods in recommendation
	Pods []corev1.Pod
	// HPA Object
	HPA *autoscalingv2.HorizontalPodAutoscaler
}

func NewRecommendationContext(context context.Context, identity ObjectIdentity, dataProviders map[providers.DataSourceType]providers.History, recommendation *v1alpha1.Recommendation, client client.Client, scaleClient scale.ScalesGetter) RecommendationContext {
	return RecommendationContext{
		Identity:       identity,
		Object:         &identity.Object,
		Context:        context,
		DataProviders:  dataProviders,
		Recommendation: recommendation,
		Client:         client,
		RestMapper:     client.RESTMapper(),
		ScaleClient:    scaleClient,
		//CancelCh:       context.Done(),
	}
}

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
	Labels     map[string]string
	Object     unstructured.Unstructured
}

func (id ObjectIdentity) GetObjectReference() corev1.ObjectReference {
	return corev1.ObjectReference{Kind: id.Kind, APIVersion: id.APIVersion, Namespace: id.Namespace, Name: id.Name}
}

//func (ctx *RecommendationContext) Canceled() bool {
//	select {
//	case <-ctx.CancelCh:
//		return true
//	default:
//		return false
//	}
//}

func ObjectConversion(object interface{}, target interface{}) error {
	bytes, err := json.Marshal(object)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, target)
}

func GetDaemonSetPods(kubeClient client.Client, namespace string, name string) ([]corev1.Pod, error) {
	ds := appsv1.DaemonSet{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &ds)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(ds.Spec.Selector.MatchLabels),
	}

	podList := &corev1.PodList{}
	err = kubeClient.List(context.TODO(), podList, opts...)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func RetrievePodTemplate(ctx *RecommendationContext) error {
	unstructed := ctx.Object.(*unstructured.Unstructured)

	// fill PodTemplate
	podTemplateObject, found, err := unstructured.NestedMap(unstructed.Object, "spec", "template")
	if !found || err != nil {
		return fmt.Errorf("get template from unstructed object %s failed. ", klog.KObj(unstructed))
	}

	return ObjectConversion(podTemplateObject, ctx.PodTemplate)
}

func RetrieveScale(ctx *RecommendationContext) error {
	targetRef := autoscalingv2.CrossVersionObjectReference{
		APIVersion: ctx.Recommendation.Spec.TargetRef.APIVersion,
		Kind:       ctx.Recommendation.Spec.TargetRef.Kind,
		Name:       ctx.Recommendation.Spec.TargetRef.Name,
	}

	if ctx.Recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		scale, _, err := utils.GetScale(context.TODO(), ctx.RestMapper, ctx.ScaleClient, ctx.Recommendation.Spec.TargetRef.Namespace, targetRef)
		if err != nil {
			return err
		}
		ctx.Scale = scale
	}
	return nil
}

func RetrievePods(ctx *RecommendationContext) error {
	if ctx.Recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		pods, err := utils.GetPodsFromScale(ctx.Client, ctx.Scale)
		ctx.Pods = pods
		return err
	} else {
		var daemonSet appsv1.DaemonSet
		err := ObjectConversion(ctx.Object, &daemonSet)
		if err != nil {
			return err
		}
		pods, err := GetDaemonSetPods(ctx.Client, ctx.Recommendation.Spec.TargetRef.Namespace, ctx.Recommendation.Spec.TargetRef.Name)
		ctx.Pods = pods
		return err
	}
}
