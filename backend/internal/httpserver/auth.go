package httpserver

import (
	"net/http"
	"strings"

	"bcb/backend/internal/identity"
	"github.com/gin-gonic/gin"
)

const claimsContextKey = "authClaims"

func authenticate(tokens *identity.TokenService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		value, found := strings.CutPrefix(header, "Bearer ")
		if !found || value == "" {
			writeError(ctx, http.StatusUnauthorized, "invalid_session", "Sessão inválida ou expirada.", nil)
			ctx.Abort()
			return
		}
		claims, err := tokens.Parse(value)
		if err != nil {
			writeError(ctx, http.StatusUnauthorized, "invalid_session", "Sessão inválida ou expirada.", nil)
			ctx.Abort()
			return
		}
		ctx.Set(claimsContextKey, claims)
		ctx.Next()
	}
}

func requireRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, ok := currentClaims(ctx)
		if !ok || claims.Role != role {
			writeError(ctx, http.StatusForbidden, "forbidden", "Você não possui permissão para esta operação.", nil)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func currentClaims(ctx *gin.Context) (identity.Claims, bool) {
	value, found := ctx.Get(claimsContextKey)
	claims, ok := value.(identity.Claims)
	return claims, found && ok
}
