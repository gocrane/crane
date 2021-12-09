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
	"github.com/go-logr/logr"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// NodeResourceReconciler reconciles a NodeResource object
type NodeResourceReconciler struct {
	client.Client
	Scheme 		*runtime.Scheme
	Log         logr.Logger
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	scaleClient scale.ScalesGetter
	K8SVersion  *version.Version
}

//+kubebuilder:rbac:groups=node-resource.crane.io,resources=noderesources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=node-resource.crane.io,resources=noderesources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node-resource.crane.io,resources=noderesources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *NodeResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("node resource reconcile", "node-resource", req.NamespacedName, req.Name)

	nodePre := &predictionapi.NodePrediction{}
	// 1.取预测结果  nodePre.Status
	err := r.Client.Get(ctx, req.NamespacedName, nodePre)
	if err != nil {
		r.Recorder.Event(nodePre, v1.EventTypeNormal, "FailedGetNodePrediction", err.Error())
		r.Log.Error(err,"get node prediction error", "node-prediction", req.NamespacedName, req.Name)
		return ctrl.Result{}, err
	}
	// TODO node timeserices api resp extract
	// 2.获取node信息
	node := &v1.Node{}
	nodeKey := client.ObjectKey{
		Namespace: "",
		Name: "",
	}
	if err := r.Client.Get(ctx, nodeKey, node); err != nil {
		r.Recorder.Event(node, v1.EventTypeNormal, "FailedGetNode", err.Error())
		r.Log.Error(err, "get node error", "node", nodeKey.Namespace, nodeKey.Name)
		return ctrl.Result{}, err
	}
	nodeCopy := node.DeepCopy()

	// 3.构建 node status
	nextPredictionResourceStatus := &nodePre.Status
	//resources := make(map[v1.ResourceName]resource.Quantity)
	for key, nextPossible := range nextPredictionResourceStatus.NextPossible {
		for _, timeSeries := range nextPossible {
			var resourceValue int64
			if result, err := strconv.ParseInt(timeSeries.Value, 10, 64); err != nil {
				r.Log.Error(err, "parse extend resource value error, resource value: %s", timeSeries.Value)
				resourceValue = result
			}
			if (key == string(predictionapi.ResourceCPU)) {
				nodeCopy.Status.Capacity["ext-resource.node.crane.io/cpu"] = *resource.NewQuantity(resourceValue, resource.DecimalSI)
			}

			if (key == string(predictionapi.ResourceMemory)) {
				nodeCopy.Status.Capacity["ext-resource.node.crane.io/memory"] = *resource.NewQuantity(resourceValue, resource.DecimalSI)
			}
		}
	}

	// 4.更新 Node status extend-resource 信息
	// https://kubernetes.io/zh/docs/tasks/administer-cluster/extended-resource-node/
	// TODO fix: strategic merge patch kubernetes
	if err := r.Client.Status().Update(context.TODO(), nodeCopy); err != nil {
		r.Recorder.Event(node, v1.EventTypeNormal, "FailedUpdateNodeExtendResource", err.Error())
		r.Log.Error(err, "update Node status extend-resource error: %v", err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}


// SetupWithManager sets up the controller with the Manager.
func (r *NodeResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&predictionapi.NodePrediction{}).
		Complete(r)
}