package accounts

import (
	"context"
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
	ErrConflict = errors.New("resource conflict")
	ErrNotFound = errors.New("resource not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func newRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) register(ctx context.Context, registration Registration) (string, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin registration: %w", err)
	}
	defer tx.Rollback(ctx)

	clientID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO client_accounts (id, name, document_type, document, status, requested_plan)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		clientID, registration.Name, registration.DocumentType, registration.Document, domain.ClientStatusPending, registration.RequestedPlan,
	)
	if uniqueViolation(err) {
		return "", ErrConflict
	}
	if err != nil {
		return "", fmt.Errorf("insert client: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, role, login, password_hash, enabled, client_account_id)
		VALUES ($1, 'client', $2, $3, TRUE, $4)`,
		uuid.NewString(), registration.Document, registration.PasswordHash, clientID,
	)
	if err != nil {
		return "", fmt.Errorf("insert client user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit registration: %w", err)
	}
	return clientID, nil
}

func (repository *Repository) clients(ctx context.Context, status string) ([]Client, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, name, document_type, document, status, requested_plan, status_reason, created_at
		FROM client_accounts
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at ASC`, status)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()

	clients := make([]Client, 0)
	for rows.Next() {
		var client Client
		if err := rows.Scan(&client.ID, &client.Name, &client.DocumentType, &client.Document, &client.Status, &client.RequestedPlan, &client.StatusReason, &client.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func (repository *Repository) activate(ctx context.Context, actorID, clientID string, activation Activation) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin activation: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus, requestedPlan string
	err = tx.QueryRow(ctx, `SELECT status, requested_plan FROM client_accounts WHERE id = $1 FOR UPDATE`, clientID).Scan(&currentStatus, &requestedPlan)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock client: %w", err)
	}
	if activation.PlanType != requestedPlan || currentStatus == string(domain.ClientStatusActive) {
		return ErrConflict
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO billing_profiles (
			client_account_id, plan_type, prepaid_balance_cents, postpaid_total_limit_cents
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (client_account_id) DO UPDATE SET
			plan_type = EXCLUDED.plan_type,
			prepaid_balance_cents = EXCLUDED.prepaid_balance_cents,
			postpaid_total_limit_cents = EXCLUDED.postpaid_total_limit_cents,
			postpaid_consumed_cents = 0,
			version = billing_profiles.version + 1,
			updated_at = NOW()`,
		clientID, activation.PlanType, activation.InitialBalanceCents, activation.PostpaidTotalLimitCents,
	)
	if err != nil {
		return fmt.Errorf("set billing profile: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE client_accounts SET status = $2, status_reason = NULL, updated_at = NOW() WHERE id = $1`, clientID, domain.ClientStatusActive)
	if err != nil {
		return fmt.Errorf("activate client: %w", err)
	}

	if err := insertAudit(ctx, tx, actorID, "client.activated", clientID, "", map[string]any{"status": currentStatus}, map[string]any{"status": domain.ClientStatusActive, "planType": activation.PlanType}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Repository) changeStatus(ctx context.Context, actorID, clientID, targetStatus, reason string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var previous string
	err = tx.QueryRow(ctx, `SELECT status FROM client_accounts WHERE id = $1 FOR UPDATE`, clientID).Scan(&previous)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock client status: %w", err)
	}
	if (targetStatus == string(domain.ClientStatusRejected) && previous != string(domain.ClientStatusPending)) || (targetStatus == string(domain.ClientStatusInactive) && previous != string(domain.ClientStatusActive)) {
		return ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE client_accounts SET status = $2, status_reason = $3, updated_at = NOW() WHERE id = $1`, clientID, targetStatus, reason)
	if err != nil {
		return fmt.Errorf("change client status: %w", err)
	}
	if err := insertAudit(ctx, tx, actorID, "client."+targetStatus, clientID, reason, map[string]any{"status": previous}, map[string]any{"status": targetStatus}); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
