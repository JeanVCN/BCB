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
	repository := NewRepository(dependencies.Postgres)
	service := NewService(repository)

	return &Module{handler: NewHandler(service)}
}

func (module *Module) RegisterRoutes(client *gin.RouterGroup) {
	client.POST("/conversations", module.handler.Create)
	client.GET("/conversations", module.handler.List)
}
