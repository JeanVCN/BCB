package response

import "github.com/gin-gonic/gin"

func Error(ctx *gin.Context, status int, code, message string, fields map[string]string) {
	ctx.JSON(status, gin.H{
		"error": gin.H{
			"code": code, "message": message, "fields": fields,
		},
	})
}
