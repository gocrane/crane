package config

import (
	"k8s.io/client-go/rest"

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

	KubeRestConfig *rest.Config `json:"KubeRestConfig"`
	StoreType      string       `json:"storeType"`
}

func NewServerConfig() *Config {
	return &Config{}
}
