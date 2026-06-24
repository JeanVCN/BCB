package conversations

import "testing"

func TestNormalizePhone(t *testing.T) {
	got, err := normalizePhone("+55 (11) 99999-9999")
	if err != nil {
		t.Fatalf("normalizePhone() error = %v", err)
	}
	if got != "+5511999999999" {
		t.Fatalf("normalizePhone() = %q, want %q", got, "+5511999999999")
	}
}

func TestNormalizePhoneRejectsLocalNumber(t *testing.T) {
	if _, err := normalizePhone("11999999999"); err == nil {
		t.Fatal("normalizePhone() error = nil, want E.164 validation error")
	}
}
