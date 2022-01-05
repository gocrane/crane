package features

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
)

const (
	// CraneAutoscaling enables the autoscaling features for workloads.
	CraneAutoscaling featuregate.Feature = "Autoscaling"

	// CraneNodeResource enables the node resource features.
	CraneNodeResource featuregate.Feature = "NodeResource"

	// CraneClusterNodePrediction enables the cluster node prediction features.
	CraneClusterNodePrediction featuregate.Feature = "ClusterNodePrediction"
)

var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	CraneAutoscaling:           {Default: true, PreRelease: featuregate.Alpha},
	CraneNodeResource:          {Default: false, PreRelease: featuregate.Alpha},
	CraneClusterNodePrediction: {Default: false, PreRelease: featuregate.Alpha},
}

func init() {
	runtime.Must(utilfeature.DefaultMutableFeatureGate.Add(defaultFeatureGates))
}
