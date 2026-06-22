package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter() http.Handler {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health/live", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
