package conversations

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

	return &Module{handler: newHandler(service)}
}

func (module *Module) RegisterRoutes(client *gin.RouterGroup) {
	client.POST("/conversations", module.handler.create)
	client.GET("/conversations", module.handler.list)
}
