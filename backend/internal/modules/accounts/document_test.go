package accounts

import "testing"

func TestNormalizeDocument(t *testing.T) {
	tests := []struct {
		name, value, documentType, want string
	}{
		{name: "CPF", value: "529.982.247-25", documentType: "cpf", want: "52998224725"},
		{name: "CNPJ", value: "11.222.333/0001-81", documentType: "cnpj", want: "11222333000181"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeDocument(test.value, test.documentType)
			if err != nil {
				t.Fatalf("NormalizeDocument() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("NormalizeDocument() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNormalizeDocumentRejectsInvalidValue(t *testing.T) {
	if _, err := NormalizeDocument("111.111.111-11", "cpf"); err == nil {
		t.Fatal("NormalizeDocument() error = nil, want invalid CPF")
	}
}
