package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// HealthCheck contains information about last time of crane-agent collect activity and timeout
type HealthCheck struct {
	mutex             *sync.Mutex
	lastActivity      time.Time
	lastConfigUpdated time.Time
	activityTimeout   time.Duration
	checkTimeout      bool
}

// NewHealthCheck builds new HealthCheck object with given timeout
func NewHealthCheck(activityTimeout time.Duration) *HealthCheck {
	now := time.Now()

	return &HealthCheck{
		lastActivity:      now,
		lastConfigUpdated: now,
		mutex:             &sync.Mutex{},
		activityTimeout:   activityTimeout,
		checkTimeout:      false,
	}
}

// StartMonitoring activates checks for crane-agent inactivity
func (hc *HealthCheck) StartMonitoring() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.checkTimeout = true
	now := time.Now()

	if now.After(hc.lastActivity) {
		hc.lastActivity = now
	}

	if now.After(hc.lastConfigUpdated) {
		hc.lastConfigUpdated = now
	}
}

// ServeHTTP implements http.Handler interface to provide a health-check endpoint
func (hc *HealthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hc.mutex.Lock()
	now := time.Now()
	checkTimeout := hc.checkTimeout
	lastActivity := hc.lastActivity
	lastConfigUpdated := hc.lastConfigUpdated
	hc.mutex.Unlock()

	//If not check timeout, return OK
	if !checkTimeout {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte("OK")); err != nil {
			klog.Errorf("Http write failed: %v", err)
			return
		}

		return
	}

	if now.After(lastActivity.Add(hc.activityTimeout)) {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err := w.Write([]byte(fmt.Sprintf("Error: last activity more %v ago", time.Since(lastActivity).String()))); err != nil {
			klog.Errorf("Http write failed: %v", err)
			return
		}

		return
	}

	if now.After(lastConfigUpdated.Add(hc.activityTimeout)) {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err := w.Write([]byte(fmt.Sprintf("Error: last config update  more %v ago", time.Since(lastConfigUpdated).String()))); err != nil {
			klog.Errorf("Http write failed: %v", err)
			return
		}

		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		klog.Errorf("Http write failed: %v", err)
		return
	}

	return
}

// UpdateLastActivity updates last time of activity
func (hc *HealthCheck) UpdateLastActivity(timestamp time.Time) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if timestamp.After(hc.lastActivity) {
		hc.lastActivity = timestamp
	}
}

// UpdateLastConfigUpdate updates last time of config update
func (hc *HealthCheck) UpdateLastConfigUpdate(timestamp time.Time) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if timestamp.After(hc.lastConfigUpdated) {
		hc.lastConfigUpdated = timestamp
	}
}
