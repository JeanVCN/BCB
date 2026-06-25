package messages

import (
	"bcb/backend/internal/modules/billing"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

type Module struct {
	handler *Handler
}

// New builds the messages module with billing integration and queue persistence.
func New(dependencies Dependencies) *Module {
	locks := billing.NewLockManager(dependencies.Redis)
	billingService := billing.NewService(billing.NewRepository(dependencies.Postgres), locks)
	repository := NewRepository(dependencies.Postgres, billingService)
	service := newService(repository, locks)

	return &Module{handler: newHandler(service)}
}

// RegisterRoutes mounts client message history and send routes.
func (module *Module) RegisterRoutes(client *gin.RouterGroup) {
	client.GET("/conversations/:conversationId/messages", module.handler.list)
	client.POST("/conversations/:conversationId/messages", module.handler.send)
}
