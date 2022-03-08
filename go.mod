module github.com/gocrane/crane

go 1.17

require (
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gocrane/api v0.0.0-20220303034349-4196a7186df2
	github.com/google/cadvisor v0.39.2
	github.com/mjibson/go-dsp v0.0.0-20180508042940-11479a337f12
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/shirou/gopsutil v3.21.10+incompatible
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	google.golang.org/grpc v1.43.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/apiserver v0.22.3
	k8s.io/autoscaler/vertical-pod-autoscaler v0.10.0
	k8s.io/client-go v0.22.3
	k8s.io/component-base v0.22.3
	k8s.io/cri-api v0.22.3
	k8s.io/klog/v2 v2.9.0
	k8s.io/kubernetes v1.22.3
	k8s.io/metrics v0.22.3
	sigs.k8s.io/controller-runtime v0.10.2
	sigs.k8s.io/custom-metrics-apiserver v1.22.0
)

require (
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-gonic/gin v1.7.7
	github.com/golang/mock v1.5.0
	github.com/grafana-tools/sdk v0.0.0-20211220201350-966b3088eec9
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/zsais/go-gin-prometheus v0.1.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/gcfg.v1 v1.2.0
	k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8 // indirect
)

require (
	cloud.google.com/go v0.84.0 // indirect
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/afero v1.6.0 // indirect
	go.etcd.io/etcd/client/v2 v2.305.1 // indirect
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/tools v0.1.8 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
)

replace (
	github.com/grafana-tools/sdk => github.com/csmarchbanks/sdk v0.0.0-20220120205302-870d00a83f4e
	golang.org/x/net => github.com/golang/net v0.0.0-20210825183410-e898025ed96a
	k8s.io/api => k8s.io/api v0.22.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.3
	k8s.io/apiserver => k8s.io/apiserver v0.22.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.3
	k8s.io/client-go => k8s.io/client-go v0.22.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.3
	k8s.io/code-generator => k8s.io/code-generator v0.22.3
	k8s.io/component-base => k8s.io/component-base v0.22.3
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.3
	k8s.io/cri-api => k8s.io/cri-api v0.22.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.3
	k8s.io/kubectl => k8s.io/kubectl v0.22.3
	k8s.io/kubelet => k8s.io/kubelet v0.22.3
	k8s.io/kubernetes => k8s.io/kubernetes v1.22.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.3
	k8s.io/metrics => k8s.io/metrics v0.22.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.3
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.3
)
