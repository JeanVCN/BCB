package messages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bcb/backend/internal/modules/billing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrClientInactive       = errors.New("client is not active")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrInsufficientBalance  = errors.New("insufficient prepaid balance")
	ErrLimitExceeded        = errors.New("postpaid limit exceeded")
	ErrIdempotencyConflict  = errors.New("idempotency key was used with a different request")
	ErrInvalidMessage       = errors.New("invalid message")
	ErrAlreadyProcessed     = errors.New("message already processed")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) Send(ctx context.Context, command SendCommand) (SendResult, error) {
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
		summary, err := repository.billingSummary(ctx, tx, command.ClientID)
		if err != nil {
			return SendResult{}, err
		}
		return SendResult{Message: existing.message, Billing: summary}, nil
	}

	recipientID, err := repository.lockConversation(ctx, tx, command.ClientID, command.ConversationID)
	if err != nil {
		return SendResult{}, err
	}

	profile, err := repository.lockBillingProfile(ctx, tx, command.ClientID)
	if err != nil {
		return SendResult{}, err
	}

	messageID := uuid.NewString()
	transactionID := uuid.NewString()
	transactionType := "debit"
	if profile.PlanType == "prepaid" {
		if profile.PrepaidBalanceCents < command.CostCents {
			return SendResult{}, ErrInsufficientBalance
		}
		profile.PrepaidBalanceCents -= command.CostCents
		_, err = tx.Exec(ctx, `
			UPDATE billing_profiles
			SET prepaid_balance_cents = prepaid_balance_cents - $2,
			    version = version + 1,
			    updated_at = NOW()
			WHERE client_account_id = $1`, command.ClientID, command.CostCents)
	} else {
		if profile.PostpaidConsumedCents+command.CostCents > profile.PostpaidTotalLimitCents {
			return SendResult{}, ErrLimitExceeded
		}
		transactionType = "consumption"
		profile.PostpaidConsumedCents += command.CostCents
		_, err = tx.Exec(ctx, `
			UPDATE billing_profiles
			SET postpaid_consumed_cents = postpaid_consumed_cents + $2,
			    version = version + 1,
			    updated_at = NOW()
			WHERE client_account_id = $1`, command.ClientID, command.CostCents)
	}
	if err != nil {
		return SendResult{}, fmt.Errorf("apply message charge: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO financial_transactions (
			id, client_account_id, type, amount_cents, message_id, actor_user_id,
			idempotency_key, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'Cobrança de mensagem')`,
		transactionID, command.ClientID, transactionType, command.CostCents, messageID, command.RequestedByUserID, command.IdempotencyKey,
	)
	if err != nil {
		return SendResult{}, fmt.Errorf("insert message charge transaction: %w", err)
	}

	var message Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (
			id, client_account_id, conversation_id, recipient_id, content, channel, priority,
			cost_cents, status, requested_by_user_id, idempotency_key, request_hash,
			billing_transaction_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'queued', $9, $10, $11, $12)
		RETURNING id::text, conversation_id::text, content, channel, priority, cost_cents,
		          status, failure_code, created_at, queued_at, processing_at, sent_at, failed_at`,
		messageID, command.ClientID, command.ConversationID, recipientID, command.Content, command.Channel,
		command.Priority, command.CostCents, command.RequestedByUserID, command.IdempotencyKey,
		command.RequestHash, transactionID,
	).Scan(
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
	)
	if uniqueViolation(err) {
		return SendResult{}, ErrAlreadyProcessed
	}
	if err != nil {
		return SendResult{}, fmt.Errorf("insert message: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO dispatch_jobs (id, message_id, priority_rank, state)
		VALUES ($1, $2, $3, 'pending')`,
		uuid.NewString(), messageID, priorityRank(command.Priority),
	)
	if err != nil {
		return SendResult{}, fmt.Errorf("insert dispatch job: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE conversations
		SET last_activity_at = NOW()
		WHERE id = $1 AND client_account_id = $2`, command.ConversationID, command.ClientID,
	)
	if err != nil {
		return SendResult{}, fmt.Errorf("update conversation activity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return SendResult{}, fmt.Errorf("commit send message: %w", err)
	}

	return SendResult{Message: message, Billing: summaryFromProfile(profile)}, nil
}

func (repository *Repository) List(ctx context.Context, clientID, conversationID string) ([]Message, error) {
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

	rows, err := repository.pool.Query(ctx, `
		SELECT id::text, conversation_id::text, content, channel, priority, cost_cents,
		       status, failure_code, created_at, queued_at, processing_at, sent_at, failed_at
		FROM messages
		WHERE conversation_id = $1 AND client_account_id = $2
		ORDER BY created_at ASC`, conversationID, clientID)
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

func (repository *Repository) ClaimNextJob(ctx context.Context, workerID string) (dispatchJob, bool, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return dispatchJob{}, false, fmt.Errorf("begin claim job: %w", err)
	}
	defer tx.Rollback(ctx)

	var job dispatchJob
	err = tx.QueryRow(ctx, `
		SELECT dispatch_jobs.id::text, messages.id::text, dispatch_jobs.attempt_count + 1,
		       messages.content, NOW()
		FROM dispatch_jobs
		JOIN messages ON messages.id = dispatch_jobs.message_id
		WHERE dispatch_jobs.state = 'pending' AND dispatch_jobs.available_at <= NOW()
		ORDER BY dispatch_jobs.priority_rank ASC, dispatch_jobs.created_at ASC, dispatch_jobs.id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1`).Scan(&job.ID, &job.MessageID, &job.AttemptCount, &job.Content, &job.StartedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return dispatchJob{}, false, nil
	}
	if err != nil {
		return dispatchJob{}, false, fmt.Errorf("select dispatch job: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE dispatch_jobs
		SET state = 'processing',
		    attempt_count = attempt_count + 1,
		    claimed_by = $2,
		    claimed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1`, job.ID, workerID)
	if err != nil {
		return dispatchJob{}, false, fmt.Errorf("claim dispatch job: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages
		SET status = 'processing',
		    processing_at = COALESCE(processing_at, NOW()),
		    updated_at = NOW()
		WHERE id = $1 AND status <> 'sent'`, job.MessageID)
	if err != nil {
		return dispatchJob{}, false, fmt.Errorf("mark message processing: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dispatchJob{}, false, fmt.Errorf("commit job claim: %w", err)
	}
	return job, true, nil
}

func (repository *Repository) CompleteJob(ctx context.Context, job dispatchJob) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin complete job: %w", err)
	}
	defer tx.Rollback(ctx)

	finishedAt := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO delivery_attempts (id, message_id, attempt_number, outcome, started_at, finished_at)
		VALUES ($1, $2, $3, 'sent', $4, $5)
		ON CONFLICT (message_id, attempt_number) DO NOTHING`,
		uuid.NewString(), job.MessageID, job.AttemptCount, job.StartedAt, finishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert sent attempt: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages
		SET status = 'sent',
		    sent_at = $2,
		    updated_at = NOW()
		WHERE id = $1 AND status <> 'sent'`, job.MessageID, finishedAt,
	)
	if err != nil {
		return fmt.Errorf("mark message sent: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE dispatch_jobs
		SET state = 'completed',
		    updated_at = NOW()
		WHERE id = $1`, job.ID)
	if err != nil {
		return fmt.Errorf("complete dispatch job: %w", err)
	}
	return tx.Commit(ctx)
}

func (repository *Repository) RetryJob(ctx context.Context, job dispatchJob, nextRetryAt time.Time, errorCode string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin retry job: %w", err)
	}
	defer tx.Rollback(ctx)

	finishedAt := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO delivery_attempts (
			id, message_id, attempt_number, outcome, error_code,
			started_at, finished_at, next_retry_at
		) VALUES ($1, $2, $3, 'transient_failure', $4, $5, $6, $7)
		ON CONFLICT (message_id, attempt_number) DO NOTHING`,
		uuid.NewString(), job.MessageID, job.AttemptCount, errorCode, job.StartedAt, finishedAt, nextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("insert retry attempt: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE dispatch_jobs
		SET state = 'pending',
		    available_at = $2,
		    claimed_by = NULL,
		    claimed_at = NULL,
		    updated_at = NOW()
		WHERE id = $1`, job.ID, nextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("schedule retry: %w", err)
	}
	return tx.Commit(ctx)
}

func (repository *Repository) FailJob(ctx context.Context, job dispatchJob, outcome, errorCode string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin fail job: %w", err)
	}
	defer tx.Rollback(ctx)

	finishedAt := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO delivery_attempts (id, message_id, attempt_number, outcome, error_code, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (message_id, attempt_number) DO NOTHING`,
		uuid.NewString(), job.MessageID, job.AttemptCount, outcome, errorCode, job.StartedAt, finishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert failed attempt: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages
		SET status = 'failed',
		    failed_at = $2,
		    failure_code = $3,
		    updated_at = NOW()
		WHERE id = $1 AND status <> 'failed'`, job.MessageID, finishedAt, errorCode,
	)
	if err != nil {
		return fmt.Errorf("mark message failed: %w", err)
	}

	if err := repository.reverseCharge(ctx, tx, job.MessageID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE dispatch_jobs
		SET state = 'failed',
		    updated_at = NOW()
		WHERE id = $1`, job.ID)
	if err != nil {
		return fmt.Errorf("fail dispatch job: %w", err)
	}
	return tx.Commit(ctx)
}

func (repository *Repository) reverseCharge(ctx context.Context, tx pgx.Tx, messageID string) error {
	var transactionID, clientID, transactionType string
	var amountCents int64
	err := tx.QueryRow(ctx, `
		SELECT id::text, client_account_id::text, type, amount_cents
		FROM financial_transactions
		WHERE message_id = $1 AND type IN ('debit', 'consumption')
		FOR UPDATE`, messageID).Scan(&transactionID, &clientID, &transactionType, &amountCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read original charge: %w", err)
	}

	var alreadyReversed bool
	err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM financial_transactions WHERE reverses_transaction_id = $1)`, transactionID).Scan(&alreadyReversed)
	if err != nil {
		return fmt.Errorf("check charge reversal: %w", err)
	}
	if alreadyReversed {
		return nil
	}

	reversalType := "refund"
	if transactionType == "debit" {
		_, err = tx.Exec(ctx, `
			UPDATE billing_profiles
			SET prepaid_balance_cents = prepaid_balance_cents + $2,
			    version = version + 1,
			    updated_at = NOW()
			WHERE client_account_id = $1`, clientID, amountCents)
	} else {
		reversalType = "consumption_reversal"
		_, err = tx.Exec(ctx, `
			UPDATE billing_profiles
			SET postpaid_consumed_cents = postpaid_consumed_cents - $2,
			    version = version + 1,
			    updated_at = NOW()
			WHERE client_account_id = $1 AND postpaid_consumed_cents >= $2`, clientID, amountCents)
	}
	if err != nil {
		return fmt.Errorf("apply charge reversal: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO financial_transactions (
			id, client_account_id, type, amount_cents, message_id,
			reverses_transaction_id, idempotency_key, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'Estorno por falha definitiva')`,
		uuid.NewString(), clientID, reversalType, amountCents, messageID, transactionID, "refund:"+messageID,
	)
	if err != nil {
		return fmt.Errorf("insert charge reversal: %w", err)
	}
	return nil
}

