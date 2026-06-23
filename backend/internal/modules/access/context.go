package access

import "github.com/gin-gonic/gin"

const claimsContextKey = "authClaims"

func SetClaims(ctx *gin.Context, claims Claims) {
	ctx.Set(claimsContextKey, claims)
}

func ClaimsFromContext(ctx *gin.Context) (Claims, bool) {
	value, found := ctx.Get(claimsContextKey)
	claims, ok := value.(Claims)
	return claims, found && ok
}
