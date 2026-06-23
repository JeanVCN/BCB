package httpserver

import (
	"context"
	"net/http"

	"bcb/backend/internal/domain"
	"bcb/backend/internal/httpserver/middlewares"
	"bcb/backend/internal/modules"
	"github.com/gin-gonic/gin"
)

type ReadinessChecker interface {
	Ready(context.Context) error
}

type Dependencies struct {
	Modules   *modules.Registry
	Readiness ReadinessChecker
}

func NewRouter(dependencies Dependencies) http.Handler {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health/live", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/health/ready", func(ctx *gin.Context) {
		if dependencies.Readiness != nil {
			if err := dependencies.Readiness.Ready(ctx); err != nil {
				ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
				return
			}
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	if dependencies.Modules == nil {
		return router
	}

	api := router.Group("/api/v1")

	authenticated := api.Group("")
	authenticated.Use(middlewares.Authenticate(dependencies.Modules.TokenService()))

	admin := authenticated.Group("/admin")
	admin.Use(middlewares.RequireRole(domain.RoleAdmin))

	client := authenticated.Group("")
	client.Use(middlewares.RequireRole(domain.RoleClient))

	dependencies.Modules.RegisterRoutes(modules.Routes{
		Public:        api,
		Authenticated: authenticated,
		Admin:         admin,
		Client:        client,
	})

	return router
}
