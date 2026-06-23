package access

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrClientInactive     = errors.New("client is not active")
	ErrRateLimited        = errors.New("authentication rate limited")
)

type Service struct {
	repository *Repository
	tokens     *TokenService
	limiter    *RateLimiter
}

func NewService(repository *Repository, tokens *TokenService, limiter *RateLimiter) *Service {
	return &Service{repository: repository, tokens: tokens, limiter: limiter}
}

func (service *Service) Login(ctx context.Context, login, password string) (string, User, error) {
	normalizedLogin := normalizeLogin(login)
	if _, blocked, err := service.limiter.Blocked(ctx, normalizedLogin); err != nil {
		return "", User{}, err
	} else if blocked {
		return "", User{}, ErrRateLimited
	}

	user, err := service.repository.UserByLogin(ctx, normalizedLogin)
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
