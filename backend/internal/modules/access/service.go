package access

import (
	"context"
	"errors"
	"strings"

	"bcb/backend/internal/domain"
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

func newService(repository *Repository, tokens *TokenService, limiter *RateLimiter) *Service {
	return &Service{repository: repository, tokens: tokens, limiter: limiter}
}

func (service *Service) login(ctx context.Context, login, password string) (string, User, error) {
	normalizedLogin := normalizeLogin(login)
	if _, blocked, err := service.limiter.blocked(ctx, normalizedLogin); err != nil {
		return "", User{}, err
	} else if blocked {
		return "", User{}, ErrRateLimited
	}

	user, err := service.repository.userByLogin(ctx, normalizedLogin)
	if err != nil || !verifyPassword(password, user.PasswordHash) {
		_ = service.limiter.registerFailure(ctx, normalizedLogin)
		return "", User{}, ErrInvalidCredentials
	}
	if !user.Enabled || (user.Role == domain.RoleClient && (user.ClientStatus == nil || *user.ClientStatus != string(domain.ClientStatusActive))) {
		_ = service.limiter.registerFailure(ctx, normalizedLogin)
		return "", User{}, ErrClientInactive
	}

	if err := service.limiter.reset(ctx, normalizedLogin); err != nil {
		return "", User{}, err
	}
	token, err := service.tokens.issue(user)
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
