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

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/gocrane/crane/pkg/ensurance/statestore"
	"github.com/gocrane/crane/pkg/utils/clogs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/ensurance/nep"
)

// NodeQOSEnsurancePolicyController reconciles a NodeQOSEnsurancePolicy object
type NodeQOSEnsurancePolicyController struct {
	client.Client
	Scheme     *runtime.Scheme
	Log        logr.Logger
	RestMapper meta.RESTMapper
	Recorder   record.EventRecorder
	Cache      *nep.NodeQOSEnsurancePolicyCache
	StateStore statestore.StateStore
}

//+kubebuilder:rbac:groups=ensurance.crane.io.crane.io,resources=nodeqosensurancepolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ensurance.crane.io.crane.io,resources=nodeqosensurancepolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ensurance.crane.io.crane.io,resources=nodeqosensurancepolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeQOSEnsurancePolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (c *NodeQOSEnsurancePolicyController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("got", "nep", req.NamespacedName)

	nep := &ensuranceapi.NodeQOSEnsurancePolicy{}
	if err := c.Client.Get(ctx, req.NamespacedName, nep); err != nil {
		// The resource may be deleted, in this case we need to stop the processing.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{Requeue: true}, err
	}

	if !nep.DeletionTimestamp.IsZero() {
		if err := c.delete(nep); err != nil {
			return ctrl.Result{}, err
		}
	}

	return c.reconcileNep(nep)
}

// SetupWithManager sets up the controller with the Manager.
func (c *NodeQOSEnsurancePolicyController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ensuranceapi.NodeQOSEnsurancePolicy{}).
		Complete(c)
}

func (c *NodeQOSEnsurancePolicyController) reconcileNep(nep *ensuranceapi.NodeQOSEnsurancePolicy) (ctrl.Result, error) {

	if !c.Cache.Exist(nep.Name) {
		if err := c.create(nep); err != nil {
			return ctrl.Result{}, err
		}
		c.Cache.Set(nep)
	} else {
		if err := c.update(nep); err != nil {
			return ctrl.Result{}, err
		}
		c.Cache.Set(nep)
	}

	return ctrl.Result{}, nil
}

func (c *NodeQOSEnsurancePolicyController) create(nep *ensuranceapi.NodeQOSEnsurancePolicy) error {
	// step1: add metrics
	for _, v := range nep.Spec.ObjectiveEnsurance {
		var key = GenerateEnsuranceQosNodePolicyKey(nep.Name, v.AvoidanceActionName)
		c.StateStore.AddMetric(key, v.MetricRule.Metric.Name, v.MetricRule.Metric.Selector)
	}
	return nil
}

func (c *NodeQOSEnsurancePolicyController) update(nep *ensuranceapi.NodeQOSEnsurancePolicy) error {
	if nepOld, ok := c.Cache.Get(nep.Name); ok {
		// step1 compare all objectiveEnsurance
		// step2 if eth1 objectiveEnsurance metric changed, update the policy status
		// step3 delete the old metric and add the new metric
		// step4 if failed, delete all metrics for this policy
		// step5 if succeed, update the policy status
		clogs.Log().V(6).Info("nepOld %v", nepOld)
	}
	return fmt.Errorf("update nep(%s),no found", nep.Name)
}

func (c *NodeQOSEnsurancePolicyController) delete(nep *ensuranceapi.NodeQOSEnsurancePolicy) error {
	// step1: delete metrics from cache nep
	if nepOld, ok := c.Cache.Get(nep.Name); ok {
		for _, v := range nepOld.Nep.Spec.ObjectiveEnsurance {
			var key = GenerateEnsuranceQosNodePolicyKey(nep.Name, v.AvoidanceActionName)
			c.StateStore.DeleteMetric(key)
		}
	}

	// step2: delete metrics from  nep
	for _, v := range nep.Spec.ObjectiveEnsurance {
		var key = GenerateEnsuranceQosNodePolicyKey(nep.Name, v.AvoidanceActionName)
		c.StateStore.DeleteMetric(key)
	}

	return nil
}

func GenerateEnsuranceQosNodePolicyKey(policyName string, avoidanceActionName string) string {
	return strings.Join([]string{"node", policyName, avoidanceActionName}, ".")
}
