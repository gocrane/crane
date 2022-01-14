package dashboards

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gocrane/crane/pkg/server/code"
	"github.com/gocrane/crane/pkg/server/errors"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/server/service/dashboard"
)

type ListPanelsRequest struct {
	UseRange bool      `json:"useRange"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
}

type DashboardHandler struct {
	manager dashboard.DashboardSrv
}

func NewDashboardHandler(manager dashboard.DashboardSrv) *DashboardHandler {
	return &DashboardHandler{
		manager: manager,
	}
}

// List list the dashboard in the grafana.
func (d *DashboardHandler) List(c *gin.Context) {
	var r ListPanelsRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		ginwrapper.WriteResponse(c, errors.WithCode(code.ErrBind, err.Error()), nil)
		return
	}

	panels, err := d.manager.Dashboards(context.TODO())
	if err != nil {
		ginwrapper.WriteResponse(c, errors.WrapC(err, code.ErrDashboardNotFound, err.Error()), nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, panels)
}
