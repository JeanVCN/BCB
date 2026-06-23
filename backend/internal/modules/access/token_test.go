package access

import "testing"

func TestTokenRoundTrip(t *testing.T) {
	clientID := "client-id"
	service := NewTokenService("a-secret-with-at-least-thirty-two-characters")
	value, err := service.Issue(User{ID: "user-id", Role: "client", ClientAccountID: &clientID})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	claims, err := service.Parse(value)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.Subject != "user-id" || claims.Role != "client" || claims.ClientID == nil || *claims.ClientID != clientID {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
