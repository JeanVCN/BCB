package access

import (
	"testing"

	"bcb/backend/internal/domain"
)

func TestTokenRoundTrip(t *testing.T) {
	clientID := "client-id"
	service := newTokenService("a-secret-with-at-least-thirty-two-characters")
	value, err := service.issue(User{ID: "user-id", Role: domain.RoleClient, ClientAccountID: &clientID})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	claims, err := service.Parse(value)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.Subject != "user-id" || claims.Role != domain.RoleClient || claims.ClientID == nil || *claims.ClientID != clientID {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
