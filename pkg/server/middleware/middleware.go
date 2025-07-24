package middleware

import (
	"github.com/gin-gonic/gin"
)

// Middlewares store registered middlewares.
var Middlewares = defaultMiddlewares()

func defaultMiddlewares() map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"log":      Logger(),
		"cors":     Cors(),
		"recovery": gin.Recovery(),
	}
}
