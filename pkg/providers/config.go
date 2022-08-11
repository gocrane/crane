package providers

import (
	"net/http"
	"time"
)

// PromConfig represents the config of prometheus
type PromConfig struct {
	Address            string
	Timeout            time.Duration
	KeepAlive          time.Duration
	InsecureSkipVerify bool
	Auth               ClientAuth

	QueryConcurrency            int
	BRateLimit                  bool
	MaxPointsLimitPerTimeSeries int
	FederatedClusterScope       bool
	// for thanos query, it must when use thanos as query source https://thanos.io/tip/components/query.md/#partial-response
	ThanosPartial     bool
	ThanosDedup       bool
	ClusterLabelName  string
	ClusterLabelValue string
}

// ClientAuth holds the HTTP client identity info.
type ClientAuth struct {
	Username    string
	BearerToken string
	Password    string
}

// Apply applies the authentication identity info to the HTTP request headers
func (auth *ClientAuth) Apply(req *http.Request) {
	if auth == nil {
		return
	}

	if auth.BearerToken != "" {
		token := "Bearer " + auth.BearerToken
		req.Header.Add("Authorization", token)
	}

	if auth.Username != "" {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
}

// MockConfig represents the config of an in-memory provider, which is for demonstration or testing purpose.
type MockConfig struct {
	SeedFile string
}

type GrpcConfig struct {
	Address string
	Timeout time.Duration
}

type DataSourceType string

const (
	MockDataSource         DataSourceType = "mock"
	PrometheusDataSource   DataSourceType = "prom"
	MetricServerDataSource DataSourceType = "metricserver"
	GrpcDataSource         DataSourceType = "grpc"
)
