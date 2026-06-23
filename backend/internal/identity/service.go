package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrClientInactive     = errors.New("client is not active")
	ErrRateLimited        = errors.New("authentication rate limited")
)

type Service struct {
	store   *Store
	tokens  *TokenService
	limiter *RateLimiter
}

func NewService(store *Store, tokens *TokenService, limiter *RateLimiter) *Service {
	return &Service{store: store, tokens: tokens, limiter: limiter}
}

func (service *Service) Register(ctx context.Context, name, documentType, document, password, plan string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("name is required")
	}
	if plan != "prepaid" && plan != "postpaid" {
		return "", errors.New("requestedPlan must be prepaid or postpaid")
	}
	normalizedDocument, err := NormalizeDocument(document, documentType)
	if err != nil {
		return "", err
	}
	if err := ValidatePassword(password); err != nil {
		return "", fmt.Errorf("password: %w", err)
	}
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}
	return service.store.Register(ctx, Registration{
		Name: name, DocumentType: documentType, Document: normalizedDocument,
		PasswordHash: hash, RequestedPlan: plan,
	})
}

func (service *Service) Login(ctx context.Context, login, password string) (string, User, error) {
	normalizedLogin := normalizeLogin(login)
	if _, blocked, err := service.limiter.Blocked(ctx, normalizedLogin); err != nil {
		return "", User{}, err
	} else if blocked {
		return "", User{}, ErrRateLimited
	}

	user, err := service.store.UserByLogin(ctx, normalizedLogin)
	if err != nil || !VerifyPassword(password, user.PasswordHash) {
		_ = service.limiter.RegisterFailure(ctx, normalizedLogin)
		return "", User{}, ErrInvalidCredentials
	}
	if !user.Enabled || (user.Role == "client" && (user.ClientStatus == nil || *user.ClientStatus != "active")) {
		_ = service.limiter.RegisterFailure(ctx, normalizedLogin)
		return "", User{}, ErrClientInactive
	}

	if err := service.limiter.Reset(ctx, normalizedLogin); err != nil {
		return "", User{}, err
	}
	token, err := service.tokens.Issue(user)
	return token, user, err
}

func normalizeLogin(login string) string {
	login = strings.TrimSpace(strings.ToLower(login))
	var digits strings.Builder
	for _, character := range login {
		if character >= '0' && character <= '9' {
			digits.WriteRune(character)
		}
	}
	if digits.Len() == 11 || digits.Len() == 14 {
		return digits.String()
	}
	return login
}
