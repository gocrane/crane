package prom

import (
	gocontext "context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gocrane/crane/pkg/providers"

	prometheus "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"

	"github.com/gocrane/crane/pkg/common"
)

const (
	PrometheusClientID = "prom"
)

type context struct {
	api  promapiv1.API
	name string
}

// NewContext creates a new Prometheus querying context from the given client.
func NewContext(client prometheus.Client) *context {
	return &context{
		api:  promapiv1.NewAPI(client),
		name: "",
	}
}

// NewNamedContext creates a new named Prometheus querying context from the given client
func NewNamedContext(client prometheus.Client, name string) *context {
	ctx := NewContext(client)
	ctx.name = name
	return ctx
}

// QueryRangeSync range query prometheus in sync way
func (c *context) QueryRangeSync(ctx gocontext.Context, query string, start, end time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	var ts []*common.TimeSeries
	r := promapiv1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
	results, warnings, err := c.api.QueryRange(ctx, query, r)
	if len(warnings) != 0 {
		logger.Info("prom query range warnings", "warnings", warnings)
	}
	if err != nil {
		return ts, err
	}
	logger.V(10).Info("prom query range result", "result", results.String(), "resultsType", results.Type())
	return c.convertPromResultsToTimeSeries(results)
}

// QuerySync query prometheus in sync way
func (c *context) QuerySync(ctx gocontext.Context, query string) ([]*common.TimeSeries, error) {
	var ts []*common.TimeSeries
	results, warnings, err := c.api.Query(ctx, query, time.Now())
	if len(warnings) != 0 {
		logger.Info("prom query warnings", "warnings", warnings)
	}
	if err != nil {
		return ts, err
	}
	logger.V(10).Info("prom query result", "result", results.String(), "resultsType", results.Type())
	return c.convertPromResultsToTimeSeries(results)

}

func (c *context) convertPromResultsToTimeSeries(value prommodel.Value) ([]*common.TimeSeries, error) {
	var results []*common.TimeSeries
	typeValue := value.Type()
	switch typeValue {
	case prommodel.ValMatrix:
		if matrix, ok := value.(prommodel.Matrix); ok {
			for _, sampleStream := range matrix {
				if sampleStream == nil {
					continue
				}
				ts := common.NewTimeSeries()
				for key, val := range sampleStream.Metric {
					ts.AppendLabel(string(key), string(val))
				}
				for _, pair := range sampleStream.Values {
					ts.AppendSample(int64(pair.Timestamp/1000), float64(pair.Value))
				}
				results = append(results, ts)
			}
			return results, nil
		} else {
			return results, fmt.Errorf("prometheus value type is %v, but assert failed", typeValue)
		}

	case prommodel.ValVector:
		if vector, ok := value.(prommodel.Vector); ok {
			for _, sample := range vector {
				if sample == nil {
					continue
				}
				ts := common.NewTimeSeries()
				for key, val := range sample.Metric {
					ts.AppendLabel(string(key), string(val))
				}
				// for vector, all the sample has the same timestamp. just one point for each metric
				ts.AppendSample(int64(sample.Timestamp/1000), float64(sample.Value))
				results = append(results, ts)
			}
			return results, nil
		} else {
			return results, fmt.Errorf("prometheus value type is %v, but assert failed", typeValue)
		}
	case prommodel.ValScalar:
		return results, fmt.Errorf("not support for scalar when use timeseries")
	case prommodel.ValString:
		return results, fmt.Errorf("not support for string when use timeseries")
	case prommodel.ValNone:
		return results, fmt.Errorf("prometheus return value type is none")
	}
	return results, fmt.Errorf("prometheus return unknown model value type %v", typeValue)
}

// NewPrometheusClient returns a prometheus.Client
func NewPrometheusClient(address string, timeout, keepAlive time.Duration, queryConcurrency int, insecureSkipVerify bool,
	needRateLimit bool, auth providers.ClientAuth) (prometheus.Client, error) {

	tlsConfig := &tls.Config{InsecureSkipVerify: insecureSkipVerify}

	pc := prometheus.Config{
		Address: address,
		RoundTripper: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: keepAlive,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
		},
	}

	return newPrometheusAuthClient(PrometheusClientID, pc, &auth)

}

// prometheusAuthClient wraps the prometheus api raw client with authentication info
type prometheusAuthClient struct {
	id     string
	auth   *providers.ClientAuth
	client prometheus.Client
}

func newPrometheusAuthClient(id string, config prometheus.Config, auth *providers.ClientAuth) (prometheus.Client, error) {
	c, err := prometheus.NewClient(config)
	if err != nil {
		return nil, err
	}

	client := &prometheusAuthClient{
		id:     id,
		client: c,
		auth:   auth,
	}

	return client, nil
}

// URL implements prometheus client interface
func (pc *prometheusAuthClient) URL(ep string, args map[string]string) *url.URL {
	return pc.client.URL(ep, args)
}

// Do implements prometheus client interface, wrapped with an auth info
func (pc *prometheusAuthClient) Do(ctx gocontext.Context, req *http.Request) (*http.Response, []byte, error) {
	pc.auth.Apply(req)
	return pc.client.Do(ctx, req)
}
