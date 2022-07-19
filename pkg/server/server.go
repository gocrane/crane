package server

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/server/middleware"
	clustersrv "github.com/gocrane/crane/pkg/server/service/cluster"
	dashboardsrv "github.com/gocrane/crane/pkg/server/service/dashboard"
	"github.com/gocrane/crane/pkg/server/store"
	"github.com/gocrane/crane/pkg/server/store/secret"
	"github.com/gocrane/crane/pkg/version"
)

type apiServer struct {
	// wrapper for gin.Engine
	*gin.Engine

	config *config.Config

	insecureServer *http.Server

	stopCh chan struct{}

	// srv
	dashboardSrv dashboardsrv.Service
	clusterSrv   clustersrv.Service
}

func NewServer(cfg *config.Config) (*apiServer, error) {
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
	s.GET("/api/healthz", func(c *gin.Context) {
		ginwrapper.WriteResponse(c, nil, map[string]string{"status": "ok"})
	})
	// install version handler
	s.GET("/api/version", func(c *gin.Context) {
		ginwrapper.WriteResponse(c, nil, version.GetVersionInfo())
	})

	s.GET("/api/klog", func(c *gin.Context) {
		v := c.Query("v")
		if v != "" {
			err := flag.Lookup("v").Value.Set(v)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}
			klog.Infof("set log level to %v", v)
		}
		ginwrapper.WriteResponse(c, nil, map[string]string{"status": "ok"})
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
		dashboardSrv, err := dashboardsrv.NewService(s.config.GrafanaConfig)
		if err != nil {
			klog.Fatal(err)
		}
		s.dashboardSrv = dashboardSrv
	}

	// Kubernetes API setup

	if s.config.KubeConfig == nil {
		klog.Fatal(fmt.Errorf("kubernetes rest config is null"))
	}
	kubeClientset, err := kubernetes.NewForConfig(s.config.KubeConfig)
	if err != nil {
		klog.Fatal(err.Error())
	}

	var serverStore store.Store
	if s.config.StoreType == secret.StoreType {
		serverStore, err = store.InitStore(secret.StoreType, &secret.StoreConfig{
			Client: kubeClientset,
		})
		if err != nil {
			klog.Fatal(err)
		}
	}

	clusterSrv := clustersrv.NewService(serverStore)
	s.clusterSrv = clusterSrv
}

// Run spawns the http server. It blocks until the server shut down or error.
func (s *apiServer) Run(ctx context.Context) {
	s.initServices()

	s.installDefaultMiddlewares()
	s.installGenericAPIs()
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
