package billing

import (
	"context"
	"encoding/json"
	"fmt"

	"bcb/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) transactions(ctx context.Context, clientID string) ([]Transaction, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT ft.id::text,
		       ft.type,
		       ft.amount_cents,
		       ft.message_id::text,
		       ft.reverses_transaction_id::text,
		       ft.actor_user_id::text,
		       ft.idempotency_key,
		       ft.reason,
		       ft.created_at
		FROM financial_transactions AS ft
		WHERE ft.client_account_id = $1
		ORDER BY ft.created_at DESC`, clientID)
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

type financialTransactionInput struct {
	ID                    string
	ClientID              string
	Type                  domain.FinancialTransactionType
	AmountCents           int64
	MessageID             string
	ActorUserID           string
	ReversesTransactionID string
	IdempotencyKey        string
	IdempotencyRecordID   string
	Reason                string
}

func (repository *Repository) insertFinancialTransaction(ctx context.Context, tx pgx.Tx, input financialTransactionInput) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO financial_transactions AS ft (
			id,
			client_account_id,
			type,
			amount_cents,
			message_id,
			reverses_transaction_id,
			actor_user_id,
			idempotency_key,
			idempotency_record_id,
			reason
		) VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, NULLIF($6, '')::uuid, NULLIF($7, '')::uuid, $8, NULLIF($9, '')::uuid, NULLIF($10, ''))`,
		input.ID,
		input.ClientID,
		input.Type,
		input.AmountCents,
		input.MessageID,
		input.ReversesTransactionID,
		input.ActorUserID,
		input.IdempotencyKey,
		input.IdempotencyRecordID,
		input.Reason,
	)
	if err != nil {
		return fmt.Errorf("insert financial transaction: %w", err)
	}
	return nil
}

func (repository *Repository) registerIdempotency(ctx context.Context, tx pgx.Tx, clientID, operation, key, hash string) (string, error) {
	recordID := newID()
	_, err := tx.Exec(ctx, `
		INSERT INTO idempotency_records AS ir (id, client_account_id, operation, idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4, $5)`,
		recordID, clientID, operation, key, hash,
	)
	if uniqueViolation(err) {
		var previousHash string
		err = tx.QueryRow(ctx, `
			SELECT ir.request_hash
			FROM idempotency_records AS ir
			WHERE ir.client_account_id = $1 AND ir.operation = $2 AND ir.idempotency_key = $3`,
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

func (repository *Repository) chargeAlreadyReversed(ctx context.Context, tx pgx.Tx, transactionID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM financial_transactions AS ft
			WHERE ft.reverses_transaction_id = $1
		)`, transactionID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check charge reversal: %w", err)
	}
	return exists, nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, actorID, action, targetID, reason string, previous, next any) error {
	previousJSON, _ := json.Marshal(previous)
	nextJSON, _ := json.Marshal(next)
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events AS ae (id, actor_user_id, action, target_type, target_id, reason, previous_values, new_values)
		VALUES ($1, $2, $3, 'client_account', $4, NULLIF($5, ''), $6::jsonb, $7::jsonb)`,
		newID(), actorID, action, targetID, reason, nullableJSON(previousJSON), nullableJSON(nextJSON),
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}
