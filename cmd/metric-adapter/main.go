package main

import (
	"flag"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	basecmd "sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	generatedopenapi "github.com/gocrane/api/pkg/generated/openapi"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/metricprovider"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(autoscalingapi.AddToScheme(scheme))
	utilruntime.Must(predictionapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

}

type MetricAdapter struct {
	basecmd.AdapterBase

	// Message is printed on successful startup
	Message string
}

func (a *MetricAdapter) makeProvider(remoteAdapter *metricprovider.RemoteAdapter, config *rest.Config, client client.Client) (provider.CustomMetricsProvider, error) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create kube client: %v", err)
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: kubeClient.CoreV1().Events(""),
	})
	recorder := broadcaster.NewRecorder(scheme, corev1.EventSource{Component: "crane-metric-adapter"})

	return metricprovider.NewMetricProvider(client, remoteAdapter, recorder), nil
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &MetricAdapter{}

	cmd.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(scheme))
	cmd.OpenAPIConfig.Info.Title = "crane-metric-adapter"
	cmd.OpenAPIConfig.Info.Version = "1.0.0"

	var enableRemoteAdapter bool
	var remoteAdapterServiceNamespace string
	var remoteAdapterServiceName string
	var remoteAdapterServicePort int
	var apiQps int
	var apiBurst int

	cmd.Flags().StringVar(&cmd.Message, "msg", "Starting adapter...", "startup message")
	cmd.Flags().BoolVar(&enableRemoteAdapter, "remote-adapter", false, "Enable a remote adapter to provide a set of custom metrics")
	cmd.Flags().StringVar(&remoteAdapterServiceNamespace, "remote-adapter-service-namespace", "", "Namespace of remote adapter's service")
	cmd.Flags().StringVar(&remoteAdapterServiceName, "remote-adapter-service-name", "", "Name of remote adapter's service")
	cmd.Flags().IntVar(&remoteAdapterServicePort, "remote-adapter-service-port", 6443, "Port of remote adapter's service")
	cmd.Flags().IntVar(&apiQps, "api-qps", 300, "QPS of rest config.")
	cmd.Flags().IntVar(&apiBurst, "api-burst", 400, "Burst of rest config.")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags
	if err := cmd.Flags().Parse(os.Args); err != nil {
		return
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		klog.Exitf("Failed to get config: %v", err)
	}

	config.QPS = float32(apiQps)
	config.Burst = apiBurst

	clientOptions := client.Options{Scheme: scheme}
	client, err := client.New(config, clientOptions)
	if err != nil {
		klog.Exitf("Failed to get client: %v", err)
	}

	var remoteAdapter *metricprovider.RemoteAdapter
	if enableRemoteAdapter {
		klog.Infof("Enable remote adapter: %s/%s", remoteAdapterServiceNamespace, remoteAdapterServiceName)
		remoteAdapter, err = metricprovider.NewRemoteAdapter(remoteAdapterServiceNamespace, remoteAdapterServiceName, remoteAdapterServicePort, config, client)
		if err != nil {
			klog.Exitf("Failed to create remote adapter: %v", err)
		}
	}

	metricProvider, err := cmd.makeProvider(remoteAdapter, config, client)
	if err != nil {
		klog.Error(err, "Failed to make provider")
		os.Exit(1)
	}
	cmd.WithCustomMetrics(metricProvider)

	klog.Infof(cmd.Message)
	if err := cmd.Run(wait.NeverStop); err != nil {
		klog.Error(err, "Failed to run metrics adapter")
		os.Exit(1)
	}
}
