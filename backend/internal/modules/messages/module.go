package messages

import (
	"log/slog"

	"bcb/backend/internal/modules/billing"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
	Logger   *slog.Logger
}

type Module struct {
	handler *Handler
	worker  *Worker
}

func New(dependencies Dependencies) *Module {
	repository := NewRepository(dependencies.Postgres)
	locks := billing.NewLockManager(dependencies.Redis)
	service := NewService(repository, locks)

	return &Module{
		handler: NewHandler(service),
		worker:  NewWorker(repository, dependencies.Logger),
	}
}

func (module *Module) RegisterRoutes(client *gin.RouterGroup) {
	client.GET("/conversations/:conversationId/messages", module.handler.List)
	client.POST("/conversations/:conversationId/messages", module.handler.Send)
}

func (module *Module) Worker() *Worker {
	return module.worker
}
