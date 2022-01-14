package server

import (
	"context"

	"net"
	"net/http"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"

	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/server/middleware"
	clustersrv "github.com/gocrane/crane/pkg/server/service/cluster"
	dashboardsrv "github.com/gocrane/crane/pkg/server/service/dashboard"
	"github.com/gocrane/crane/pkg/server/store/configmap"
	"github.com/gocrane/crane/pkg/version"
)

type apiServer struct {
	// wrapper for gin.Engine
	*gin.Engine

	config *config.Config

	insecureServer *http.Server

	stopCh chan struct{}

	// srv
	dashboardSrv dashboardsrv.DashboardSrv
	clusterSrv   clustersrv.ClusterSrv
}

func NewAPIServer(cfg *config.Config) (*apiServer, error) {

	gin.SetMode(cfg.Mode)
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		klog.Infof("%-6s %-s --> %s (%d handlers)", httpMethod, absolutePath, handlerName, nuHandlers)
	}

	server := &apiServer{
		config: cfg,
		Engine: gin.New(),
	}

	return server, nil
}

func (s *apiServer) installGenericAPIs() {
	// install metric handler
	if s.config.EnableMetrics {
		prometheus := ginprometheus.NewPrometheus("gin")
		prometheus.Use(s.Engine)
	}

	// install pprof handler
	if s.config.EnableProfiling {
		pprof.Register(s.Engine)
	}

	// install healthz handler
	s.GET("/healthz", func(c *gin.Context) {
		ginwrapper.WriteResponse(c, nil, map[string]string{"status": "ok"})
	})
	// install version handler
	s.GET("/version", func(c *gin.Context) {
		ginwrapper.WriteResponse(c, nil, version.GetVersionInfo())
	})
}

func (s *apiServer) installDefaultMiddlewares() {
	for m, mw := range middleware.Middlewares {
		klog.Infof("install crane api server middleware: %s", m)
		s.Use(mw)
	}
}

func (s *apiServer) initServices() {
	if s.config.EnableGrafana {
		dashboardMgr, err := dashboardsrv.NewManager(s.config.GrafanaConfig)
		if err != nil {
			klog.Fatal(err)
		}
		s.dashboardSrv = dashboardMgr
	}

	// Kubernetes API setup
	var err error
	var kc *rest.Config
	if s.config.KubeConfig != "" {
		kc, err = clientcmd.BuildConfigFromFlags("", s.config.KubeConfig)
	} else {
		kc, err = rest.InClusterConfig()
	}

	if err != nil {
		klog.Fatal(err.Error())
	}
	kubeClientset, err := kubernetes.NewForConfig(kc)
	if err != nil {
		klog.Fatal(err.Error())
	}

	k8sStore, err := configmap.NewK8SStoreFactory(kubeClientset)
	if err != nil {
		klog.Fatal(err)
	}
	clusterSrv := clustersrv.NewManager(k8sStore)
	s.clusterSrv = clusterSrv
}

// Run spawns the http server. It blocks until the server shut down or error.
func (s *apiServer) Run(ctx context.Context) {
	s.initServices()

	s.installGenericAPIs()
	s.installDefaultMiddlewares()
	s.initRouter()

	s.startGracefulShutDownManager(ctx)

	go func() {
		s.insecureServer = &http.Server{
			Addr:         net.JoinHostPort(s.config.BindAddress, strconv.Itoa(s.config.BindPort)),
			Handler:      s,
			ReadTimeout:  120 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		klog.Infof("Start to listening on http address: %s", s.insecureServer.Addr)

		if err := s.insecureServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
		klog.Infof("Stop to listening on http address: %s", s.insecureServer.Addr)

	}()

	<-s.stopCh
	klog.Infof("Server on %s stopped", s.insecureServer.Addr)
}

func (s *apiServer) startGracefulShutDownManager(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.Close()
		s.stopCh <- struct{}{}
	}()
}

// Close graceful shutdown the crane server.
func (s *apiServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.insecureServer.Shutdown(ctx); err != nil {
		klog.Warningf("Shutdown insecure server failed: %s", err.Error())
	}
}
