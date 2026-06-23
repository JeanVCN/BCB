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

func New(dependencies Dependencies) *Module {
	locks := billing.NewLockManager(dependencies.Redis)
	billingService := billing.NewService(billing.NewRepository(dependencies.Postgres), locks)
	repository := NewRepository(dependencies.Postgres, billingService)
	service := NewService(repository, locks)

	return &Module{handler: NewHandler(service)}
}

func (module *Module) RegisterRoutes(client *gin.RouterGroup) {
	client.GET("/conversations/:conversationId/messages", module.handler.List)
	client.POST("/conversations/:conversationId/messages", module.handler.Send)
}
