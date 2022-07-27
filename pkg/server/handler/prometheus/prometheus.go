package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
)

type Handler struct {
	promApi promapiv1.API
}

func NewPrometheusAPIHandler(config *config.Config) *Handler {
	return &Handler{
		promApi: config.Api,
	}
}

// Query delicate prometheus query api.
func (h *Handler) Query(c *gin.Context) {
	ts, err := time.Parse("2006-01-02 15:04:05", c.Query("time"))
	if err != nil {
		ginwrapper.WriteResponse(c, fmt.Errorf("parse time failed: %v", err), nil)
		return
	}

	value, warnings, err := h.promApi.Query(context.TODO(), c.Query("query"), ts)
	if len(warnings) != 0 {
		klog.InfoS("Prom query warnings", "warnings", warnings)
	}
	if err != nil {
		ginwrapper.WriteResponse(c, fmt.Errorf("prom query failed: %v", err), nil)
		return
	}

	ginwrapper.WriteResponse(c, nil, value)
}

// RangeQuery delicate prometheus range query api.
func (h *Handler) RangeQuery(c *gin.Context) {
	tsStart, err := time.Parse("2006-01-02 15:04:05", c.Query("start"))
	if err != nil {
		ginwrapper.WriteResponse(c, fmt.Errorf("parse start failed: %v", err), nil)
		return
	}

	tsEnd, err := time.Parse("2006-01-02 15:04:05", c.Query("end"))
	if err != nil {
		ginwrapper.WriteResponse(c, fmt.Errorf("parse end failed: %v", err), nil)
		return
	}

	step, err := time.ParseDuration(c.Query("step"))
	if err != nil {
		ginwrapper.WriteResponse(c, fmt.Errorf("parse step failed: %v", err), nil)
		return
	}

	queryRange := promapiv1.Range{}
	queryRange.Start = tsStart
	queryRange.End = tsEnd
	queryRange.Step = step

	value, warnings, err := h.promApi.QueryRange(context.TODO(), c.Query("query"), queryRange)
	if len(warnings) != 0 {
		klog.InfoS("Prom query range warnings", "warnings", warnings)
	}
	if err != nil {
		ginwrapper.WriteResponse(c, fmt.Errorf("prom query range failed: %v", err), nil)
		return
	}

	ginwrapper.WriteResponse(c, nil, value)
}
