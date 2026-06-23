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
	repository := NewRepository(dependencies.Postgres)
	service := NewService(repository)

	return &Module{
		handler: NewHandler(service, repository),
	}
}

func (module *Module) RegisterRoutes(public, admin *gin.RouterGroup) {
	public.POST("/auth/register", module.handler.Register)
	admin.GET("/clients", module.handler.ListClients)
	admin.POST("/clients/:clientId/activate", module.handler.Activate)
	admin.POST("/clients/:clientId/reject", module.handler.Reject)
	admin.POST("/clients/:clientId/deactivate", module.handler.Deactivate)
}
