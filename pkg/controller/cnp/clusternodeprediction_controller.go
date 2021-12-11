package cnp

import (
	"context"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
)

type ClusterNodePredictionController struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	Recorder    record.EventRecorder
	scaleClient scale.ScalesGetter
	K8SVersion  *version.Version
}

func (c *ClusterNodePredictionController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Log for controller information
	c.Log.Info("got", "cnp", req.NamespacedName)

	// Get cnp object, Ignore object not found event
	var cnp predictionapi.ClusterNodePrediction
	if err := c.Client.Get(ctx, req.NamespacedName, &cnp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Match nodes according to labels
	var nodeList v1.NodeList
	var matchingLabels client.MatchingLabels = cnp.Spec.NodeSelector
	opts := []client.ListOption{
		matchingLabels,
	}
	if err := c.Client.List(ctx, &nodeList, opts...); err != nil {
		return ctrl.Result{}, err
	}

	// When labels no match any node
	// Set the status, Then reconcile func end
	if len(nodeList.Items) == 0 {
		status := predictionapi.ClusterNodePredictionStatus{
			CurrentNumberCreated: 0,
			DesiredNumberCreated: 0,
			Conditions: nil,
		}
		if err := c.updateStatus(ctx, &cnp, status); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Mutate TimeSeriesPrediction according to the number of node list
	var successCount int
	for _, node := range nodeList.Items {
		var tsp predictionapi.TimeSeriesPrediction
		tsp.Name = cnp.Name + "-" + node.Name
		tsp.Namespace = cnp.Namespace
		if _, err := ctrl.CreateOrUpdate(ctx, c.Client, &tsp, func() error {
			c.mutateTimeSeriesPrediction(&cnp, &tsp, &node)
			successCount++
			return controllerutil.SetControllerReference(&cnp, &tsp, c.Scheme)
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set the status, Then reconcile func end
	status := predictionapi.ClusterNodePredictionStatus{
		CurrentNumberCreated:successCount,
		DesiredNumberCreated: len(nodeList.Items),
	 }
	if err := c.updateStatus(ctx, &cnp, status); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (c *ClusterNodePredictionController) updateStatus(ctx context.Context, cnp *predictionapi.ClusterNodePrediction, status predictionapi.ClusterNodePredictionStatus) error {
	cnpCopy := cnp.DeepCopy()
	cnpCopy.Status = status
	return c.Client.Status().Update(ctx, cnpCopy)
}

func (c *ClusterNodePredictionController) mutateTimeSeriesPrediction(cnp *predictionapi.ClusterNodePrediction, tsp *predictionapi.TimeSeriesPrediction, node *v1.Node) {
	tsp.Spec = predictionapi.TimeSeriesPredictionSpec{
		PredictionMetrics: cnp.Spec.PredictionTemplate.Spec.PredictionMetrics,
		TargetRef: v1.ObjectReference{
			Kind:       node.Kind,
			APIVersion: node.APIVersion,
			Name:       node.Name,
		},
		PredictionWindowSeconds: cnp.Spec.PredictionTemplate.Spec.PredictionWindowSeconds,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (c *ClusterNodePredictionController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&predictionapi.ClusterNodePrediction{}).
		Owns(&predictionapi.TimeSeriesPrediction{}).
		Complete(c)
}
