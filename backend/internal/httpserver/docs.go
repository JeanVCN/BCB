package httpserver

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate cp ../../../project-docs/openapi.yaml openapi.yaml

//go:embed openapi.yaml
var openAPISpec []byte

const swaggerHTML = `<!doctype html>
<html lang="pt-BR">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Big Chat Brasil API - Swagger</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
    <style>
      body { margin: 0; background: #f7f8fb; }
      .topbar { display: none; }
      .swagger-ui .info { margin: 32px 0; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
      window.onload = function () {
        window.ui = SwaggerUIBundle({
          url: "/openapi.yaml",
          dom_id: "#swagger-ui",
          deepLinking: true,
          displayRequestDuration: true,
          persistAuthorization: true,
          presets: [
            SwaggerUIBundle.presets.apis,
            SwaggerUIStandalonePreset
          ],
          layout: "StandaloneLayout"
        })
      }
    </script>
  </body>
</html>`

func registerDocumentationRoutes(router *gin.Engine) {
	router.GET("/swagger", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusMovedPermanently, "/swagger.html")
	})
	router.GET("/swagger.html", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
	router.GET("/openapi.yaml", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", openAPISpec)
	})
}