type existingMessage struct {
	message     Message
	requestHash string
}

func (repository *Repository) findExistingMessage(ctx context.Context, tx pgx.Tx, clientID, idempotencyKey string) (existingMessage, bool, error) {
	row := tx.QueryRow(ctx, `
		SELECT id::text, conversation_id::text, content, channel, priority, cost_cents,
		       status, failure_code, created_at, queued_at, processing_at, sent_at, failed_at,
		       request_hash
		FROM messages
		WHERE client_account_id = $1 AND idempotency_key = $2`, clientID, idempotencyKey)
	message, hash, err := scanMessageWithHash(row)
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
		SELECT recipient_id::text
		FROM conversations
		WHERE id = $1 AND client_account_id = $2
		FOR UPDATE`, conversationID, clientID).Scan(&recipientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrConversationNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lock conversation: %w", err)
	}
	return recipientID, nil
}

type billingProfile struct {
	PlanType                string
	PrepaidBalanceCents     int64
	PostpaidTotalLimitCents int64
	PostpaidConsumedCents   int64
}

func (repository *Repository) lockBillingProfile(ctx context.Context, tx pgx.Tx, clientID string) (billingProfile, error) {
	var profile billingProfile
	var status string
	err := tx.QueryRow(ctx, `
		SELECT billing_profiles.plan_type,
		       billing_profiles.prepaid_balance_cents,
		       billing_profiles.postpaid_total_limit_cents,
		       billing_profiles.postpaid_consumed_cents,
		       client_accounts.status
		FROM billing_profiles
		JOIN client_accounts ON client_accounts.id = billing_profiles.client_account_id
		WHERE billing_profiles.client_account_id = $1
		FOR UPDATE OF billing_profiles, client_accounts`,
		clientID,
	).Scan(
		&profile.PlanType,
		&profile.PrepaidBalanceCents,
		&profile.PostpaidTotalLimitCents,
		&profile.PostpaidConsumedCents,
		&status,
	)
	if errors.Is(err, pgx.ErrNoRows) || status != "active" {
		return billingProfile{}, ErrClientInactive
	}
	if err != nil {
		return billingProfile{}, fmt.Errorf("lock billing profile: %w", err)
	}
	return profile, nil
}

func (repository *Repository) billingSummary(ctx context.Context, tx pgx.Tx, clientID string) (BillingSummary, error) {
	profile, err := repository.lockBillingProfile(ctx, tx, clientID)
	if err != nil {
		return BillingSummary{}, err
	}
	return summaryFromProfile(profile), nil
}

func (repository *Repository) ensureActiveClient(ctx context.Context, clientID string) error {
	var status string
	err := repository.pool.QueryRow(ctx, `SELECT status FROM client_accounts WHERE id = $1`, clientID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) || status != "active" {
		return ErrClientInactive
	}
	if err != nil {
		return fmt.Errorf("check client status: %w", err)
	}
	return nil
}

func scanMessage(rows pgx.Rows) (Message, error) {
	var message Message
	if err := rows.Scan(
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
	); err != nil {
		return Message{}, fmt.Errorf("scan message: %w", err)
	}
	return message, nil
}

func scanMessageWithHash(row pgx.Row) (Message, string, error) {
	var message Message
	var hash string
	err := row.Scan(
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
		&hash,
	)
	return message, hash, err
}

func summaryFromProfile(profile billingProfile) BillingSummary {
	available := profile.PostpaidTotalLimitCents - profile.PostpaidConsumedCents
	summary := BillingSummary{
		PlanType:                profile.PlanType,
		PrepaidBalanceCents:     profile.PrepaidBalanceCents,
		PostpaidTotalLimitCents: profile.PostpaidTotalLimitCents,
		PostpaidConsumedCents:   profile.PostpaidConsumedCents,
		PostpaidAvailableCents:  available,
	}
	if profile.PlanType == "prepaid" {
		summary.CurrentPlanAvailableCents = profile.PrepaidBalanceCents
	} else {
		summary.CurrentPlanAvailableCents = available
	}
	return summary
}

func priorityRank(priority string) int {
	if priority == "urgent" {
		return 0
	}
	return 1
}

func uniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

type SendCommand struct {
	ClientID          string
	ConversationID    string
	RequestedByUserID string
	Content           string
	Channel           string
	Priority          string
	CostCents         int64
	IdempotencyKey    string
	RequestHash       string
}

func RequestHash(content, channel, priority string) string {
	return billing.RequestHash("message.send", struct {
		Content  string `json:"content"`
		Channel  string `json:"channel"`
		Priority string `json:"priority"`
	}{Content: content, Channel: channel, Priority: priority})
}
