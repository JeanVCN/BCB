package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

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
	ErrIdempotencyConflict = errors.New("idempotency key was used with a different request")
	ErrAlreadyProcessed    = errors.New("operation already processed")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) Profile(ctx context.Context, clientID string) (Profile, error) {
	row := repository.pool.QueryRow(ctx, `
		SELECT billing_profiles.plan_type,
		       billing_profiles.prepaid_balance_cents,
		       billing_profiles.postpaid_total_limit_cents,
		       billing_profiles.postpaid_consumed_cents,
		       billing_profiles.updated_at,
		       client_accounts.status
		FROM billing_profiles
		JOIN client_accounts ON client_accounts.id = billing_profiles.client_account_id
		WHERE billing_profiles.client_account_id = $1`, clientID)
	profile, status, err := scanProfileWithStatus(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("read billing profile: %w", err)
	}
	if status != "active" {
		return Profile{}, ErrClientNotActive
	}
	return profile, nil
}

func (repository *Repository) Transactions(ctx context.Context, clientID string) ([]Transaction, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id::text, type, amount_cents, message_id::text, reverses_transaction_id::text,
		       actor_user_id::text, idempotency_key, reason, created_at
		FROM financial_transactions
		WHERE client_account_id = $1
		ORDER BY created_at DESC`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list financial transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]Transaction, 0)
	for rows.Next() {
		var transaction Transaction
		if err := rows.Scan(
			&transaction.ID,
			&transaction.Type,
			&transaction.AmountCents,
			&transaction.MessageID,
			&transaction.ReversesTransactionID,
			&transaction.ActorUserID,
			&transaction.IdempotencyKey,
			&transaction.Reason,
			&transaction.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan financial transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}
	return transactions, rows.Err()
}

func (repository *Repository) AddCredit(ctx context.Context, actorID, clientID string, amountCents int64, reason, idempotencyKey, requestHash string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add credit: %w", err)
	}
	defer tx.Rollback(ctx)

	idempotencyRecordID, err := repository.registerIdempotency(ctx, tx, clientID, "admin.credit", idempotencyKey, requestHash)
	if err != nil {
		return err
	}

	profile, err := repository.lockProfile(ctx, tx, clientID)
	if err != nil {
		return err
	}
	if profile.PlanType != "prepaid" {
		return ErrPlanMismatch
	}

	_, err = tx.Exec(ctx, `
		UPDATE billing_profiles
		SET prepaid_balance_cents = prepaid_balance_cents + $2,
		    version = version + 1,
		    updated_at = NOW()
		WHERE client_account_id = $1`, clientID, amountCents)
	if err != nil {
		return fmt.Errorf("increase prepaid balance: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO financial_transactions (
			id, client_account_id, type, amount_cents, actor_user_id,
			idempotency_key, idempotency_record_id, reason
		) VALUES ($1, $2, 'credit', $3, $4, $5, $6, NULLIF($7, ''))`,
		uuid.NewString(), clientID, amountCents, actorID, idempotencyKey, idempotencyRecordID, reason,
	)
	if err != nil {
		return fmt.Errorf("insert credit transaction: %w", err)
	}

	return tx.Commit(ctx)
}

