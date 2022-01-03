package features

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
)

const (
	// CraneAutoscaling enables the autoscaling features for workloads.
	CraneAutoscaling featuregate.Feature = "Autoscaling"
)

var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	CraneAutoscaling: {Default: true, PreRelease: featuregate.Alpha},
}

func init() {
	runtime.Must(utilfeature.DefaultMutableFeatureGate.Add(defaultFeatureGates))
}
