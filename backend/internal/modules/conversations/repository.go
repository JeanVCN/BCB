package conversations

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrClientInactive       = errors.New("client is not active")
	ErrConversationNotFound = errors.New("conversation not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) CreateOrGet(ctx context.Context, clientID, name, phone string) (Conversation, bool, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return Conversation{}, false, fmt.Errorf("begin conversation creation: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureActiveClient(ctx, tx, clientID); err != nil {
		return Conversation{}, false, err
	}

	var recipient Recipient
	err = tx.QueryRow(ctx, `
		INSERT INTO recipients (id, client_account_id, name, phone)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (client_account_id, phone) DO UPDATE SET
			updated_at = recipients.updated_at
		RETURNING id, name, phone`,
		uuid.NewString(), clientID, name, phone,
	).Scan(&recipient.ID, &recipient.Name, &recipient.Phone)
	if err != nil {
		return Conversation{}, false, fmt.Errorf("upsert recipient: %w", err)
	}

	conversationID := uuid.NewString()
	var conversation Conversation
	err = tx.QueryRow(ctx, `
		INSERT INTO conversations (id, client_account_id, recipient_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (client_account_id, recipient_id) DO UPDATE SET
			recipient_id = conversations.recipient_id
		RETURNING id, last_activity_at`,
		conversationID, clientID, recipient.ID,
	).Scan(&conversation.ID, &conversation.LastActivityAt)
	if err != nil {
		return Conversation{}, false, fmt.Errorf("upsert conversation: %w", err)
	}
	conversation.Recipient = recipient

	if err := tx.Commit(ctx); err != nil {
		return Conversation{}, false, fmt.Errorf("commit conversation creation: %w", err)
	}

	return conversation, conversation.ID == conversationID, nil
}

func (repository *Repository) List(ctx context.Context, clientID string) ([]Conversation, error) {
	if err := repository.ensureActiveClient(ctx, clientID); err != nil {
		return nil, err
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT c.id, c.last_activity_at, r.id, r.name, r.phone
		FROM conversations c
		JOIN recipients r ON r.id = c.recipient_id
		WHERE c.client_account_id = $1
		ORDER BY c.last_activity_at DESC NULLS LAST, c.created_at DESC`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	conversations := make([]Conversation, 0)
	for rows.Next() {
		var conversation Conversation
		if err := rows.Scan(
			&conversation.ID,
			&conversation.LastActivityAt,
			&conversation.Recipient.ID,
			&conversation.Recipient.Name,
			&conversation.Recipient.Phone,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func (repository *Repository) Messages(ctx context.Context, clientID, conversationID string) ([]Message, error) {
	if err := repository.ensureActiveClient(ctx, clientID); err != nil {
		return nil, err
	}

	var exists bool
	err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM conversations
			WHERE id = $1 AND client_account_id = $2
		)`, conversationID, clientID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check conversation ownership: %w", err)
	}
	if !exists {
		return nil, ErrConversationNotFound
	}

	return []Message{}, nil
}

func (repository *Repository) ensureActiveClient(ctx context.Context, clientID string) error {
	return ensureActiveClient(ctx, repository.pool, clientID)
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func ensureActiveClient(ctx context.Context, query queryer, clientID string) error {
	var status string
	err := query.QueryRow(ctx, `SELECT status FROM client_accounts WHERE id = $1`, clientID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) || status != "active" {
		return ErrClientInactive
	}
	if err != nil {
		return fmt.Errorf("check client status: %w", err)
	}
	return nil
}
