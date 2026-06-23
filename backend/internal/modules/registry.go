package modules

import (
	"bcb/backend/internal/config"
	"bcb/backend/internal/modules/access"
	"bcb/backend/internal/modules/accounts"

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
	access   *access.Module
	accounts *accounts.Module
}

type Routes struct {
	Public        *gin.RouterGroup
	Authenticated *gin.RouterGroup
	Admin         *gin.RouterGroup
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

	return &Registry{
		access:   accessModule,
		accounts: accountsModule,
	}
}

func (registry *Registry) TokenService() *access.TokenService {
	return registry.access.TokenService()
}

func (registry *Registry) RegisterRoutes(routes Routes) {
	registry.access.RegisterRoutes(routes.Public, routes.Authenticated)
	registry.accounts.RegisterRoutes(routes.Public, routes.Admin)
}
