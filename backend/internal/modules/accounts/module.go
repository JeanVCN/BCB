package accounts

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

func New(dependencies Dependencies) *Module {
	repository := newRepository(dependencies.Postgres)
	service := newService(repository)

	return &Module{
		handler: newHandler(service, repository),
	}
}

func (module *Module) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.POST("/auth/register", module.handler.register)
	admin.GET("/clients", module.handler.listClients)
	admin.POST("/clients/:clientId/activate", module.handler.activate)
	admin.POST("/clients/:clientId/reject", module.handler.reject)
	admin.POST("/clients/:clientId/deactivate", module.handler.deactivate)
}
