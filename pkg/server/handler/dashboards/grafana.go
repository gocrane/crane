package dashboards

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/server/service/dashboard"
)

type ListPanelsRequest struct {
	UseRange bool      `json:"useRange"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
}

type DashboardHandler struct {
	service dashboard.Service
}

func NewDashboardHandler(service dashboard.Service) *DashboardHandler {
	return &DashboardHandler{
		service: service,
	}
}

// List list the dashboard in the grafana.
func (d *DashboardHandler) List(c *gin.Context) {
	dbs, err := d.service.Dashboards(context.TODO())
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, dbs)
}

func (d *DashboardHandler) ListPanels(c *gin.Context) {
	var r ListPanelsRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	panels, err := d.service.PanelEmbeddings(context.TODO(), r.UseRange, r.From, r.To)
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, panels)

}
