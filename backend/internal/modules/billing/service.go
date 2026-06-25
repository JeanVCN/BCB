package billing

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidAmount         = errors.New("amount must be greater than zero")
	ErrMissingIdempotencyKey = errors.New("idempotency key is required")
)

type Service struct {
	repository *Repository
	locks      *LockManager
}

// NewService creates the billing service that coordinates repository access and locks.
func NewService(repository *Repository, locks *LockManager) *Service {
	return &Service{repository: repository, locks: locks}
}

func (service *Service) profile(ctx context.Context, clientID string) (Profile, error) {
	return service.repository.profile(ctx, clientID)
}

func (service *Service) transactions(ctx context.Context, clientID string) ([]Transaction, error) {
	return service.repository.transactions(ctx, clientID)
}

// ChargeMessage applies the financial charge for a message inside the caller transaction.
func (service *Service) ChargeMessage(ctx context.Context, tx pgx.Tx, command MessageChargeCommand) (MessageChargeResult, error) {
	return service.repository.chargeMessage(ctx, tx, command)
}

// ReverseMessageCharge reverses a previously charged message inside the caller transaction.
func (service *Service) ReverseMessageCharge(ctx context.Context, tx pgx.Tx, messageID string) error {
	return service.repository.reverseMessageCharge(ctx, tx, messageID)
}

// ProfileInTransaction returns the locked billing profile inside the caller transaction.
func (service *Service) ProfileInTransaction(ctx context.Context, tx pgx.Tx, clientID string) (Profile, error) {
	return service.repository.profileInTransaction(ctx, tx, clientID)
}

func (service *Service) addCredit(ctx context.Context, actorID, clientID string, amountCents int64, reason, idempotencyKey string) error {
	if err := validatePositiveMutation(amountCents, idempotencyKey); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	hash := RequestHash("admin.credit", struct {
		AmountCents int64  `json:"amountCents"`
		Reason      string `json:"reason"`
	}{AmountCents: amountCents, Reason: reason})
	return service.locks.WithClientLock(ctx, clientID, func() error {
		err := service.repository.addCredit(ctx, actorID, clientID, amountCents, reason, strings.TrimSpace(idempotencyKey), hash)
		if errors.Is(err, ErrAlreadyProcessed) {
			return nil
		}
		return err
	})
}

func (service *Service) adjustPostpaidLimit(ctx context.Context, actorID, clientID string, totalLimitCents int64, reason, idempotencyKey string) error {
	if err := validateNonNegativeMutation(totalLimitCents, idempotencyKey); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	hash := RequestHash("admin.postpaid_limit", struct {
		TotalLimitCents int64  `json:"totalLimitCents"`
		Reason          string `json:"reason"`
	}{TotalLimitCents: totalLimitCents, Reason: reason})
	return service.locks.WithClientLock(ctx, clientID, func() error {
		err := service.repository.adjustPostpaidLimit(ctx, actorID, clientID, totalLimitCents, reason, strings.TrimSpace(idempotencyKey), hash)
		if errors.Is(err, ErrAlreadyProcessed) {
			return nil
		}
		return err
	})
}

func (service *Service) zeroCurrentBalance(ctx context.Context, actorID, clientID, reason, idempotencyKey string) error {
	if strings.TrimSpace(idempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}
	reason = strings.TrimSpace(reason)
	hash := RequestHash("admin.zero_current_balance", struct {
		Reason string `json:"reason"`
	}{Reason: reason})
	return service.locks.WithClientLock(ctx, clientID, func() error {
		err := service.repository.zeroCurrentBalance(ctx, actorID, clientID, reason, strings.TrimSpace(idempotencyKey), hash)
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
