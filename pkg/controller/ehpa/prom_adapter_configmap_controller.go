package ehpa

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/prometheus-adapter/pkg/config"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/utils"
)

// controller for configMap of prometheus-adapter
type PromAdapterConfigMapController struct {
	client.Client
	Scheme         *runtime.Scheme
	RestMapper     meta.RESTMapper
	Recorder       record.EventRecorder
	ConfigMap      string
	EhpaController *EffectiveHPAController
}

type PromAdapterConfigMapChangedPredicate struct {
	predicate.Funcs
	Name      string
	Namespace string
}

func (pc *PromAdapterConfigMapController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	metricRulesResource, metricRulesCustomer, metricRulesExternal, err := utils.GetMetricRules(*cfg, pc.RestMapper)
	if err != nil {
		klog.Errorf("Got metricRules failed[%s] %v", pc.ConfigMap, err)
	} else {
		pc.EhpaController.MetricRulesResource = metricRulesResource
		pc.EhpaController.MetricRulesCustomer = metricRulesCustomer
		pc.EhpaController.MetricRulesExternal = metricRulesExternal
	}

	//get ehpaList and match metric of configmap
	ehpaList := &autoscalingapi.EffectiveHorizontalPodAutoscalerList{}
	opts := []client.ListOption{}
	err = pc.Client.List(ctx, ehpaList, opts...)
	if err != nil || len(ehpaList.Items) == 0 {
		return ctrl.Result{}, err
	}
	for _, ehpa := range ehpaList.Items {
		if !utils.IsEHPAPredictionEnabled(&ehpa) {
			continue
		}

		extensionLabels, err := utils.GetExtensionLabelsAnnotationPromAdapter(ehpa.Annotations)
		if err != nil {
			klog.Errorf("Got extensionLabels by prometheus-adapter annotation ehpa[%s] %v", ehpa.Name, err)
		}

		for _, metric := range ehpa.Spec.Metrics {
			var metricName string
			var metricIdentifier string
			var expressionQuery string
			switch metric.Type {
			case autoscalingv2.ResourceMetricSourceType:
				metricName = metric.Resource.Name.String()
				metricIdentifier = utils.GetMetricIdentifier(metric, metric.Resource.Name.String())
				if len(pc.EhpaController.MetricRulesResource) > 0 {
					for _, metricRule := range pc.EhpaController.MetricRulesResource {
						if match, _ := (regexp.Match(metricRule.MetricMatches, []byte(metricName))); match {
							klog.V(4).Infof("Got MetricRulesResource prometheus-adapter-resource MetricMatches[%s] SeriesName[%s]", metricRule.MetricMatches, metricRule.SeriesName)
							expressionQuery, err = metricRule.QueryForSeriesResource(ehpa.Namespace, labels.SelectorFromSet(extensionLabels), ehpa.Spec.ScaleTargetRef.Name, utils.GetPodNameReg(ehpa.Spec.ScaleTargetRef.Name, ehpa.Spec.ScaleTargetRef.Kind))
							if err != nil {
								klog.Errorf("Got promSelector prometheus-adapter-resource %v", err)
							} else {
								klog.V(4).Infof("Got expressionQuery prometheus-adapter-resource [%s]", expressionQuery)
							}
						}
					}
				}
			case autoscalingv2.PodsMetricSourceType:
				metricName = metric.Pods.Metric.Name
				metricIdentifier = utils.GetMetricIdentifier(metric, metric.Pods.Metric.Name)
				if len(pc.EhpaController.MetricRulesCustomer) > 0 {
					for _, metricRule := range pc.EhpaController.MetricRulesCustomer {
						if match, _ := (regexp.Match(metricRule.MetricMatches, []byte(metricName))); match {
							klog.V(4).Infof("Got metricRule prometheus-adapter-customer ehpa[%s] MetricMatches[%s] SeriesName[%s]", ehpa.Name, metricRule.MetricMatches, metricRule.SeriesName)
							var matchLabels map[string]string
							if metric.Pods.Metric.Selector != nil {
								matchLabels = GetMatchLabels(extensionLabels, metric.Pods.Metric.Selector.MatchLabels)
							} else {
								matchLabels = extensionLabels
							}

							expressionQuery, err = metricRule.QueryForSeriesCustomer(metricRule.SeriesName, ehpa.Namespace, labels.SelectorFromSet(matchLabels))
							if err != nil {
								klog.Errorf("Got promSelector prometheus-adapter-customer ehpa[%s] %v", ehpa.Name, err)
							} else {
								klog.V(4).Infof("Got expressionQuery prometheus-adapter-customer ehpa[%s] [%s]", ehpa.Name, expressionQuery)
							}
						}
					}
				}
			case autoscalingv2.ExternalMetricSourceType:
				metricName = metric.External.Metric.Name
				metricIdentifier = utils.GetMetricIdentifier(metric, metric.External.Metric.Name)
				if len(pc.EhpaController.MetricRulesExternal) > 0 {
					for _, metricRule := range pc.EhpaController.MetricRulesExternal {
						if match, _ := (regexp.Match(metricRule.MetricMatches, []byte(metricName))); match {
							klog.V(4).Infof("Got metricRule prometheus-adapter-external ehpa[%s] MetricMatches[%s] SeriesName[%s]", ehpa.Name, metricRule.MetricMatches, metricRule.SeriesName)
							var matchLabels map[string]string
							if metric.External.Metric.Selector != nil {
								matchLabels = GetMatchLabels(extensionLabels, metric.External.Metric.Selector.MatchLabels)
							} else {
								matchLabels = extensionLabels
							}

							expressionQuery, err = metricRule.QueryForSeriesExternal(metricRule.SeriesName, ehpa.Namespace, labels.SelectorFromSet(matchLabels))
							if err != nil {
								klog.Errorf("Got promSelector prometheus-adapter-external ehpa[%s] %v", ehpa.Name, err)
							} else {
								klog.V(4).Infof("Got expressionQuery prometheus-adapter-external ehpa[%s] [%s]", ehpa.Name, expressionQuery)
							}
						}
					}
				}
			}
			if utils.IsExpressionQueryAnnotationEnabled(metricIdentifier, ehpa.Annotations) {
				klog.V(4).Infof("ExpressionQueryAnnotationEnabled [%s/%s]", ehpa.Name, metricIdentifier)
				continue
			}
			if expressionQuery == "" {
				klog.Errorf("unable to get expressionQuery for ehpa [%s] series:%s, skipping", ehpa.Name, metricName)
				continue
			}

			// reconcile prediction if enabled
			tsp := &predictionapi.TimeSeriesPrediction{}
			if err := pc.Client.Get(ctx, types.NamespacedName{Namespace: ehpa.Namespace, Name: "ehpa-" + ehpa.Name}, tsp); err != nil {
				klog.Errorf("unable to get TimeSeriesPrediction [%s] %v", ehpa.Namespace+"/ehpa-"+ehpa.Name, err)
				break
			}
			var deleteTsp bool
			for _, i := range tsp.Spec.PredictionMetrics {
				if metricIdentifier == i.ResourceIdentifier {
					if expressionQuery != i.ExpressionQuery.Expression {
						deleteTsp = true
						err := pc.Client.Delete(context.TODO(), tsp)
						if err != nil {
							klog.Errorf("unable to delete TimeSeriesPrediction [%s] %v", ehpa.Namespace+"/ehpa-"+ehpa.Name, err)
						}
						break
					}
				}
			}

			if deleteTsp {
				break
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager creates a controller and register to controller manager.
func (pc *PromAdapterConfigMapController) SetupWithManager(mgr ctrl.Manager) error {
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

func GetMatchLabels(extensionLabels map[string]string, metricLabels map[string]string) map[string]string {
	var matchLabels = make(map[string]string)

	for k := range extensionLabels {
		matchLabels[k] = extensionLabels[k]
	}

	for k := range metricLabels {
		matchLabels[k] = metricLabels[k]
	}

	return matchLabels
}