func (repository *Repository) AdjustPostpaidLimit(ctx context.Context, actorID, clientID string, totalLimitCents int64, reason, idempotencyKey, requestHash string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin adjust postpaid limit: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := repository.registerIdempotency(ctx, tx, clientID, "admin.postpaid_limit", idempotencyKey, requestHash); err != nil {
		return err
	}

	profile, err := repository.lockProfile(ctx, tx, clientID)
	if err != nil {
		return err
	}
	if profile.PlanType != "postpaid" {
		return ErrPlanMismatch
	}
	if totalLimitCents < profile.PostpaidConsumedCents {
		return ErrLimitBelowConsumed
	}

	_, err = tx.Exec(ctx, `
		UPDATE billing_profiles
		SET postpaid_total_limit_cents = $2,
		    version = version + 1,
		    updated_at = NOW()
		WHERE client_account_id = $1`, clientID, totalLimitCents)
	if err != nil {
		return fmt.Errorf("update postpaid limit: %w", err)
	}

	if err := insertAudit(ctx, tx, actorID, "billing.postpaid_limit_adjusted", clientID, reason, map[string]any{"totalLimitCents": profile.PostpaidTotalLimitCents}, map[string]any{"totalLimitCents": totalLimitCents}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Repository) registerIdempotency(ctx context.Context, tx pgx.Tx, clientID, operation, key, hash string) (string, error) {
	recordID := uuid.NewString()
	_, err := tx.Exec(ctx, `
		INSERT INTO idempotency_records (id, client_account_id, operation, idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4, $5)`,
		recordID, clientID, operation, key, hash,
	)
	if uniqueViolation(err) {
		var previousHash string
		err = tx.QueryRow(ctx, `
			SELECT request_hash
			FROM idempotency_records
			WHERE client_account_id = $1 AND operation = $2 AND idempotency_key = $3`,
			clientID, operation, key,
		).Scan(&previousHash)
		if err != nil {
			return "", fmt.Errorf("read idempotency record: %w", err)
		}
		if previousHash != hash {
			return "", ErrIdempotencyConflict
		}
		return "", ErrAlreadyProcessed
	}
	if err != nil {
		return "", fmt.Errorf("insert idempotency record: %w", err)
	}
	return recordID, nil
}

func (repository *Repository) lockProfile(ctx context.Context, tx pgx.Tx, clientID string) (Profile, error) {
	row := tx.QueryRow(ctx, `
		SELECT plan_type, prepaid_balance_cents, postpaid_total_limit_cents,
		       postpaid_consumed_cents, updated_at
		FROM billing_profiles
		WHERE client_account_id = $1
		FOR UPDATE`, clientID)
	profile, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("lock billing profile: %w", err)
	}
	return profile, nil
}

func scanProfile(row pgx.Row) (Profile, error) {
	var profile Profile
	if err := row.Scan(
		&profile.PlanType,
		&profile.PrepaidBalanceCents,
		&profile.PostpaidTotalLimitCents,
		&profile.PostpaidConsumedCents,
		&profile.UpdatedAt,
	); err != nil {
		return Profile{}, err
	}
	profile.PostpaidAvailableCents = profile.PostpaidTotalLimitCents - profile.PostpaidConsumedCents
	if profile.PlanType == "prepaid" {
		profile.CurrentPlanAvailableCents = profile.PrepaidBalanceCents
	} else {
		profile.CurrentPlanAvailableCents = profile.PostpaidAvailableCents
	}
	return profile, nil
}

func scanProfileWithStatus(row pgx.Row) (Profile, string, error) {
	var profile Profile
	var status string
	if err := row.Scan(
		&profile.PlanType,
		&profile.PrepaidBalanceCents,
		&profile.PostpaidTotalLimitCents,
		&profile.PostpaidConsumedCents,
		&profile.UpdatedAt,
		&status,
	); err != nil {
		return Profile{}, "", err
	}
	profile.PostpaidAvailableCents = profile.PostpaidTotalLimitCents - profile.PostpaidConsumedCents
	if profile.PlanType == "prepaid" {
		profile.CurrentPlanAvailableCents = profile.PrepaidBalanceCents
	} else {
		profile.CurrentPlanAvailableCents = profile.PostpaidAvailableCents
	}
	return profile, status, nil
}

func RequestHash(operation string, payload any) string {
	encoded, _ := json.Marshal(struct {
		Operation string `json:"operation"`
		Payload   any    `json:"payload"`
	}{Operation: operation, Payload: payload})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func insertAudit(ctx context.Context, tx pgx.Tx, actorID, action, targetID, reason string, previous, next any) error {
	previousJSON, _ := json.Marshal(previous)
	nextJSON, _ := json.Marshal(next)
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id, actor_user_id, action, target_type, target_id, reason, previous_values, new_values)
		VALUES ($1, $2, $3, 'client_account', $4, NULLIF($5, ''), $6::jsonb, $7::jsonb)`,
		uuid.NewString(), actorID, action, targetID, reason, nullableJSON(previousJSON), nullableJSON(nextJSON),
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
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
