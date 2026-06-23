package middlewares

import (
	"net/http"
	"strings"

	"bcb/backend/internal/httpserver/response"
	"bcb/backend/internal/modules/access"
	"github.com/gin-gonic/gin"
)

func Authenticate(tokens *access.TokenService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		value, found := strings.CutPrefix(header, "Bearer ")
		if !found || value == "" {
			response.Error(ctx, http.StatusUnauthorized, "invalid_session", "Sessão inválida ou expirada.", nil)
			ctx.Abort()
			return
		}
		claims, err := tokens.Parse(value)
		if err != nil {
			response.Error(ctx, http.StatusUnauthorized, "invalid_session", "Sessão inválida ou expirada.", nil)
			ctx.Abort()
			return
		}
		access.SetClaims(ctx, claims)
		ctx.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, ok := access.ClaimsFromContext(ctx)
		if !ok || claims.Role != role {
			response.Error(ctx, http.StatusForbidden, "forbidden", "Você não possui permissão para esta operação.", nil)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
