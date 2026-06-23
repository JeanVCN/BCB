package identity

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	password := "Correct-Horse-123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !VerifyPassword(password, hash) {
		t.Fatal("VerifyPassword() = false, want true")
	}
	if VerifyPassword("Wrong-Password-123!", hash) {
		t.Fatal("VerifyPassword() accepted wrong password")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("Correct-Horse-123!"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	if err := ValidatePassword("short1!"); err == nil {
		t.Fatal("short password accepted")
	}
}
