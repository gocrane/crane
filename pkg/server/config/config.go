package config

import (
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/server/service/dashboard"
)

type Config struct {
	Mode        string `json:"mode"`
	BindAddress string `json:"bindAddress"`
	BindPort    int    `json:"bindPort"`

	EnableProfiling bool `json:"profiling"`
	EnableMetrics   bool `json:"enableMetrics"`

	EnableGrafana bool                     `json:"enableGrafana"`
	GrafanaConfig *dashboard.GrafanaConfig `json:"grafanaConfig"`

	KubeConfig *rest.Config    `json:"kubeConfig"`
	Scheme     *runtime.Scheme `json:"scheme"`
	Client     client.Client   `json:"client"`
	RestMapper meta.RESTMapper `json:"restMapper"`

	StoreType string `json:"storeType"`

	PredictorMgr predictormgr.Manager
	Api          promapiv1.API
}

func NewServerConfig() *Config {
	return &Config{}
}
