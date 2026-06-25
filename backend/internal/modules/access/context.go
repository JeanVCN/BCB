package access

import "github.com/gin-gonic/gin"

const claimsContextKey = "authClaims"

// SetClaims stores authenticated claims in the Gin context for downstream handlers.
func SetClaims(ctx *gin.Context, claims Claims) {
	ctx.Set(claimsContextKey, claims)
}

// ClaimsFromContext reads authenticated claims previously stored in the Gin context.
func ClaimsFromContext(ctx *gin.Context) (Claims, bool) {
	value, found := ctx.Get(claimsContextKey)
	claims, ok := value.(Claims)
	return claims, found && ok
}
