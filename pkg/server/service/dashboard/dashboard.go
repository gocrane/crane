package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	grafanasdk "github.com/grafana-tools/sdk"
	"k8s.io/klog/v2"
)

type GrafanaConfig struct {
	Address
	Auth
}

type Address struct {
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

// DashboardSrv is not used now. front end can get the grafana embedding directly
type Service interface {
	Dashboards(ctx context.Context) ([]grafanasdk.FoundBoard, error)
	PanelEmbeddings(ctx context.Context, useRange bool, from time.Time, to time.Time) (*PanelList, error)
}

type dashboardService struct {
	config        *GrafanaConfig
	grafanaClient *grafanasdk.Client
}

func NewService(config *GrafanaConfig) (*dashboardService, error) {
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
	return &dashboardService{
		config:        config,
		grafanaClient: gc,
	}, err
}

func (s *dashboardService) Dashboards(ctx context.Context) ([]grafanasdk.FoundBoard, error) {
	r, err := s.grafanaClient.Search(ctx)
	if err != nil {
		klog.Error(err)
	}
	return r, err
}

func (s *dashboardService) DashboardPanels(ctx context.Context, uid string) ([]*grafanasdk.Panel, error) {
	board, _, err := s.grafanaClient.GetDashboardByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return board.Panels, nil
}

func (s *dashboardService) RawDashboard(ctx context.Context, uid string) (map[string]interface{}, error) {
	boardBytes, _, err := s.grafanaClient.GetRawDashboardByUID(ctx, uid)
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
	EmbeddingLink string `json:"embeddingLink,omitempty"`
	Title         string `json:"title,omitempty"`
}

func (s *dashboardService) PanelEmbeddings(ctx context.Context, useRange bool, from time.Time, to time.Time) (*PanelList, error) {
	foundBoards, err := s.Dashboards(ctx)
	if err != nil {
		return nil, err
	}
	var links []*DashboardPanel
	var count int32
	for _, foundBoard := range foundBoards {
		panels, err := s.DashboardPanels(ctx, foundBoard.UID)
		if err != nil {
			klog.Warningf("Failed to get dashboard (%v, %v) panels, err: %v", foundBoard.Title, foundBoard.UID, err)
		}
		for _, panel := range panels {
			embeddingLink := BuildEmbeddingLink(s.config.Scheme, s.config.Host, foundBoard.URL, useRange, from.UnixNano(), to.UnixNano(), panel.ID)
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
