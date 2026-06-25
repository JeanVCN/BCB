package modules

import (
	"testing"

	"bcb/backend/internal/config"
	"github.com/gin-gonic/gin"
)

func TestNewRegistryWiresAccessTokenService(t *testing.T) {
	registry := New(Dependencies{
		Config: config.Config{JWTSecret: "01234567890123456789012345678901"},
	})

	if registry == nil {
		t.Fatalf("registry is nil")
	}
	if registry.TokenService() == nil {
		t.Fatalf("token service is nil")
	}
}

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	authenticated := api.Group("")
	admin := authenticated.Group("/admin")
	client := authenticated.Group("")

	registry := New(Dependencies{
		Config: config.Config{JWTSecret: "01234567890123456789012345678901"},
	})
	registry.RegisterRoutes(Routes{
		Public:        api,
		Authenticated: authenticated,
		Admin:         admin,
		Client:        client,
	})

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	expectedRoutes := []string{
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/login",
		"GET /api/v1/me",
		"GET /api/v1/admin/clients",
		"POST /api/v1/admin/clients/:clientId/activate",
		"GET /api/v1/billing",
		"GET /api/v1/conversations",
		"POST /api/v1/conversations/:conversationId/messages",
		"POST /api/v1/plan-change-requests",
		"GET /api/v1/admin/plan-change-requests",
	}

	for _, route := range expectedRoutes {
		if !registered[route] {
			t.Fatalf("route %q was not registered", route)
		}
	}
}
