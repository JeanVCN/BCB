package billing

import (
	"errors"
	"testing"
)

func TestValidatePositiveMutation(t *testing.T) {
	tests := []struct {
		name           string
		amountCents    int64
		idempotencyKey string
		wantErr        error
	}{
		{name: "valid amount and key", amountCents: 1, idempotencyKey: "key-1"},
		{name: "zero amount", amountCents: 0, idempotencyKey: "key-1", wantErr: ErrInvalidAmount},
		{name: "negative amount", amountCents: -1, idempotencyKey: "key-1", wantErr: ErrInvalidAmount},
		{name: "missing key", amountCents: 1, idempotencyKey: "  ", wantErr: ErrMissingIdempotencyKey},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePositiveMutation(test.amountCents, test.idempotencyKey)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestValidateNonNegativeMutation(t *testing.T) {
	tests := []struct {
		name           string
		amountCents    int64
		idempotencyKey string
		wantErr        error
	}{
		{name: "zero amount is accepted", amountCents: 0, idempotencyKey: "key-1"},
		{name: "positive amount is accepted", amountCents: 1, idempotencyKey: "key-1"},
		{name: "negative amount", amountCents: -1, idempotencyKey: "key-1", wantErr: ErrInvalidAmount},
		{name: "missing key", amountCents: 0, idempotencyKey: "", wantErr: ErrMissingIdempotencyKey},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateNonNegativeMutation(test.amountCents, test.idempotencyKey)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestRequestHash(t *testing.T) {
	payload := struct {
		AmountCents int64  `json:"amountCents"`
		Reason      string `json:"reason"`
	}{AmountCents: 100, Reason: "Ajuste"}

	first := RequestHash("admin.credit", payload)
	second := RequestHash("admin.credit", payload)
	if first == "" {
		t.Fatalf("hash must not be empty")
	}
	if first != second {
		t.Fatalf("hash is not stable: %q != %q", first, second)
	}

	changedOperation := RequestHash("admin.zero_current_balance", payload)
	if first == changedOperation {
		t.Fatalf("hash should include the operation name")
	}
}
