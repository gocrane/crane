package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/klog/v2"

	grafanasdk "github.com/grafana-tools/sdk"
)

type GrafanaConfig struct {
	Service
	Auth
}

type Service struct {
	Scheme string
	Host   string
}
type Auth struct {
	APIKey   string
	Username string
	Password string
}

type PanelList struct {
	TotalCount int32             `json:"totalCount"`
	Items      []*DashboardPanel `json:"items"`
}

type DashboardSrv interface {
	Dashboards(ctx context.Context) ([]grafanasdk.FoundBoard, error)
	PanelEmbeddings(ctx context.Context, useRange bool, from time.Time, to time.Time) (*PanelList, error)
}

type manager struct {
	config        *GrafanaConfig
	grafanaClient *grafanasdk.Client
}

func NewManager(config *GrafanaConfig) (*manager, error) {
	//var gc *grafanaclient.Client
	//var err error
	//if config.APIKey != "" {
	//	gc, err = grafanaclient.New(config.BaseUrl, grafanaclient.Config{
	//		APIKey: config.APIKey,
	//	})
	//} else {
	//	gc, err = grafanaclient.New(config.BaseUrl, grafanaclient.Config{
	//		BasicAuth: url.UserPassword(config.Username, config.Password),
	//	})
	//}
	var gc *grafanasdk.Client
	var err error
	if config.Scheme == "" {
		config.Scheme = "http"
	}

	baseUrl := config.Scheme + "://" + config.Host
	if config.APIKey != "" {
		gc, err = grafanasdk.NewClient(baseUrl, config.APIKey, http.DefaultClient)
	} else {
		gc, err = grafanasdk.NewClient(baseUrl, strings.Join([]string{config.Username, config.Password}, ":"), http.DefaultClient)
	}
	if err != nil {
		return nil, err
	}
	return &manager{
		config:        config,
		grafanaClient: gc,
	}, err
}

func (m *manager) Dashboards(ctx context.Context) ([]grafanasdk.FoundBoard, error) {
	return m.grafanaClient.Search(ctx)
}

func (m *manager) DashboardPanels(ctx context.Context, uid string) ([]*grafanasdk.Panel, error) {
	board, _, err := m.grafanaClient.GetDashboardByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return board.Panels, nil
}

func (m *manager) RawDashboard(ctx context.Context, uid string) (map[string]interface{}, error) {
	boardBytes, _, err := m.grafanaClient.GetRawDashboardByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	var boardMap map[string]interface{}
	err = json.Unmarshal(boardBytes, &boardMap)
	if err != nil {
		return nil, err
	}
	return boardMap, nil
}

type DashboardPanel struct {
	// http://127.0.0.1:3000/d/Pq1y8i07z/costs-by-dimension?orgId=1&from=1642321150064&to=1642407550064&viewPanel=21
	EmbeddingLink string `json:"embeddingLink,omitempty"`
	Title         string `json:"title,omitempty"`
}

func (m *manager) PanelEmbeddings(ctx context.Context, useRange bool, from time.Time, to time.Time) (*PanelList, error) {
	fmt.Println(m.config)
	foundBoards, err := m.Dashboards(ctx)
	if err != nil {
		return nil, err
	}
	var links []*DashboardPanel
	var count int32
	for _, foundBoard := range foundBoards {
		//todo: panels, err: unmarshal board: json: cannot unmarshal object into Go struct field Board.panels of type string
		panels, err := m.DashboardPanels(ctx, foundBoard.UID)
		if err != nil {
			klog.Warningf("Failed to get dashboard (%v, %v) panels, err: %v", foundBoard.Title, foundBoard.UID, err)
		}
		for _, panel := range panels {
			embeddingLink := BuildEmbeddingLink(m.config.Scheme, m.config.Host, foundBoard.URL, useRange, from.UnixNano(), to.UnixNano(), panel.ID)
			links = append(links, &DashboardPanel{
				EmbeddingLink: embeddingLink,
				Title:         panel.Title,
			})
			count++
		}
	}
	return &PanelList{TotalCount: count, Items: links}, nil
}

func BuildEmbeddingLink(scheme, host, dashboardUrl string, useRange bool, from, to int64, panelId uint) string {
	query := url.Values{}

	// org is not set now, default 1.
	if useRange {
		query.Set("from", fmt.Sprintf("%v", from))
		query.Set("to", fmt.Sprintf("%v", to))
	}
	query.Set("viewPanel", fmt.Sprintf("%v", panelId))

	u := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     dashboardUrl,
		RawQuery: query.Encode(),
	}

	return u.String()

}
