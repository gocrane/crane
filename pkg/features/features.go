package features

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
)

const (
	// CraneAutoscaling enables the autoscaling features for workloads.
	CraneAutoscaling featuregate.Feature = "Autoscaling"

	// CraneAnalysis enables analysis features, including analytics and recommendation.
	CraneAnalysis featuregate.Feature = "Analysis"

	// CraneNodeResource enables the node resource features.
	CraneNodeResource featuregate.Feature = "NodeResource"

	// CraneNodeResourceTopology enables node resource topology features.
	CraneNodeResourceTopology featuregate.Feature = "NodeResourceTopology"

	// CranePodResource enables the pod resource features.
	CranePodResource featuregate.Feature = "PodResource"

	// CraneClusterNodePrediction enables the cluster node prediction features.
	CraneClusterNodePrediction featuregate.Feature = "ClusterNodePrediction"

	// CraneTimeSeriesPrediction enables the time series prediction features.
	CraneTimeSeriesPrediction featuregate.Feature = "TimeSeriesPrediction"

	// CraneCpuSetManager enables the cpuset manger features.
	CraneCpuSetManager featuregate.Feature = "CpusetManager"
)

var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	CraneAutoscaling:           {Default: true, PreRelease: featuregate.Alpha},
	CraneAnalysis:              {Default: true, PreRelease: featuregate.Alpha},
	CraneNodeResource:          {Default: true, PreRelease: featuregate.Alpha},
	CraneNodeResourceTopology:  {Default: false, PreRelease: featuregate.Alpha},
	CranePodResource:           {Default: true, PreRelease: featuregate.Alpha},
	CraneClusterNodePrediction: {Default: false, PreRelease: featuregate.Alpha},
	CraneTimeSeriesPrediction:  {Default: true, PreRelease: featuregate.Alpha},
	CraneCpuSetManager:         {Default: false, PreRelease: featuregate.Alpha},
}

func init() {
	runtime.Must(utilfeature.DefaultMutableFeatureGate.Add(defaultFeatureGates))
}
