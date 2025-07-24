package prom

import (
	gocontext "context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	prometheus "github.com/prometheus/client_golang/api"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/providers"
)

// NewPrometheusClient returns a prometheus.Client
func NewPrometheusClient(config *providers.PromConfig) (prometheus.Client, error) {

	tlsConfig := &tls.Config{InsecureSkipVerify: config.InsecureSkipVerify}

	pc := prometheus.Config{
		Address: config.Address,
		RoundTripper: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   config.Timeout,
				KeepAlive: config.KeepAlive,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
		},
	}
	if config.BRateLimit {
		return newPrometheusRateLimitClient(PrometheusClientID, pc, &config.Auth, config.QueryConcurrency)
	}
	return newPrometheusAuthClient(PrometheusClientID, pc, &config.Auth)

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

type prometheusRateLimitClient struct {
	id     string
	auth   *providers.ClientAuth
	client prometheus.Client

	lock            *sync.Mutex
	cond            *sync.Cond
	maxInFlight     int
	currentInFlight int
}

func newPrometheusRateLimitClient(id string, config prometheus.Config, auth *providers.ClientAuth, maxInFlight int) (prometheus.Client, error) {
	c, err := prometheus.NewClient(config)
	if err != nil {
		return nil, err
	}

	lock := &sync.Mutex{}
	client := &prometheusRateLimitClient{
		id:          id,
		client:      c,
		auth:        auth,
		maxInFlight: maxInFlight,
		lock:        lock,
		cond:        sync.NewCond(lock),
	}

	return client, nil
}

// URL implements prometheus client interface
func (pc *prometheusRateLimitClient) URL(ep string, args map[string]string) *url.URL {
	return pc.client.URL(ep, args)
}

// Do implements prometheus client interface, wrapped with an auth info
func (pc *prometheusRateLimitClient) Do(ctx gocontext.Context, req *http.Request) (*http.Response, []byte, error) {
	pc.auth.Apply(req)
	klog.V(4).InfoS("Prometheus rate limit", "ratelimit", pc.Runtime())
	// block wait until at least one InFlight request finished if current inflighting requests reach the max limit, avoid many time consuming requests hit the prometheus.
	// we use inflight to record the number of inflighting requests, because prometheus query is time-consuming when the range is large
	// Caller will be blocked by lock if it reached maxFlight. so caller should set timeout for context and cancel it.
	pc.increaseInFightWait()
	defer pc.decreaseInFlightSignal()
	return pc.client.Do(ctx, req)
}

func (mfc *prometheusRateLimitClient) increaseInFightWait() {
	mfc.lock.Lock()
	mfc.currentInFlight++
	if mfc.currentInFlight > mfc.maxInFlight {
		mfc.cond.Wait()
	}
	mfc.lock.Unlock()
}

func (mfc *prometheusRateLimitClient) decreaseInFlightSignal() {
	mfc.lock.Lock()
	defer mfc.lock.Unlock()
	mfc.currentInFlight--
	mfc.cond.Signal()
}

func (mfc *prometheusRateLimitClient) Runtime() FlowControlRuntime {
	mfc.lock.Lock()
	defer mfc.lock.Unlock()
	return FlowControlRuntime{
		InFight:      mfc.currentInFlight,
		InFightLimit: mfc.maxInFlight,
	}
}

type FlowControlRuntime struct {
	// Current InfFight limit.
	InFightLimit int
	// Current InfFight requests.
	InFight int
}

func (fcr *FlowControlRuntime) String() string {
	return fmt.Sprintf("InFightLimit: %v, InFlight: %v", fcr.InFightLimit, fcr.InFight)
}
