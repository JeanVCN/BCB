package conversations

import "testing"

func TestNormalizePhone(t *testing.T) {
	got, err := NormalizePhone("+55 (11) 99999-9999")
	if err != nil {
		t.Fatalf("NormalizePhone() error = %v", err)
	}
	if got != "+5511999999999" {
		t.Fatalf("NormalizePhone() = %q, want %q", got, "+5511999999999")
	}
}

func TestNormalizePhoneRejectsLocalNumber(t *testing.T) {
	if _, err := NormalizePhone("11999999999"); err == nil {
		t.Fatal("NormalizePhone() error = nil, want E.164 validation error")
	}
}
