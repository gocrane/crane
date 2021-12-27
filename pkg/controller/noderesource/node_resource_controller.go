/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package noderesource

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction/config"
)

const (
	ExtResourcePrefix = "ext-resource.node.gocrane.io/%s"
	MinDeltaRatio     = 0.1
	CoolDownTime      = 5 * time.Minute
)

// NodeResourceReconciler reconciles a NodeResource object
type NodeResourceReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=node-resource.crane.io,resources=noderesources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node-resource.crane.io,resources=noderesources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node-resource.crane.io,resources=noderesources/finalizers,verbs=update

func (r *NodeResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("node resource reconcile", "node-resource", req.NamespacedName, req.Name)

	tsp := &predictionapi.TimeSeriesPrediction{}
	// get TimeSeriesPrediction result
	err := r.Client.Get(ctx, req.NamespacedName, tsp)
	if err != nil {
		r.Recorder.Event(tsp, v1.EventTypeNormal, "FailedGetTimeSeriesPrediction", err.Error())
		r.Log.Error(err, "get TimeSeriesPrediction error", "TimeSeriesPrediction", req.NamespacedName, req.Name)
		return ctrl.Result{}, err
	}

	// get current node info
	target := tsp.Spec.TargetRef
	if target.Kind != config.TargetKindNode {
		return ctrl.Result{}, nil
	}
	node, err, retry := r.FindTargetNode(ctx, tsp)
	if err != nil {
		r.Recorder.Event(tsp, v1.EventTypeNormal, "FailedGetTargetNode", err.Error())
		if !retry {
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	nodeCopy := node.DeepCopy()
	r.BuildNodeStatus(tsp, nodeCopy)
	if !equality.Semantic.DeepEqual(&node.Status, &nodeCopy.Status) {
		// update Node status extend-resource info
		// TODO fix: strategic merge patch kubernetes
		if err := r.Client.Status().Update(context.TODO(), nodeCopy); err != nil {
			r.Recorder.Event(tsp, v1.EventTypeNormal, "FailedUpdateNodeExtendResource", err.Error())
			r.Log.Error(err, "update Node status extend-resource error: %v", "node", nodeCopy.Name)
			return ctrl.Result{}, err
		}
		r.Recorder.Event(tsp, v1.EventTypeNormal, "UpdateNode", "Update Node Extend Resource Success")
	}
	return ctrl.Result{}, nil
}

func (r *NodeResourceReconciler) FindTargetNode(ctx context.Context, tsp *predictionapi.TimeSeriesPrediction) (*v1.Node, error, bool) {
	address := tsp.Spec.TargetRef.Name
	if address == "" {
		return nil, fmt.Errorf("target is not specified"), false
	}
	nodeList := &v1.NodeList{}
	if err := r.Client.List(ctx, nodeList); err != nil {
		r.Log.Error(err, "list node error")
		return nil, err, true
	}
	// the reason we use node ip instead of node name as the target name is
	// some monitoring system does not persist node name
	for _, n := range nodeList.Items {
		for _, addr := range n.Status.Addresses {
			if addr.Address == address {
				r.Log.Info("Found target node", "nodeName", n.Name)
				return &n, nil, false
			}
		}
	}
	return nil, fmt.Errorf("target [%s] not found", address), false
}

func (r *NodeResourceReconciler) BuildNodeStatus(tsp *predictionapi.TimeSeriesPrediction, node *v1.Node) {
	idToResourceMap := map[string]*v1.ResourceName{}
	for _, metrics := range tsp.Spec.PredictionMetrics {
		if metrics.ResourceQuery == nil {
			continue
		}
		idToResourceMap[metrics.ResourceIdentifier] = metrics.ResourceQuery
	}
	// build node status
	nextPredictionResourceStatus := &tsp.Status
	for _, metrics := range nextPredictionResourceStatus.PredictionMetrics {
		resourceName, exists := idToResourceMap[metrics.ResourceIdentifier]
		if !exists {
			continue
		}
		for _, timeSeries := range metrics.Prediction {
			var maxUsage, nextUsage float64
			var nextUsageFloat float64
			var err error
			for _, sample := range timeSeries.Samples {
				if nextUsageFloat, err = strconv.ParseFloat(sample.Value, 64); err != nil {
					r.Log.Error(err, "parse extend resource value error", "value", sample.Value)
					continue
				}
				nextUsage = nextUsageFloat
				if maxUsage < nextUsage {
					maxUsage = nextUsage
				}
			}
			var nextRecommendation float64
			switch *resourceName {
			case v1.ResourceCPU:
				// cpu need to be scaled to m as ext resource cannot be decimal
				nextRecommendation = (float64(node.Status.Allocatable.Cpu().Value()) - maxUsage) * 1000
			case v1.ResourceMemory:
				// unit of memory in prometheus is in Ki, need to be converted to byte
				nextRecommendation = float64(node.Status.Allocatable.Memory().Value()) - (maxUsage * 1000)
			default:
				continue
			}
			if nextRecommendation <= 0 {
				r.Log.Info("Unexpected recommendation", "nodeName", node.Name, "maxUsage", maxUsage, "nextRecommendation", nextRecommendation)
				continue
			}
			extResourceName := fmt.Sprintf(ExtResourcePrefix, string(*resourceName))
			resValue, exists := node.Status.Capacity[v1.ResourceName(extResourceName)]
			if exists && resValue.Value() != 0 &&
				math.Abs(float64(resValue.Value())-
					nextRecommendation)/float64(resValue.Value()) <= MinDeltaRatio {
				continue
			}
			node.Status.Capacity[v1.ResourceName(extResourceName)] =
				*resource.NewQuantity(int64(nextRecommendation), resource.DecimalSI)
			node.Status.Allocatable[v1.ResourceName(extResourceName)] =
				*resource.NewQuantity(int64(nextRecommendation), resource.DecimalSI)
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&predictionapi.TimeSeriesPrediction{}).
		Complete(r)
}
