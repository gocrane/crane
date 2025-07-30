package timeseriesprediction

import (
	"context"
	"fmt"
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

	predictionv1alpha1 "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	tspNamespace = "crane-system"
	tspNameBase  = "tsp"
)

// TimeSeriesPredictionRuleController reconciles TimeSeriesPredictionRule objects
type TimeSeriesPredictionRuleController struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	DynamicClient dynamic.Interface
	Discovery     discovery.DiscoveryInterface
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

	// 处理一次性任务
	lastUpdateTime := tspRule.Status.LastUpdateTime
	if len(strings.TrimSpace(tspRule.Spec.RunInterval)) == 0 {
		if lastUpdateTime != nil {
			// 一次性任务已完成
			return ctrl.Result{}, nil
		}
	}

	// 检查是否已到执行时间
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
			klog.V(1).InfoS("Will re-sync", "after", planingTime.Sub(now))
			return ctrl.Result{
				RequeueAfter: planingTime.Sub(now),
			}, nil
		}
	}

	// 调用核心同步逻辑
	finished, err := c.syncTimeSeriesPredictionRule(ctx, tspRule)
	if err != nil {
		c.Recorder.Event(tspRule, corev1.EventTypeWarning, "SyncFailed", err.Error())
		klog.ErrorS(err, "Failed to sync TimeSeriesPredictionRule", "NamespacedName", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// 更新状态
	if finished {
		updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			latestTspRule := &predictionv1alpha1.TimeSeriesPredictionRule{}
			if err := c.Client.Get(ctx, req.NamespacedName, latestTspRule); err != nil {
				klog.V(1).Infof("Failed to get latest TimeSeriesPredictionRule: %v", err)
				return err
			}

			// 更新LastUpdateTime
			now := metav1.Now()
			latestTspRule.Status.LastUpdateTime = &now

			return c.Client.Update(ctx, latestTspRule)
		})

		if updateErr != nil {
			c.Recorder.Event(tspRule, corev1.EventTypeWarning, "StatusUpdateFailed", updateErr.Error())
			klog.ErrorS(updateErr, "Failed to update TimeSeriesPredictionRule status", "NamespacedName", req.NamespacedName)
			return ctrl.Result{}, updateErr
		}
	}

	// 根据同步结果设置重新协调时间
	requeueAfter := 30 * time.Second
	if finished && len(strings.TrimSpace(tspRule.Spec.RunInterval)) != 0 {
		klog.V(4).InfoS("Will re-sync", "after", interval)
		requeueAfter = interval
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// syncTimeSeriesPredictionRule 实现核心同步逻辑
func (c *TimeSeriesPredictionRuleController) syncTimeSeriesPredictionRule(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule) (bool, error) {
	klog.V(1).Infof("Syncing TimeSeriesPredictionRule: %s/%s", tspRule.Namespace, tspRule.Name)

	// 获取所有匹配的workload
	workloads, err := c.getMatchingWorkloads(ctx, tspRule)
	if err != nil {
		return false, fmt.Errorf("failed to get matching workloads: %v", err)
	}
	klog.V(1).Infof("Found %d matching workloads for rule %s/%s", len(workloads), tspRule.Namespace, tspRule.Name)

	// 获取当前关联的TSP对象
	var currentTSPs predictionv1alpha1.TimeSeriesPredictionList
	labelSelector := labels.SelectorFromSet(labels.Set{
		"timeseriespredictionrule.name": tspRule.Name,
	})
	err = c.List(ctx, &currentTSPs, &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return false, fmt.Errorf("failed to list existing TimeSeriesPredictions: %v", err)
	}

	// 创建TSP对象索引用于快速查找
	tspIndex := make(map[string]*predictionv1alpha1.TimeSeriesPrediction)
	for i := range currentTSPs.Items {
		tsp := &currentTSPs.Items[i]
		key := fmt.Sprintf("%s/%s", tsp.Namespace, tsp.Spec.TargetRef.Name)
		tspIndex[key] = tsp
	}

	// 创建或更新TSP对象
	var wg sync.WaitGroup
	errCh := make(chan error, len(workloads))

	for _, workload := range workloads {
		wg.Add(1)
		go func(wl workloadInfo) {
			defer wg.Done()
			key := fmt.Sprintf("%s/%s", wl.Namespace, wl.Name)
			klog.V(1).Infof("Processing workload: %s", key)

			// 检查是否已存在TSP对象
			existingTSP, exists := tspIndex[key]
			if exists {
				klog.V(1).Infof("Updating existing TimeSeriesPrediction for workload: %s", key)
				// 更新现有TSP对象
				if err := c.updateTimeSeriesPrediction(ctx, tspRule, existingTSP, wl); err != nil {
					errCh <- fmt.Errorf("failed to update TimeSeriesPrediction for %s: %v", key, err)
				}
				delete(tspIndex, key) // 从索引中移除，保留的将是孤儿对象
				return
			}

			klog.V(1).Infof("Creating new TimeSeriesPrediction for workload: %s", key)
			// 创建新的TSP对象
			if err := c.createTimeSeriesPrediction(ctx, tspRule, wl); err != nil {
				errCh <- fmt.Errorf("failed to create TimeSeriesPrediction for %s: %v", key, err)
			}
		}(workload)
	}

	wg.Wait()
	close(errCh)

	// 处理错误
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return false, fmt.Errorf("encountered %d errors during sync", len(errors))
	}

	// 清理孤儿TSP对象(TSP对象的目标workload已经消失)
	for _, tsp := range tspIndex {
		klog.V(1).Infof("Deleting orphan TimeSeriesPrediction: %s/%s", tsp.Namespace, tsp.Name)
		if err := c.Client.Delete(ctx, tsp); err != nil {
			klog.ErrorS(err, "Failed to delete orphan TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
		} else {
			klog.InfoS("Deleted orphan TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
		}
	}

	return true, nil
}

// getMatchingWorkloads 获取匹配的workload列表
func (c *TimeSeriesPredictionRuleController) getMatchingWorkloads(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule) ([]workloadInfo, error) {
	var workloads []workloadInfo

	for _, rs := range tspRule.Spec.ResourceSelectors {
		if rs.Kind == "" {
			return nil, fmt.Errorf("empty kind in resource selector")
		}

		gvr, err := utils.GetGroupVersionResource(c.Discovery, rs.APIVersion, rs.Kind)
		if err != nil {
			return nil, fmt.Errorf("failed to get GVR for %s/%s: %v", rs.APIVersion, rs.Kind, err)
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
			// 按名称过滤
			if rs.Name != "" && u.GetName() != rs.Name {
				continue
			}

			// 按标签选择器过滤
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

// createTimeSeriesPrediction 创建新的TimeSeriesPrediction对象
func (c *TimeSeriesPredictionRuleController) createTimeSeriesPrediction(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule, workload workloadInfo) error {
	tsp := &predictionv1alpha1.TimeSeriesPrediction{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", tspNameBase, workload.Namespace), // Name不作为索引，使用GenerateName保证不重复
			Namespace:    tspNamespace,
			Labels: map[string]string{
				"timeseriespredictionrule.uid":  string(tspRule.UID),
				"timeseriespredictionrule.name": tspRule.Name,
			},
		},
		Spec: predictionv1alpha1.TimeSeriesPredictionSpec{
			TargetRef: corev1.ObjectReference{
				APIVersion: workload.APIVersion,
				Kind:       workload.Kind,
				Namespace:  workload.Namespace,
				Name:       workload.Name,
			},
			PredictionWindowSeconds: 86400,
			PredictionMetrics: []predictionv1alpha1.PredictionMetric{
				getDefaultPredictionMetric(workload),
			},
		},
	}

	if err := c.Client.Create(ctx, tsp); err != nil {
		klog.V(1).Infof("Failed to create TimeSeriesPrediction: %v", err)
		return fmt.Errorf("failed to create TimeSeriesPrediction: %v", err)
	}

	klog.InfoS("Created TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
	c.Recorder.Eventf(tspRule, corev1.EventTypeNormal, "Created", "Created TimeSeriesPrediction %s/%s", tsp.Namespace, tsp.Name)
	return nil
}

func getDefaultPredictionMetric(workload workloadInfo) predictionv1alpha1.PredictionMetric {
	workloadCPUUsageExprTemplate := `max(irate(container_cpu_usage_seconds_total{namespace="%s", pod=~"%s", container!="" }[3m]))`
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

// updateTimeSeriesPrediction 更新现有的TimeSeriesPrediction对象
func (c *TimeSeriesPredictionRuleController) updateTimeSeriesPrediction(ctx context.Context, tspRule *predictionv1alpha1.TimeSeriesPredictionRule, tsp *predictionv1alpha1.TimeSeriesPrediction, workload workloadInfo) error {
	// 检查是否需要更新
	needsUpdate := false

	//// 检查预测指标是否变化
	//if !predictionMetricsEqual(tsp.Spec.PredictionMetrics, tspRule.Spec.PredictionMetrics) {
	//	tsp.Spec.PredictionMetrics = tspRule.Spec.PredictionMetrics
	//	needsUpdate = true
	//}

	if needsUpdate {
		if err := c.Client.Update(ctx, tsp); err != nil {
			return fmt.Errorf("failed to update TimeSeriesPrediction: %v", err)
		}
		klog.InfoS("Updated TimeSeriesPrediction", "NamespacedName", types.NamespacedName{Namespace: tsp.Namespace, Name: tsp.Name})
		c.Recorder.Eventf(tspRule, corev1.EventTypeNormal, "Updated", "Updated TimeSeriesPrediction %s/%s", tsp.Namespace, tsp.Name)
	}

	return nil
}

// predictionMetricsEqual 检查两个预测指标列表是否相等
func predictionMetricsEqual(a, b []predictionv1alpha1.PredictionMetric) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// workloadInfo 包含workload的基本信息
type workloadInfo struct {
	Namespace  string
	Name       string
	APIVersion string
	Kind       string
}

// SetupWithManager sets up the controller with the Manager
func (c *TimeSeriesPredictionRuleController) SetupWithManager(mgr ctrl.Manager) error {
	// 初始化动态客户端和发现客户端
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
		For(&predictionv1alpha1.TimeSeriesPredictionRule{}).
		Complete(c)
}
