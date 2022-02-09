package middleware

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()
		klog.Infof("| %3d | %13v | %15s | %s  %s |", c.Writer.Status(), end.Sub(start), c.ClientIP(), c.Request.Method, c.Request.URL.Path)
	}
}
