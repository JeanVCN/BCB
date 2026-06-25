package planchanges

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Postgres *pgxpool.Pool
}

type Module struct {
	handler *Handler
}

// New builds the plan change module with request workflow dependencies.
func New(dependencies Dependencies) *Module {
	repository := newRepository(dependencies.Postgres)
	service := newService(repository)

	return &Module{handler: newHandler(service)}
}

// RegisterRoutes mounts client and admin plan change workflow routes.
func (module *Module) RegisterRoutes(admin, client *gin.RouterGroup) {
	client.POST("/plan-change-requests", module.handler.create)
	client.GET("/plan-change-requests/current", module.handler.current)
	client.POST("/plan-change-requests/:requestId/cancel", module.handler.cancel)

	admin.GET("/notifications/summary", module.handler.summary)
	admin.GET("/plan-change-requests", module.handler.adminList)
	admin.POST("/plan-change-requests/:requestId/approve", module.handler.approve)
	admin.POST("/plan-change-requests/:requestId/reject", module.handler.reject)
}
