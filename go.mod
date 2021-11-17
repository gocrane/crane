module github.com/gocrane-io/crane

go 1.17

require (
	github.com/go-logr/logr v0.4.0
	github.com/gocrane-io/api v0.0.2
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/apiserver v0.22.2
	k8s.io/client-go v0.22.3
	k8s.io/component-base v0.22.2
	k8s.io/klog/v2 v2.9.0
	k8s.io/metrics v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
	sigs.k8s.io/custom-metrics-apiserver v1.22.0
)
