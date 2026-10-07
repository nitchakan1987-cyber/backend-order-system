package router

import (
	"github.com/gin-gonic/gin"

	"orders_backend/api"
	"orders_backend/health"
)

type Dependencies struct {
	Health *health.Handler
	API    *api.Handler
}

func New(deps Dependencies) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/health", gin.WrapF(deps.Health.Check))
	deps.API.Register(r.Group("/api/v1"))

	return r
}
