package timeseriesprediction

import (
	"context"
	"fmt"
	"github.com/gocrane/crane/pkg/utils/target"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	predictionv1alpha1 "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	tspNamespace                   = "crane-system"
	tspNameBase                    = "tsp"
	defaultPredictionWindowSeconds = 86400 // Default prediction window is 24 hours
)

// TimeSeriesPredictionRuleController reconciles TimeSeriesPredictionRule objects
type TimeSeriesPredictionRuleController struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	DynamicClient dynamic.Interface
	Discovery     discovery.DiscoveryInterface
	TargetFetcher target.SelectorFetcher
	TemplateStore *TSPTemplateStore
}

// Reconcile is the main reconciliation loop for TimeSeriesPredictionRule
func (c *TimeSeriesPredictionRuleController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).InfoS("Reconciling TimeSeriesPredictionRule", "NamespacedName", req.NamespacedName)
	startTime := time.Now()
	defer func() {
		klog.V(1).Infof("Finished syncing tspr[%s] (%v)", req.NamespacedName, time.Since(startTime))
	}()
	tspRule := &predictionv1alpha1.TimeSeriesPredictionRule{}
	err := c.Client.Get(ctx, req.NamespacedName, tspRule)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if tspRule.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// Handle one-time tasks
	lastUpdateTime := tspRule.Status.LastUpdateTime
	if len(strings.TrimSpace(tspRule.Spec.RunInterval)) == 0 {
		if lastUpdateTime != nil {
			// One-time task completed
			return ctrl.Result{}, nil
		}
	}

	// Check if it's time to execute
	interval, err := time.ParseDuration(tspRule.Spec.RunInterval)
	if err != nil {
		c.Recorder.Event(tspRule, corev1.EventTypeWarning, "FailedParseRunInterval", err.Error())
		klog.ErrorS(err, "Failed to parse RunInterval", "TimeSeriesPredictionRule", klog.KObj(tspRule))
		return ctrl.Result{}, err
	}

	if lastUpdateTime != nil {
		planingTime := lastUpdateTime.Add(interval)
		now := time.Now()
		if now.Before(planingTime) {
			klog.V(1).InfoS("Will re-sync", "after", planingTime.Sub(now), "tspr", klog.KObj(tspRule))
			return ctrl.Result{
				RequeueAfter: planingTime.Sub(now),
			}, nil
		}
	}

	// Call core sync logic
	finished, err := c.syncTimeSeriesPredictionRule(ctx, tspRule)
	if err != nil {
		c.Recorder.Event(tspRule, corev1.EventTypeWarning, "SyncFailed", err.Error())
		klog.ErrorS(err, "Failed to sync TimeSeriesPredictionRule", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// Update status
	if finished {
		updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			latestTspRule := &predictionv1alpha1.TimeSeriesPredictionRule{}
			if err := c.Client.Get(ctx, req.NamespacedName, latestTspRule); err != nil {
				klog.V(1).Infof("Failed to get latest TimeSeriesPredictionRule: %v", err)
				return err
			}

			// Update LastUpdateTime
			now := metav1.Now()
			latestTspRule.Status.LastUpdateTime = &now
			return c.Client.Status().Update(ctx, latestTspRule)
		})

		if updateErr != nil {
			c.Recorder.Event(tspRule, corev1.EventTypeWarning, "StatusUpdateFailed", updateErr.Error())
			klog.ErrorS(updateErr, "Failed to update TimeSeriesPredictionRule status", "NamespacedName", req.NamespacedName)
			return ctrl.Result{}, updateErr
		}
	}

	// Set requeue time based on sync result
	requeueAfter := 30 * time.Second
	if finished && len(strings.TrimSpace(tspRule.Spec.RunInterval)) != 0 {
		klog.V(4).InfoS("Will re-sync", "after", interval, "tspr", klog.KObj(tspRule))
		requeueAfter = interval
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// syncTimeSeriesPredictionRule implements the core sync logic
func (c *TimeSeriesPredictionRuleController) syncTimeSeriesPredictionRule(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule) (bool, error) {
	klog.V(1).Infof("Syncing TimeSeriesPredictionRule: %s/%s", tspRule.Namespace, tspRule.Name)

	// Get all matching workloads
	workloads, err := c.getMatchingWorkloads(ctx, tspRule)
	if err != nil {
		return false, fmt.Errorf("failed to get matching workloads: %v", err)
	}
	klog.V(1).Infof("Found %d matching workloads for rule %s/%s", len(workloads), tspRule.Namespace, tspRule.Name)

	// Get currently associated TSP objects
	var currentTSPs predictionv1alpha1.TimeSeriesPredictionList
	labelSelector := labels.SelectorFromSet(labels.Set{
		"timeseriespredictionrule.name": tspRule.Name,
	})
	err = c.List(ctx, &currentTSPs, &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, fmt.Errorf("failed to list existing TimeSeriesPredictions: %v", err)
	}

	// Create TSP object index for quick lookup
	tspIndex := make(map[string]*predictionv1alpha1.TimeSeriesPrediction)
	for i := range currentTSPs.Items {
		tsp := &currentTSPs.Items[i]
		key, err := getWorkloadMatchNameFromTSP(tsp)
		if err != nil {
			klog.Errorf("Failed to get workload match name from tsp: %v", err)
			continue
		}
		tspIndex[key] = tsp
	}
	klog.V(1).Infof("Found %d existing TimeSeriesPredictions for rule %s/%s", len(tspIndex), tspRule.Namespace, tspRule.Name)
	// Create or update TSP objects
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(workloads))

	for _, workload := range workloads {
		wg.Add(1)
		go func(wl workloadInfo) {
			defer wg.Done()
			key := getWorkloadMatchNameFromWorkloadInfo(&wl)

			mu.Lock()
			// Check if TSP object already exists
			existingTSP, exists := tspIndex[key]
			if exists {
				delete(tspIndex, key) // Remove from index, remaining will be orphan objects
			}
			mu.Unlock()

			if exists {
				// Update existing TSP object
				if err := c.updateTimeSeriesPrediction(ctx, tspRule, existingTSP, wl); err != nil {
					errCh <- fmt.Errorf("failed to update TimeSeriesPrediction for %s: %v", key, err)
				}
				return
			}

			klog.V(1).Infof("Creating new TimeSeriesPrediction for workload: %s", key)
			// Create new TSP object
			if err := c.createTimeSeriesPrediction(ctx, tspRule, wl); err != nil {
				errCh <- fmt.Errorf("failed to create TimeSeriesPrediction for %s: %v", key, err)
			}
		}(workload)
	}

	wg.Wait()
	close(errCh)

	// Handle errors
	var syncErrors []error
	for err := range errCh {
		syncErrors = append(syncErrors, err)
	}
	if len(syncErrors) > 0 {
		return false, fmt.Errorf("encountered %d errors during sync", len(syncErrors))
	}

	// Clean up orphan TSP objects (TSP object's target workload has disappeared)
	klog.V(1).Infof("Cleaning up orphan TimeSeriesPredictions for rule %s/%s, number: %v", tspRule.Namespace, tspRule.Name, len(tspIndex))
	for _, tsp := range tspIndex {
		if err := c.Client.Delete(ctx, tsp); err != nil {
			klog.ErrorS(err, "Failed to delete orphan TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
		} else {
			klog.InfoS("Deleted orphan TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
		}
	}

	return true, nil
}

// getMatchingWorkloads gets the list of matching workloads
func (c *TimeSeriesPredictionRuleController) getMatchingWorkloads(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule) ([]workloadInfo, error) {
	var workloads []workloadInfo

	for _, rs := range tspRule.Spec.ResourceSelectors {
		if rs.Kind == "" {
			return nil, fmt.Errorf("empty kind in resource selector")
		}

		gvr, err := utils.GetGroupVersionResource(c.Discovery, rs.APIVersion, rs.Kind)
		if err != nil {
			klog.V(1).Infof("Warning: failed to get GVR for %s/%s: %v, skip", rs.APIVersion, rs.Kind, err)
			continue
		}

		var unstructureds []unstructured.Unstructured
		if tspRule.Spec.NamespaceSelector.Any {
			unstructuredList, err := c.DynamicClient.Resource(*gvr).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to list resources in all namespaces: %v", err)
			}
			unstructureds = append(unstructureds, unstructuredList.Items...)
		} else {
			for _, namespace := range tspRule.Spec.NamespaceSelector.MatchNames {
				unstructuredList, err := c.DynamicClient.Resource(*gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, fmt.Errorf("failed to list resources in namespace %s: %v", namespace, err)
				}
				unstructureds = append(unstructureds, unstructuredList.Items...)
			}
		}

		for _, u := range unstructureds {
			// Filter by name
			if rs.Name != "" && u.GetName() != rs.Name {
				continue
			}

			// Filter by label selector
			if rs.LabelSelector != nil {
				selector, err := metav1.LabelSelectorAsSelector(rs.LabelSelector)
				if err != nil {
					return nil, fmt.Errorf("invalid label selector: %v", err)
				}
				if !selector.Matches(labels.Set(u.GetLabels())) {
					continue
				}
			}

			workloads = append(workloads, workloadInfo{
				Namespace:  u.GetNamespace(),
				Name:       u.GetName(),
				APIVersion: rs.APIVersion,
				Kind:       rs.Kind,
			})
		}
	}

	return workloads, nil
}

// createTimeSeriesPrediction creates a new TimeSeriesPrediction object
func (c *TimeSeriesPredictionRuleController) createTimeSeriesPrediction(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule, workload workloadInfo) error {
	// Validate if target has selector
	objRef := &corev1.ObjectReference{
		APIVersion: workload.APIVersion,
		Kind:       workload.Kind,
		Namespace:  workload.Namespace,
		Name:       workload.Name,
	}
	_, err := c.TargetFetcher.Fetch(objRef)
	if err != nil {
		klog.V(1).Infof("Skip creating TimeSeriesPrediction for %s/%s: no selector found (%v)", workload.Namespace, workload.Name, err)
		return nil
	}

	// Create TSP using template
	var tspSpec predictionv1alpha1.TimeSeriesPredictionSpec

	// Try to get from template
	templateName := "default" // Default template name
	if len(tspRule.Spec.Recommenders) > 0 {
		templateName = tspRule.Spec.Recommenders[0].Name
	}

	if template, exists := c.TemplateStore.GetTemplate(templateName); exists {
		tspSpec = predictionv1alpha1.TimeSeriesPredictionSpec{
			TargetRef: corev1.ObjectReference{
				APIVersion: workload.APIVersion,
				Kind:       workload.Kind,
				Namespace:  workload.Namespace,
				Name:       workload.Name,
			},
			PredictionWindowSeconds: template.PredictionWindowSeconds,
			PredictionMetrics:       getRenderTemplatePredictionMetrics(workload, template.PredictionMetrics),
		}
	} else {
		// Fallback to default configuration
		klog.Warningf("Template for %s not found, using default configuration", templateName)
		tspSpec = predictionv1alpha1.TimeSeriesPredictionSpec{
			TargetRef: corev1.ObjectReference{
				APIVersion: workload.APIVersion,
				Kind:       workload.Kind,
				Namespace:  workload.Namespace,
				Name:       workload.Name,
			},
			PredictionWindowSeconds: defaultPredictionWindowSeconds,
			PredictionMetrics: []predictionv1alpha1.PredictionMetric{
				getDefaultPredictionMetric(workload),
			},
		}
	}

	tsp := &predictionv1alpha1.TimeSeriesPrediction{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", tspNameBase, workload.Namespace),
			Namespace:    tspNamespace,
			Labels: map[string]string{
				"type":                          "tsp",
				"timeseriespredictionrule.uid":  string(tspRule.UID),
				"timeseriespredictionrule.name": tspRule.Name,
			},
		},
		Spec: tspSpec,
	}

	if err := c.Client.Create(ctx, tsp); err != nil {
		klog.V(1).Infof("Failed to create TimeSeriesPrediction: %v", err)
		return fmt.Errorf("failed to create TimeSeriesPrediction: %v", err)
	}

	klog.InfoS("Created TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
	c.Recorder.Eventf(tspRule, corev1.EventTypeNormal, "Created", "Created TimeSeriesPrediction %s/%s", tsp.Namespace, tsp.Name)
	return nil
}

func getRenderTemplatePredictionMetrics(workload workloadInfo, templateMetrics []predictionv1alpha1.PredictionMetric) []predictionv1alpha1.PredictionMetric {
	// Iterate PredictionMetrics and replace placeholders in expressions
	for _, metric := range templateMetrics {
		if metric.Type == predictionv1alpha1.ExpressionQueryMetricType && metric.ExpressionQuery != nil {
			// todo : Specify two %s, corresponding to namespace and pod name
			// Replace placeholders in expression
			metric.ExpressionQuery.Expression = fmt.Sprintf(
				metric.ExpressionQuery.Expression,
				workload.Namespace,
				utils.GetPodNameReg(workload.Name, workload.Kind),
			)
		}
	}
	return templateMetrics
}

func getDefaultPredictionMetric(workload workloadInfo) predictionv1alpha1.PredictionMetric {
	workloadCPUUsageExprTemplate := `max(rate(container_cpu_usage_seconds_total{namespace="%s", pod=~"%s", container=""}[3m]))`
	defaultPredictionMetric := predictionv1alpha1.PredictionMetric{
		ResourceIdentifier: "workload-cpu",
		Type:               predictionv1alpha1.ExpressionQueryMetricType,
		ExpressionQuery: &predictionv1alpha1.ExpressionQuery{
			Expression: fmt.Sprintf(workloadCPUUsageExprTemplate, workload.Namespace, utils.GetPodNameReg(workload.Name, workload.Kind)),
		},
		Algorithm: predictionv1alpha1.Algorithm{
			AlgorithmType: predictionv1alpha1.AlgorithmTypeSegmentedPercentile,
			SP: &predictionv1alpha1.SP{
				Histogram: predictionv1alpha1.HistogramConfig{
					BucketSize: "0.01",
					Epsilon:    "1e-10",
					MaxValue:   "10.0",
					HalfLife:   "0h",
				},
				MinSampleWeight: "1.0",
				SampleInterval:  "5m",
				HistoryLength:   "24h",
			},
		},
	}
	return defaultPredictionMetric
}

// updateTimeSeriesPrediction updates an existing TimeSeriesPrediction object
func (c *TimeSeriesPredictionRuleController) updateTimeSeriesPrediction(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule, tsp *predictionv1alpha1.TimeSeriesPrediction, workload workloadInfo) error {
	// Check if update is needed
	needsUpdate := false

	// Get the template applied to the current workload (if any), render new metrics list
	templateName := "default"
	if len(tspRule.Spec.Recommenders) > 0 {
		templateName = tspRule.Spec.Recommenders[0].Name
	}

	var newPredictionMetrics []predictionv1alpha1.PredictionMetric
	var predictionWindowSeconds int32
	if template, exists := c.TemplateStore.GetTemplate(templateName); exists {
		predictionWindowSeconds = template.PredictionWindowSeconds
		newPredictionMetrics = getRenderTemplatePredictionMetrics(workload, template.PredictionMetrics)
	} else {
		predictionWindowSeconds = defaultPredictionWindowSeconds
		newPredictionMetrics = []predictionv1alpha1.PredictionMetric{
			getDefaultPredictionMetric(workload),
		}
	}

	// Check if prediction metrics have changed
	if predictionWindowSeconds != tsp.Spec.PredictionWindowSeconds || !predictionMetricsEqual(tsp.Spec.PredictionMetrics, newPredictionMetrics) {
		tsp.Spec.PredictionMetrics = newPredictionMetrics
		needsUpdate = true
		klog.V(1).Infof("need update TimeSeriesPrediction %s/%s", tsp.Namespace, tsp.Name)
	}

	if needsUpdate {
		if err := c.Client.Update(ctx, tsp); err != nil {
			return fmt.Errorf("failed to update TimeSeriesPrediction: %v", err)
		}
		klog.InfoS("Updated TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
		c.Recorder.Eventf(tspRule, corev1.EventTypeNormal, "Updated", "Updated TimeSeriesPrediction %s/%s", tsp.Namespace, tsp.Name)
	}

	return nil
}

// predictionMetricsEqual checks if two prediction metrics lists are equal
func predictionMetricsEqual(a, b []predictionv1alpha1.PredictionMetric) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// workloadInfo contains basic information about a workload
type workloadInfo struct {
	Namespace  string
	Name       string
	APIVersion string
	Kind       string
}

// SetupWithManager sets up the controller with the Manager
func (c *TimeSeriesPredictionRuleController) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize dynamic client and discovery client
	config := mgr.GetConfig()
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %v", err)
	}

	c.DynamicClient = dynamicClient
	c.Discovery = discoveryClient

	return ctrl.NewControllerManagedBy(mgr).
		For(&predictionv1alpha1.TimeSeriesPredictionRule{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(c)
}

func getWorkloadMatchNameFromTSP(tsp *predictionv1alpha1.TimeSeriesPrediction) (string, error) {
	// Name format: namespace|apiVersion|kind|name, kind is lowercase
	if tsp == nil {
		return "", fmt.Errorf("tsp is nil")
	}
	targetRef := tsp.Spec.TargetRef
	if targetRef.Namespace == "" || targetRef.APIVersion == "" || targetRef.Kind == "" || targetRef.Name == "" {
		return "", fmt.Errorf("targetRef field is incomplete: namespace=%v, apiVersion=%v, kind=%v, name=%v", targetRef.Namespace, targetRef.APIVersion, targetRef.Kind, targetRef.Name)
	}
	workloadMatchName := fmt.Sprintf("%s|%s|%s|%s", targetRef.Namespace, targetRef.APIVersion, strings.ToLower(targetRef.Kind), targetRef.Name)
	return workloadMatchName, nil
}

func getWorkloadMatchNameFromWorkloadInfo(w *workloadInfo) string {
	// Name format: namespace|apiVersion|kind|name, kind is lowercase
	if w == nil {
		return ""
	}
	return fmt.Sprintf("%s|%s|%s|%s", w.Namespace, w.APIVersion, strings.ToLower(w.Kind), w.Name)
}
