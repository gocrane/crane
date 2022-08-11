package prom

import (
	gocontext "context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	prometheus "github.com/prometheus/client_golang/api"
	"k8s.io/klog/v2"

	datasource "github.com/gocrane/crane/pkg/providers"
)

type ThanosConfig struct {
	Partial bool
	Dedup   bool
}

// NewPrometheusClient returns a prometheus.Client
func NewPrometheusClient(config *datasource.PromConfig) (prometheus.Client, error) {

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
		return newPrometheusRateLimitClient(PrometheusClientID, pc, &config.Auth, config.QueryConcurrency, &ThanosConfig{Dedup: config.ThanosDedup, Partial: config.ThanosPartial})
	}
	return newPrometheusAuthClient(PrometheusClientID, pc, &config.Auth, &ThanosConfig{Dedup: config.ThanosDedup, Partial: config.ThanosPartial})

}

// prometheusAuthClient wraps the prometheus api raw client with authentication info
type prometheusAuthClient struct {
	id     string
	auth   *datasource.ClientAuth
	client prometheus.Client
	// hack here, later maybe a datasource for thanos
	thanos *ThanosConfig
}

func newPrometheusAuthClient(id string, config prometheus.Config, auth *datasource.ClientAuth, thanos *ThanosConfig) (prometheus.Client, error) {
	c, err := prometheus.NewClient(config)
	if err != nil {
		return nil, err
	}

	client := &prometheusAuthClient{
		thanos: thanos,
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
	newReq := req
	// hacking for thanos here, intercept the request and modify param
	if pc.thanos != nil && (pc.thanos.Dedup || pc.thanos.Partial) {
		var q url.Values
		var err error
		var bodyData []byte
		if req.Method == http.MethodPost {
			bodyReader, err := req.GetBody()
			if err != nil {
				return nil, nil, err
			}
			bodyData, err = ioutil.ReadAll(bodyReader)
			if err != nil {
				return nil, nil, err
			}
			q, err = url.ParseQuery(string(bodyData))
			if err != nil {
				return nil, nil, err
			}
		} else if req.Method == http.MethodGet {
			q = req.URL.Query()
		}

		if pc.thanos.Partial {
			q.Set("partial_response", "true")
		}
		if pc.thanos.Dedup {
			q.Set("dedup", "true")
		}

		klog.V(6).InfoS("Hacking thanos", "originalQueryBody", string(bodyData), "newQuery", q.Encode())

		if req.Method == http.MethodGet {
			req.URL.RawQuery = q.Encode()
		} else if req.Method == http.MethodPost {
			newReq, err = http.NewRequest(req.Method, req.URL.String(), strings.NewReader(q.Encode()))
			if err != nil {
				return nil, nil, err
			}
			newReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	pc.auth.Apply(newReq)
	klog.V(6).InfoS("Query url", "url", newReq.URL, "newReq", newReq, "oldReq", req)
	return pc.client.Do(ctx, newReq)
}

type prometheusRateLimitClient struct {
	id     string
	auth   *datasource.ClientAuth
	client prometheus.Client
	// hack here, later maybe a datasource for thanos
	thanos *ThanosConfig

	lock            *sync.Mutex
	cond            *sync.Cond
	maxInFlight     int
	currentInFlight int
}

func newPrometheusRateLimitClient(id string, config prometheus.Config, auth *datasource.ClientAuth, maxInFlight int, thanos *ThanosConfig) (prometheus.Client, error) {
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
		thanos:      thanos,
	}

	return client, nil
}

// URL implements prometheus client interface
func (pc *prometheusRateLimitClient) URL(ep string, args map[string]string) *url.URL {
	return pc.client.URL(ep, args)
}

// Do implements prometheus client interface, wrapped with an auth info
func (pc *prometheusRateLimitClient) Do(ctx gocontext.Context, req *http.Request) (*http.Response, []byte, error) {
	newReq := req
	// hacking for thanos here, intercept the request and modify param
	if pc.thanos != nil && (pc.thanos.Dedup || pc.thanos.Partial) {
		var q url.Values
		var err error
		var bodyData []byte
		if req.Method == http.MethodPost {
			bodyReader, err := req.GetBody()
			if err != nil {
				return nil, nil, err
			}
			bodyData, err = ioutil.ReadAll(bodyReader)
			if err != nil {
				return nil, nil, err
			}
			q, err = url.ParseQuery(string(bodyData))
			if err != nil {
				return nil, nil, err
			}
		} else if req.Method == http.MethodGet {
			q = req.URL.Query()
		}

		if pc.thanos.Partial {
			q.Set("partial_response", "true")
		}
		if pc.thanos.Dedup {
			q.Set("dedup", "true")
		}

		klog.V(6).InfoS("Hacking thanos", "originalQueryBody", string(bodyData), "newQuery", q.Encode())

		if req.Method == http.MethodGet {
			req.URL.RawQuery = q.Encode()
		} else if req.Method == http.MethodPost {
			newReq, err = http.NewRequest(req.Method, req.URL.String(), strings.NewReader(q.Encode()))
			if err != nil {
				return nil, nil, err
			}
			newReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	pc.auth.Apply(newReq)
	klog.V(6).InfoS("Query url", "url", newReq.URL, "newReq", newReq, "oldReq", req)
	klog.V(4).InfoS("Prometheus rate limit", "ratelimit", pc.Runtime())
	// block wait until at least one InFlight request finished if current inflighting requests reach the max limit, avoid many time consuming requests hit the prometheus.
	// we use inflight to record the number of inflighting requests, because prometheus query is time-consuming when the range is large
	// Caller will be blocked by lock if it reached maxFlight. so caller should set timeout for context and cancel it.
	pc.increaseInFightWait()
	defer pc.decreaseInFlightSignal()
	return pc.client.Do(ctx, newReq)
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
