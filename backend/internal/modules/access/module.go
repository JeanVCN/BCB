package access

import (
	"bcb/backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Config   config.Config
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

type Module struct {
	tokens  *TokenService
	handler *Handler
}

func New(dependencies Dependencies) *Module {
	repository := NewRepository(dependencies.Postgres)
	tokens := newTokenService(dependencies.Config.JWTSecret)
	service := newService(repository, tokens, newRateLimiter(dependencies.Redis))

	return &Module{
		tokens:  tokens,
		handler: newHandler(service),
	}
}

func (module *Module) TokenService() *TokenService {
	return module.tokens
}

func (module *Module) RegisterRoutes(public, authenticated *gin.RouterGroup) {
	public.POST("/auth/login", module.handler.login)
	authenticated.GET("/me", module.handler.me)
}
