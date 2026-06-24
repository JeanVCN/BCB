package accounts

import (
	"testing"

	"bcb/backend/internal/domain"
)

func TestNormalizeDocument(t *testing.T) {
	tests := []struct {
		name, value, documentType, want string
	}{
		{name: "CPF", value: "529.982.247-25", documentType: string(domain.DocumentCPF), want: "52998224725"},
		{name: "CNPJ", value: "11.222.333/0001-81", documentType: string(domain.DocumentCNPJ), want: "11222333000181"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeDocument(test.value, test.documentType)
			if err != nil {
				t.Fatalf("normalizeDocument() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("normalizeDocument() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNormalizeDocumentRejectsInvalidValue(t *testing.T) {
	if _, err := normalizeDocument("111.111.111-11", string(domain.DocumentCPF)); err == nil {
		t.Fatal("normalizeDocument() error = nil, want invalid CPF")
	}
}
