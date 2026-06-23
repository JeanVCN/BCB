package messages

import (
	"context"
	"errors"
	"fmt"

	"bcb/backend/internal/domain"
	"bcb/backend/internal/modules/billing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (repository *Repository) findExistingMessage(ctx context.Context, tx pgx.Tx, clientID, idempotencyKey string) (existingMessage, bool, error) {
	row := tx.QueryRow(ctx, `
		SELECT m.id::text,
		       m.conversation_id::text,
		       m.content,
		       m.channel,
		       m.priority,
		       m.cost_cents,
		       m.status,
		       m.failure_code,
		       m.created_at,
		       m.queued_at,
		       m.processing_at,
		       m.sent_at,
		       m.failed_at,
		       m.request_hash
		FROM messages AS m
		WHERE m.client_account_id = $1 AND m.idempotency_key = $2`, clientID, idempotencyKey)
	message, hash, err := scanMessageWithOptionalHash(row, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return existingMessage{}, false, nil
	}
	if err != nil {
		return existingMessage{}, false, fmt.Errorf("read idempotent message: %w", err)
	}
	return existingMessage{message: message, requestHash: hash}, true, nil
}

func (repository *Repository) lockConversation(ctx context.Context, tx pgx.Tx, clientID, conversationID string) (string, error) {
	var recipientID string
	err := tx.QueryRow(ctx, `
		SELECT c.recipient_id::text
		FROM conversations AS c
		WHERE c.id = $1 AND c.client_account_id = $2
		FOR UPDATE OF c`, conversationID, clientID).Scan(&recipientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrConversationNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lock conversation: %w", err)
	}
	return recipientID, nil
}

func (repository *Repository) ensureConversationOwnership(ctx context.Context, clientID, conversationID string) error {
	var exists bool
	err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM conversations AS c
			WHERE c.id = $1 AND c.client_account_id = $2
		)`, conversationID, clientID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check conversation ownership: %w", err)
	}
	if !exists {
		return ErrConversationNotFound
	}
	return nil
}

func (repository *Repository) ensureActiveClient(ctx context.Context, clientID string) error {
	var status domain.ClientStatus
	err := repository.pool.QueryRow(ctx, `
		SELECT ca.status
		FROM client_accounts AS ca
		WHERE ca.id = $1`, clientID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) || status != domain.ClientStatusActive {
		return ErrClientInactive
	}
	if err != nil {
		return fmt.Errorf("check client status: %w", err)
	}
	return nil
}

type messageScanner interface {
	Scan(dest ...any) error
}

func scanMessage(rows pgx.Rows) (Message, error) {
	message, _, err := scanMessageWithOptionalHash(rows, false)
	if err != nil {
		return Message{}, fmt.Errorf("scan message: %w", err)
	}
	return message, nil
}

func scanMessageWithOptionalHash(scanner messageScanner, includeHash bool) (Message, string, error) {
	var message Message
	var hash string
	dest := []any{
		&message.ID,
		&message.ConversationID,
		&message.Content,
		&message.Channel,
		&message.Priority,
		&message.CostCents,
		&message.Status,
		&message.FailureCode,
		&message.CreatedAt,
		&message.QueuedAt,
		&message.ProcessingAt,
		&message.SentAt,
		&message.FailedAt,
	}
	if includeHash {
		dest = append(dest, &hash)
	}
	err := scanner.Scan(dest...)
	return message, hash, err
}

func summaryFromBillingProfile(profile billing.Profile) BillingSummary {
	available := profile.PostpaidTotalLimitCents - profile.PostpaidConsumedCents
	summary := BillingSummary{
		PlanType:                domain.PlanType(profile.PlanType),
		PrepaidBalanceCents:     profile.PrepaidBalanceCents,
		PostpaidTotalLimitCents: profile.PostpaidTotalLimitCents,
		PostpaidConsumedCents:   profile.PostpaidConsumedCents,
		PostpaidAvailableCents:  available,
	}
	if profile.PlanType == string(domain.PlanPrepaid) {
		summary.CurrentPlanAvailableCents = profile.PrepaidBalanceCents
	} else {
		summary.CurrentPlanAvailableCents = available
	}
	return summary
}

func uniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}
