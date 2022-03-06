package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getFakeResponse(start time.Time, activityTimeout time.Duration, checkMonitoring bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/health-check", nil)
	w := httptest.NewRecorder()
	healthCheck := NewHealthCheck(activityTimeout)
	if checkMonitoring {
		healthCheck.StartMonitoring()
	}
	healthCheck.lastActivity = start
	healthCheck.lastConfigUpdated = start
	healthCheck.ServeHTTP(w, req)
	return w
}

func TestServeHTTPStatusOK(t *testing.T) {
	w := getFakeResponse(time.Now(), time.Second*2, true)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServeHTTPTimeout(t *testing.T) {
	w := getFakeResponse(time.Now().Add(time.Second*-2), time.Second, true)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServeHTTPNoCheck(t *testing.T) {
	w := getFakeResponse(time.Now().Add(time.Second*-2), time.Second, false)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServeHTTPNoTimeout(t *testing.T) {
	w := getFakeResponse(time.Now().Add(time.Second*2), time.Second, true)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLastActivity(t *testing.T) {
	timeout := time.Second

	req := httptest.NewRequest("GET", "/health-check", nil)
	healthCheck := NewHealthCheck(timeout)
	healthCheck.StartMonitoring()

	cases := map[string]struct {
		LastActivity     time.Duration
		LastConfigUpdate time.Duration
		Code             int
	}{
		"LastActivity Failed": {
			LastActivity:     timeout * -2,
			LastConfigUpdate: timeout * 5,
			Code:             http.StatusInternalServerError,
		},
		"LastConfigUpdate Failed": {
			LastActivity:     timeout * 5,
			LastConfigUpdate: timeout * -2,
			Code:             http.StatusInternalServerError,
		},
		"All Failed": {
			LastActivity:     timeout * -2,
			LastConfigUpdate: timeout * -2,
			Code:             http.StatusInternalServerError,
		},
		"All Succeed": {
			LastActivity:     timeout * 5,
			LastConfigUpdate: timeout * 5,
			Code:             http.StatusOK,
		},
	}

	for key, c := range cases {
		w := httptest.NewRecorder()
		healthCheck.lastActivity = time.Now().Add(c.LastActivity)
		healthCheck.lastConfigUpdated = time.Now().Add(c.LastConfigUpdate)

		healthCheck.ServeHTTP(w, req)
		assert.Equal(t, c.Code, w.Code, key)
	}
}
