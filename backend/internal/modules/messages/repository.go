package messages

import (
	"context"
	"errors"
	"fmt"

	"bcb/backend/internal/domain"
	"bcb/backend/internal/modules/billing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrClientInactive       = errors.New("client is not active")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrIdempotencyConflict  = errors.New("idempotency key was used with a different request")
	ErrInvalidMessage       = errors.New("invalid message")
	ErrAlreadyProcessed     = errors.New("message already processed")
)

type Repository struct {
	pool    *pgxpool.Pool
	billing BillingGateway
}

type BillingGateway interface {
	ChargeMessage(context.Context, pgx.Tx, billing.MessageChargeCommand) (billing.MessageChargeResult, error)
	ReverseMessageCharge(context.Context, pgx.Tx, string) error
	ProfileInTransaction(context.Context, pgx.Tx, string) (billing.Profile, error)
}

func NewRepository(pool *pgxpool.Pool, billing BillingGateway) *Repository {
	return &Repository{pool: pool, billing: billing}
}

func (repository *Repository) send(ctx context.Context, command SendCommand) (SendResult, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return SendResult{}, fmt.Errorf("begin send message: %w", err)
	}
	defer tx.Rollback(ctx)

	existing, found, err := repository.findExistingMessage(ctx, tx, command.ClientID, command.IdempotencyKey)
	if err != nil {
		return SendResult{}, err
	}
	if found {
		if existing.requestHash != command.RequestHash {
			return SendResult{}, ErrIdempotencyConflict
		}
		profile, err := repository.billing.ProfileInTransaction(ctx, tx, command.ClientID)
		if err != nil {
			return SendResult{}, err
		}
		return SendResult{Message: existing.message, Billing: summaryFromBillingProfile(profile)}, nil
	}

	recipientID, err := repository.lockConversation(ctx, tx, command.ClientID, command.ConversationID)
	if err != nil {
		return SendResult{}, err
	}

	command.MessageID = uuid.NewString()
	charge, err := repository.billing.ChargeMessage(ctx, tx, billing.MessageChargeCommand{
		ClientID:       command.ClientID,
		MessageID:      command.MessageID,
		ActorUserID:    command.RequestedByUserID,
		AmountCents:    command.CostCents,
		IdempotencyKey: command.IdempotencyKey,
	})
	if err != nil {
		return SendResult{}, err
	}

	message, err := repository.insertMessage(ctx, tx, command, recipientID, charge.TransactionID)
	if err != nil {
		return SendResult{}, err
	}

	if err := repository.insertDispatchJob(ctx, tx, message.ID, command.Priority); err != nil {
		return SendResult{}, err
	}

	if err := repository.touchConversation(ctx, tx, command.ClientID, command.ConversationID); err != nil {
		return SendResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SendResult{}, fmt.Errorf("commit send message: %w", err)
	}

	return SendResult{Message: message, Billing: summaryFromBillingProfile(charge.Profile)}, nil
}

func (repository *Repository) list(ctx context.Context, clientID, conversationID string) ([]Message, error) {
	if err := repository.ensureActiveClient(ctx, clientID); err != nil {
		return nil, err
	}
	if err := repository.ensureConversationOwnership(ctx, clientID, conversationID); err != nil {
		return nil, err
	}

	rows, err := repository.pool.Query(ctx, `
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
		       m.failed_at
		FROM messages AS m
		WHERE m.conversation_id = $1 AND m.client_account_id = $2
		ORDER BY m.created_at ASC`, conversationID, clientID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	messages := make([]Message, 0)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (repository *Repository) insertMessage(ctx context.Context, tx pgx.Tx, command SendCommand, recipientID, transactionID string) (Message, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO messages AS m (
			id,
			client_account_id,
			conversation_id,
			recipient_id,
			content,
			channel,
			priority,
			cost_cents,
			status,
			requested_by_user_id,
			idempotency_key,
			request_hash,
			billing_transaction_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'queued', $9, $10, $11, $12)
		RETURNING m.id::text,
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
		          m.failed_at`,
		command.MessageID,
		command.ClientID,
		command.ConversationID,
		recipientID,
		command.Content,
		command.Channel,
		command.Priority,
		command.CostCents,
		command.RequestedByUserID,
		command.IdempotencyKey,
		command.RequestHash,
		transactionID,
	)
	message, _, err := scanMessageWithOptionalHash(row, false)
	if uniqueViolation(err) {
		return Message{}, ErrAlreadyProcessed
	}
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}
	return message, nil
}

func (repository *Repository) insertDispatchJob(ctx context.Context, tx pgx.Tx, messageID string, priority domain.Priority) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO dispatch_jobs AS dj (id, message_id, priority_rank, state)
		VALUES ($1, $2, $3, 'pending')`,
		uuid.NewString(), messageID, priorityRank(priority),
	)
	if err != nil {
		return fmt.Errorf("insert dispatch job: %w", err)
	}
	return nil
}

func (repository *Repository) touchConversation(ctx context.Context, tx pgx.Tx, clientID, conversationID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE conversations AS c
		SET last_activity_at = NOW()
		WHERE c.id = $1 AND c.client_account_id = $2`, conversationID, clientID,
	)
	if err != nil {
		return fmt.Errorf("update conversation activity: %w", err)
	}
	return nil
}
