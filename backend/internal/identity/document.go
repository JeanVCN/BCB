package identity

import (
	"errors"
	"strings"
	"unicode"
)

func NormalizeDocument(value, documentType string) (string, error) {
	var digits strings.Builder
	for _, character := range value {
		if unicode.IsDigit(character) {
			digits.WriteRune(character)
		}
	}

	normalized := digits.String()
	switch documentType {
	case "cpf":
		if !validCPF(normalized) {
			return "", errors.New("invalid CPF")
		}
	case "cnpj":
		if !validCNPJ(normalized) {
			return "", errors.New("invalid CNPJ")
		}
	default:
		return "", errors.New("documentType must be cpf or cnpj")
	}

	return normalized, nil
}

func validCPF(value string) bool {
	if len(value) != 11 || allDigitsEqual(value) {
		return false
	}
	return checkDigit(value[:9], []int{10, 9, 8, 7, 6, 5, 4, 3, 2}) == int(value[9]-'0') &&
		checkDigit(value[:10], []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}) == int(value[10]-'0')
}

func validCNPJ(value string) bool {
	if len(value) != 14 || allDigitsEqual(value) {
		return false
	}
	return checkDigit(value[:12], []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}) == int(value[12]-'0') &&
		checkDigit(value[:13], []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}) == int(value[13]-'0')
}

func checkDigit(value string, weights []int) int {
	sum := 0
	for index, character := range value {
		sum += int(character-'0') * weights[index]
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

func allDigitsEqual(value string) bool {
	for index := 1; index < len(value); index++ {
		if value[index] != value[0] {
			return false
		}
	}
	return true
}
