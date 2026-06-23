package conversations

import (
	"errors"
	"strings"
	"unicode"
)

func NormalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("phone is required")
	}

	var builder strings.Builder
	for index, character := range value {
		switch {
		case character == '+' && index == 0:
			builder.WriteRune(character)
		case unicode.IsDigit(character):
			builder.WriteRune(character)
		case character == ' ' || character == '-' || character == '(' || character == ')':
			continue
		default:
			return "", errors.New("phone must be in E.164 format")
		}
	}

	normalized := builder.String()
	digits := strings.TrimPrefix(normalized, "+")
	if !strings.HasPrefix(normalized, "+") || len(digits) < 8 || len(digits) > 15 {
		return "", errors.New("phone must be in E.164 format")
	}

	return normalized, nil
}
