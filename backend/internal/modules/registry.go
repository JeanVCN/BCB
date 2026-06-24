package modules

import (
	"bcb/backend/internal/config"
	"bcb/backend/internal/modules/access"
	"bcb/backend/internal/modules/accounts"
	"bcb/backend/internal/modules/billing"
	"bcb/backend/internal/modules/conversations"
	"bcb/backend/internal/modules/messages"
	"bcb/backend/internal/modules/planchanges"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Config   config.Config
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

type Registry struct {
	access        *access.Module
	accounts      *accounts.Module
	billing       *billing.Module
	conversations *conversations.Module
	messages      *messages.Module
	planChanges   *planchanges.Module
}

type Routes struct {
	Public        *gin.RouterGroup
	Authenticated *gin.RouterGroup
	Admin         *gin.RouterGroup
	Client        *gin.RouterGroup
}

func New(dependencies Dependencies) *Registry {
	accessModule := access.New(access.Dependencies{
		Config:   dependencies.Config,
		Postgres: dependencies.Postgres,
		Redis:    dependencies.Redis,
	})

	accountsModule := accounts.New(accounts.Dependencies{
		Postgres: dependencies.Postgres,
	})

	billingModule := billing.New(billing.Dependencies{
		Postgres: dependencies.Postgres,
		Redis:    dependencies.Redis,
	})

	conversationsModule := conversations.New(conversations.Dependencies{
		Postgres: dependencies.Postgres,
	})

	messagesModule := messages.New(messages.Dependencies{
		Postgres: dependencies.Postgres,
		Redis:    dependencies.Redis,
	})

	planChangesModule := planchanges.New(planchanges.Dependencies{
		Postgres: dependencies.Postgres,
	})

	return &Registry{
		access:        accessModule,
		accounts:      accountsModule,
		billing:       billingModule,
		conversations: conversationsModule,
		messages:      messagesModule,
		planChanges:   planChangesModule,
	}
}

func (registry *Registry) TokenService() *access.TokenService {
	return registry.access.TokenService()
}

func (registry *Registry) RegisterRoutes(routes Routes) {
	registry.access.RegisterRoutes(routes.Public, routes.Authenticated)
	registry.accounts.RegisterRoutes(routes.Public, routes.Admin)
	registry.billing.RegisterRoutes(routes.Admin, routes.Client)
	registry.conversations.RegisterRoutes(routes.Client)
	registry.messages.RegisterRoutes(routes.Client)
	registry.planChanges.RegisterRoutes(routes.Admin, routes.Client)
}
