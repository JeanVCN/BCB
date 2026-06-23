package billing

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidAmount         = errors.New("amount must be greater than zero")
	ErrMissingIdempotencyKey = errors.New("idempotency key is required")
)

type Service struct {
	repository *Repository
	locks      *LockManager
}

func NewService(repository *Repository, locks *LockManager) *Service {
	return &Service{repository: repository, locks: locks}
}

func (service *Service) Profile(ctx context.Context, clientID string) (Profile, error) {
	return service.repository.Profile(ctx, clientID)
}

func (service *Service) Transactions(ctx context.Context, clientID string) ([]Transaction, error) {
	return service.repository.Transactions(ctx, clientID)
}

func (service *Service) AddCredit(ctx context.Context, actorID, clientID string, amountCents int64, reason, idempotencyKey string) error {
	if err := validatePositiveMutation(amountCents, idempotencyKey); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	hash := RequestHash("admin.credit", struct {
		AmountCents int64  `json:"amountCents"`
		Reason      string `json:"reason"`
	}{AmountCents: amountCents, Reason: reason})
	return service.locks.WithClientLock(ctx, clientID, func() error {
		err := service.repository.AddCredit(ctx, actorID, clientID, amountCents, reason, strings.TrimSpace(idempotencyKey), hash)
		if errors.Is(err, ErrAlreadyProcessed) {
			return nil
		}
		return err
	})
}

func (service *Service) AdjustPostpaidLimit(ctx context.Context, actorID, clientID string, totalLimitCents int64, reason, idempotencyKey string) error {
	if err := validateNonNegativeMutation(totalLimitCents, idempotencyKey); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	hash := RequestHash("admin.postpaid_limit", struct {
		TotalLimitCents int64  `json:"totalLimitCents"`
		Reason          string `json:"reason"`
	}{TotalLimitCents: totalLimitCents, Reason: reason})
	return service.locks.WithClientLock(ctx, clientID, func() error {
		err := service.repository.AdjustPostpaidLimit(ctx, actorID, clientID, totalLimitCents, reason, strings.TrimSpace(idempotencyKey), hash)
		if errors.Is(err, ErrAlreadyProcessed) {
			return nil
		}
		return err
	})
}

func validatePositiveMutation(amountCents int64, idempotencyKey string) error {
	if amountCents <= 0 {
		return ErrInvalidAmount
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}
	return nil
}

func validateNonNegativeMutation(amountCents int64, idempotencyKey string) error {
	if amountCents < 0 {
		return ErrInvalidAmount
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}
	return nil
}
