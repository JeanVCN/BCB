package httpserver

import (
	"context"
	"net/http"

	"bcb/backend/internal/identity"
	"github.com/gin-gonic/gin"
)

type ReadinessChecker interface {
	Ready(context.Context) error
}

type Dependencies struct {
	Identity  *IdentityHandler
	Tokens    *identity.TokenService
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

	if dependencies.Identity == nil || dependencies.Tokens == nil {
		return router
	}

	api := router.Group("/api/v1")
	api.POST("/auth/register", dependencies.Identity.Register)
	api.POST("/auth/login", dependencies.Identity.Login)

	authenticated := api.Group("")
	authenticated.Use(authenticate(dependencies.Tokens))
	authenticated.GET("/me", dependencies.Identity.Me)

	admin := authenticated.Group("/admin")
	admin.Use(requireRole("admin"))
	admin.GET("/clients", dependencies.Identity.ListClients)
	admin.POST("/clients/:clientId/activate", dependencies.Identity.Activate)
	admin.POST("/clients/:clientId/reject", dependencies.Identity.Reject)
	admin.POST("/clients/:clientId/deactivate", dependencies.Identity.Deactivate)

	return router
}
