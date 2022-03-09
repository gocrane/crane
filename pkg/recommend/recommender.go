package recommend

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommend/advisor"
	"github.com/gocrane/crane/pkg/recommend/inspector"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

func NewRecommender(kubeClient client.Client, restMapper meta.RESTMapper,
	scaleClient scale.ScalesGetter, recommendation *analysisapi.Recommendation,
	predictors map[predictionapi.AlgorithmType]prediction.Interface, dataSource providers.Interface,
	configSet *analysisapi.ConfigSet) (*Recommender, error) {
	c, err := GetContext(kubeClient, restMapper, scaleClient, recommendation, predictors, dataSource, configSet)
	if err != nil {
		return nil, err
	}

	return &Recommender{
		Context:    c,
		Inspectors: inspector.NewInspectors(c),
		Advisors:   advisor.NewAdvisors(c),
	}, nil
}

// Recommender take charge of the executor for recommendation
type Recommender struct {
	// Context contains all contexts during the recommendation
	Context *types.Context

	// Inspectors is an array of Inspector that needed for this recommendation
	Inspectors []inspector.Inspector

	// Advisors is an array of Advisor that needed for this recommendation
	Advisors []advisor.Advisor
}

func (r *Recommender) Offer() (proposed *types.ProposedRecommendation, err error) {
	proposed = &types.ProposedRecommendation{}

	// Run inspectors to validate target is ready to recommend
	var errs []error
	for _, inspector := range r.Inspectors {
		klog.V(4).Infof("Start inspector %s", inspector.Name())
		err := inspector.Inspect()
		klog.V(4).Infof("Complete inspector %s", inspector.Name())
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		err = utilerrors.NewAggregate(errs)
		return
	}

	// Run advisors to propose recommends
	for _, advisor := range r.Advisors {
		klog.V(4).Infof("Start advisor %s", advisor.Name())
		err := advisor.Advise(proposed)
		klog.V(4).Infof("Complete advisor %s", advisor.Name())
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		err = utilerrors.NewAggregate(errs)
	}

	return
}

func GetContext(kubeClient client.Client, restMapper meta.RESTMapper,
	scaleClient scale.ScalesGetter, recommendation *analysisapi.Recommendation,
	predictors map[predictionapi.AlgorithmType]prediction.Interface, dataSource providers.Interface,
	configSet *analysisapi.ConfigSet) (*types.Context, error) {
	c := &types.Context{}

	targetRef := autoscalingv2.CrossVersionObjectReference{
		APIVersion: recommendation.Spec.TargetRef.APIVersion,
		Kind:       recommendation.Spec.TargetRef.Kind,
		Name:       recommendation.Spec.TargetRef.Name,
	}

	target := analysisapi.Target{
		Kind:      recommendation.Spec.TargetRef.Kind,
		Namespace: recommendation.Spec.TargetRef.Namespace,
		Name:      recommendation.Spec.TargetRef.Name,
	}
	c.ConfigProperties = GetProperties(configSet, target)

	var scale *autoscalingv1.Scale
	var mapping *meta.RESTMapping
	var err error
	if recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		scale, mapping, err = utils.GetScale(context.TODO(), restMapper, scaleClient, recommendation.Spec.TargetRef.Namespace, targetRef)
		if err != nil {
			return nil, err
		}
		c.Scale = scale
		c.RestMapping = mapping
	}

	unstructured := &unstructured.Unstructured{}
	unstructured.SetKind(recommendation.Spec.TargetRef.Kind)
	unstructured.SetAPIVersion(recommendation.Spec.TargetRef.APIVersion)

	if err := kubeClient.Get(context.TODO(), client.ObjectKey{Namespace: recommendation.Spec.TargetRef.Namespace, Name: recommendation.Spec.TargetRef.Name}, unstructured); err != nil {
		return nil, err
	}

	if recommendation.Spec.TargetRef.Kind == "Deployment" && mapping.GroupVersionKind.Group == "apps" {
		var deployment appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &deployment); err != nil {
			return nil, err
		}
		c.Deployment = &deployment
	}

	if recommendation.Spec.TargetRef.Kind == "StatefulSet" && mapping.GroupVersionKind.Group == "apps" {
		var statefulSet appsv1.StatefulSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &statefulSet); err != nil {
			return nil, err
		}
		c.StatefulSet = &statefulSet
	}

	var pods []corev1.Pod
	if recommendation.Spec.TargetRef.Kind != "DaemonSet" {
		pods, err = utils.GetPodsFromScale(kubeClient, scale)
	} else {
		pods, err = getDaemonSetPods(kubeClient, recommendation.Spec.TargetRef.Namespace, recommendation.Spec.TargetRef.Name)
	}
	if err != nil {
		return nil, err
	}

	if recommendation.Spec.Type == analysisapi.AnalysisTypeHPA {
		c.PodTemplate, err = utils.GetPodTemplate(context.TODO(),
			recommendation.Spec.TargetRef.Namespace,
			recommendation.Spec.TargetRef.Name,
			recommendation.Spec.TargetRef.Kind,
			recommendation.Spec.TargetRef.APIVersion,
			kubeClient)
		if err != nil {
			return nil, err
		}

		hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
		opts := []client.ListOption{
			client.InNamespace(recommendation.Spec.TargetRef.Namespace),
		}
		err := kubeClient.List(context.TODO(), hpaList, opts...)
		if err != nil {
			return nil, err
		}

		for _, hpa := range hpaList.Items {
			// bypass hpa that controller by ehpa
			if hpa.Labels != nil && hpa.Labels["app.kubernetes.io/managed-by"] == known.EffectiveHorizontalPodAutoscalerManagedBy {
				continue
			}

			if hpa.Spec.ScaleTargetRef.Name == recommendation.Spec.TargetRef.Name &&
				hpa.Spec.ScaleTargetRef.Kind == recommendation.Spec.TargetRef.APIVersion &&
				hpa.Spec.ScaleTargetRef.APIVersion == recommendation.Spec.TargetRef.APIVersion {
				c.HPA = &hpa
			}
		}
	}

	c.Pods = pods
	c.Predictors = predictors
	c.DataSource = dataSource
	c.Recommendation = recommendation

	return c, nil
}

func getDaemonSetPods(kubeClient client.Client, namespace string, name string) ([]corev1.Pod, error) {
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
