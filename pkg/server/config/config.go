package config

import (
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

	KubeConfig string `json:"kubeConfig"`
}

func NewServerConfig() *Config {
	return &Config{}
}
