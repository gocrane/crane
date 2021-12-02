module github.com/gocrane/crane

go 1.17

require (
	github.com/go-logr/logr v0.4.0
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

require (
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gocrane/api v0.0.0-20211202040734-84c4d8bf59d6
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/mjibson/go-dsp v0.0.0-20180508042940-11479a337f12
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/tools v0.1.5 // indirect
	k8s.io/autoscaler/vertical-pod-autoscaler v0.9.2
	k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8 // indirect
)
