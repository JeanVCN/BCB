package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"bcb/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileNotFound     = errors.New("billing profile not found")
	ErrClientNotActive     = errors.New("client is not active")
	ErrPlanMismatch        = errors.New("operation is not allowed for current plan")
	ErrLimitBelowConsumed  = errors.New("postpaid limit is below consumed amount")
	ErrInsufficientBalance = errors.New("insufficient prepaid balance")
	ErrLimitExceeded       = errors.New("postpaid limit exceeded")
	ErrIdempotencyConflict = errors.New("idempotency key was used with a different request")
	ErrAlreadyProcessed    = errors.New("operation already processed")
)

type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a billing repository backed by PostgreSQL.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) chargeMessage(ctx context.Context, tx pgx.Tx, command MessageChargeCommand) (MessageChargeResult, error) {
	profile, err := repository.lockActiveProfile(ctx, tx, command.ClientID)
	if err != nil {
		return MessageChargeResult{}, err
	}

	transactionID := newID()
	transactionType, err := repository.applyChargeToProfile(ctx, tx, command.ClientID, command.AmountCents, &profile)
	if err != nil {
		return MessageChargeResult{}, err
	}

	if err := repository.insertFinancialTransaction(ctx, tx, financialTransactionInput{
		ID:             transactionID,
		ClientID:       command.ClientID,
		Type:           transactionType,
		AmountCents:    command.AmountCents,
		MessageID:      command.MessageID,
		ActorUserID:    command.ActorUserID,
		IdempotencyKey: command.IdempotencyKey,
		Reason:         "Cobrança de mensagem",
	}); err != nil {
		return MessageChargeResult{}, err
	}

	return MessageChargeResult{TransactionID: transactionID, Profile: profile}, nil
}

func (repository *Repository) profileInTransaction(ctx context.Context, tx pgx.Tx, clientID string) (Profile, error) {
	return repository.lockActiveProfile(ctx, tx, clientID)
}

func (repository *Repository) reverseMessageCharge(ctx context.Context, tx pgx.Tx, messageID string) error {
	charge, found, err := repository.findOriginalMessageCharge(ctx, tx, messageID)
	if err != nil || !found {
		return err
	}

	reversed, err := repository.chargeAlreadyReversed(ctx, tx, charge.TransactionID)
	if err != nil || reversed {
		return err
	}

	reversalType, err := repository.applyChargeReversalToProfile(ctx, tx, charge)
	if err != nil {
		return err
	}

	return repository.insertFinancialTransaction(ctx, tx, financialTransactionInput{
		ID:                    newID(),
		ClientID:              charge.ClientID,
		Type:                  reversalType,
		AmountCents:           charge.AmountCents,
		MessageID:             messageID,
		ReversesTransactionID: charge.TransactionID,
		IdempotencyKey:        "refund:" + messageID,
		Reason:                "Estorno por falha definitiva",
	})
}

func (repository *Repository) applyChargeToProfile(ctx context.Context, tx pgx.Tx, clientID string, amountCents int64, profile *Profile) (domain.FinancialTransactionType, error) {
	switch profile.PlanType {
	case string(domain.PlanPrepaid):
		if profile.PrepaidBalanceCents < amountCents {
			return "", ErrInsufficientBalance
		}
		_, err := tx.Exec(ctx, `
			UPDATE billing_profiles AS bp
			SET prepaid_balance_cents = bp.prepaid_balance_cents - $2,
			    version = bp.version + 1,
			    updated_at = NOW()
			WHERE bp.client_account_id = $1`, clientID, amountCents)
		if err != nil {
			return "", fmt.Errorf("decrease prepaid balance: %w", err)
		}
		profile.PrepaidBalanceCents -= amountCents
		fillProfileAvailability(profile)
		return domain.FinancialTransactionDebit, nil

	case string(domain.PlanPostpaid):
		if profile.PostpaidConsumedCents+amountCents > profile.PostpaidTotalLimitCents {
			return "", ErrLimitExceeded
		}
		_, err := tx.Exec(ctx, `
			UPDATE billing_profiles AS bp
			SET postpaid_consumed_cents = bp.postpaid_consumed_cents + $2,
			    version = bp.version + 1,
			    updated_at = NOW()
			WHERE bp.client_account_id = $1`, clientID, amountCents)
		if err != nil {
			return "", fmt.Errorf("increase postpaid consumption: %w", err)
		}
		profile.PostpaidConsumedCents += amountCents
		fillProfileAvailability(profile)
		return domain.FinancialTransactionConsumption, nil

	default:
		return "", ErrPlanMismatch
	}
}

func (repository *Repository) applyChargeReversalToProfile(ctx context.Context, tx pgx.Tx, charge originalMessageCharge) (domain.FinancialTransactionType, error) {
	switch charge.Type {
	case domain.FinancialTransactionDebit:
		_, err := tx.Exec(ctx, `
			UPDATE billing_profiles AS bp
			SET prepaid_balance_cents = bp.prepaid_balance_cents + $2,
			    version = bp.version + 1,
			    updated_at = NOW()
			WHERE bp.client_account_id = $1`, charge.ClientID, charge.AmountCents)
		if err != nil {
			return "", fmt.Errorf("refund prepaid charge: %w", err)
		}
		return domain.FinancialTransactionRefund, nil

	case domain.FinancialTransactionConsumption:
		_, err := tx.Exec(ctx, `
			UPDATE billing_profiles AS bp
			SET postpaid_consumed_cents = bp.postpaid_consumed_cents - $2,
			    version = bp.version + 1,
			    updated_at = NOW()
			WHERE bp.client_account_id = $1 AND bp.postpaid_consumed_cents >= $2`, charge.ClientID, charge.AmountCents)
		if err != nil {
			return "", fmt.Errorf("reverse postpaid consumption: %w", err)
		}
		return domain.FinancialTransactionConsumptionReversal, nil

	default:
		return "", fmt.Errorf("unsupported charge type for reversal: %s", charge.Type)
	}
}

type originalMessageCharge struct {
	TransactionID string
	ClientID      string
	Type          domain.FinancialTransactionType
	AmountCents   int64
}

func (repository *Repository) findOriginalMessageCharge(ctx context.Context, tx pgx.Tx, messageID string) (originalMessageCharge, bool, error) {
	var charge originalMessageCharge
	err := tx.QueryRow(ctx, `
		SELECT ft.id::text,
		       ft.client_account_id::text,
		       ft.type,
		       ft.amount_cents
		FROM financial_transactions AS ft
		WHERE ft.message_id = $1 AND ft.type IN ($2, $3)
		FOR UPDATE`,
		messageID,
		domain.FinancialTransactionDebit,
		domain.FinancialTransactionConsumption,
	).Scan(&charge.TransactionID, &charge.ClientID, &charge.Type, &charge.AmountCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return originalMessageCharge{}, false, nil
	}
	if err != nil {
		return originalMessageCharge{}, false, fmt.Errorf("read original charge: %w", err)
	}
	return charge, true, nil
}

func newID() string {
	return uuid.NewString()
}

// RequestHash produces a stable hash for idempotent billing operations.
func RequestHash(operation string, payload any) string {
	encoded, _ := json.Marshal(struct {
		Operation string `json:"operation"`
		Payload   any    `json:"payload"`
	}{Operation: operation, Payload: payload})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func nullableJSON(value []byte) any {
	if string(value) == "null" {
		return nil
	}
	return string(value)
}

func uniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}
