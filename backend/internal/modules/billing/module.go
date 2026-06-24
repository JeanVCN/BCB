package billing

import (
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
	repository := NewRepository(dependencies.Postgres)
	locks := NewLockManager(dependencies.Redis)
	service := NewService(repository, locks)

	return &Module{handler: newHandler(service)}
}

func (module *Module) RegisterRoutes(admin, client *gin.RouterGroup) {
	client.GET("/billing", module.handler.profile)
	client.GET("/billing/transactions", module.handler.clientTransactions)

	admin.POST("/admin/clients/:clientId/credits", module.handler.addCredit)
	admin.PUT("/admin/clients/:clientId/postpaid-limit", module.handler.adjustPostpaidLimit)
	admin.GET("/admin/clients/:clientId/financial-transactions", module.handler.adminTransactions)
}
