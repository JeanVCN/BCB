package access

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

var commonPasswords = map[string]struct{}{
	"password": {}, "password123": {}, "123456789": {}, "qwerty123": {},
	"admin123": {}, "senha123": {}, "senha123!": {}, "1234567890": {},
}

func ValidatePassword(password string) error {
	length := len([]rune(password))
	if length < 9 || length > 128 {
		return errors.New("must contain between 9 and 128 characters")
	}

	var hasLetter, hasNumber, hasSpecial bool
	for _, character := range password {
		switch {
		case unicode.IsLetter(character):
			hasLetter = true
		case unicode.IsNumber(character):
			hasNumber = true
		default:
			hasSpecial = true
		}
	}
	if !hasLetter || !hasNumber || !hasSpecial {
		return errors.New("must contain letters, numbers and a special character")
	}

	if _, found := commonPasswords[strings.ToLower(password)]; found {
		return errors.New("is too common")
	}

	return nil
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
