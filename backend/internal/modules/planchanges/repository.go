package planchanges

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
	ErrClientNotActive       = errors.New("client is not active")
	ErrRequestNotFound       = errors.New("plan change request not found")
	ErrSamePlan              = errors.New("target plan is already active")
	ErrFinancialStateBlocked = errors.New("financial state does not allow plan change")
	ErrPendingRequestExists  = errors.New("pending plan change request already exists")
	ErrInvalidTransition     = errors.New("invalid plan change request transition")
	ErrApprovalPayload       = errors.New("approval payload is invalid for target plan")
)

type Repository struct {
	pool *pgxpool.Pool
}

func newRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) create(ctx context.Context, clientID, userID, targetPlan string) (Request, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return Request{}, fmt.Errorf("begin plan change request: %w", err)
	}
	defer tx.Rollback(ctx)

	profile, err := repository.lockProfile(ctx, tx, clientID)
	if err != nil {
		return Request{}, err
	}
	if profile.planType == targetPlan {
		return Request{}, ErrSamePlan
	}
	requestID := uuid.NewString()
	request, err := repository.insertRequest(ctx, tx, requestID, clientID, userID, profile.planType, targetPlan)
	if uniqueViolation(err) {
		return Request{}, ErrPendingRequestExists
	}
	if err != nil {
		return Request{}, err
	}

	if err := insertAudit(ctx, tx, userID, "plan_change.requested", "plan_change_request", requestID, "", map[string]any{"planType": profile.planType}, map[string]any{"status": request.Status, "toPlan": targetPlan}); err != nil {
		return Request{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Request{}, fmt.Errorf("commit plan change request: %w", err)
	}
	return request, nil
}

func (repository *Repository) current(ctx context.Context, clientID string) (Request, error) {
	row := repository.pool.QueryRow(ctx, `
		SELECT pcr.id::text,
		       pcr.client_account_id::text,
		       ca.name,
		       pcr.from_plan,
		       pcr.to_plan,
		       pcr.status,
		       pcr.requested_by_user_id::text,
		       pcr.cancelled_by_user_id::text,
		       pcr.decided_by_user_id::text,
		       pcr.rejection_reason,
		       pcr.created_at,
		       pcr.cancelled_at,
		       pcr.decided_at
		FROM plan_change_requests AS pcr
		INNER JOIN client_accounts AS ca ON ca.id = pcr.client_account_id
		WHERE pcr.client_account_id = $1
		ORDER BY pcr.created_at DESC
		LIMIT 1`, clientID)
	request, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrRequestNotFound
	}
	if err != nil {
		return Request{}, fmt.Errorf("read current plan change request: %w", err)
	}
	return request, nil
}

func (repository *Repository) cancel(ctx context.Context, clientID, userID, requestID string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin cancel plan change request: %w", err)
	}
	defer tx.Rollback(ctx)

	request, err := repository.lockRequest(ctx, tx, requestID)
	if err != nil {
		return err
	}
	if request.ClientID != clientID {
		return ErrRequestNotFound
	}
	if request.Status != string(domain.PlanChangeRequestPending) {
		return ErrInvalidTransition
	}

	_, err = tx.Exec(ctx, `
		UPDATE plan_change_requests
		SET status = $2,
		    cancelled_by_user_id = $3,
		    cancelled_at = NOW()
		WHERE id = $1`,
		requestID, domain.PlanChangeRequestCancelled, userID,
	)
	if err != nil {
		return fmt.Errorf("cancel plan change request: %w", err)
	}

	if err := insertAudit(ctx, tx, userID, "plan_change.cancelled", "plan_change_request", requestID, "", map[string]any{"status": request.Status}, map[string]any{"status": domain.PlanChangeRequestCancelled}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Repository) summary(ctx context.Context) (Summary, error) {
	var summary Summary
	err := repository.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE ca.status = 'pending') AS pending_client_activations,
			(SELECT COUNT(*) FROM plan_change_requests WHERE status = 'pending') AS pending_plan_changes
		FROM client_accounts AS ca`,
	).Scan(&summary.PendingClientActivations, &summary.PendingPlanChanges)
	if err != nil {
		return Summary{}, fmt.Errorf("read admin summary: %w", err)
	}
	return summary, nil
}

func (repository *Repository) list(ctx context.Context, status string) ([]Request, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT pcr.id::text,
		       pcr.client_account_id::text,
		       ca.name,
		       pcr.from_plan,
		       pcr.to_plan,
		       pcr.status,
		       pcr.requested_by_user_id::text,
		       pcr.cancelled_by_user_id::text,
		       pcr.decided_by_user_id::text,
		       pcr.rejection_reason,
		       pcr.created_at,
		       pcr.cancelled_at,
		       pcr.decided_at
		FROM plan_change_requests AS pcr
		INNER JOIN client_accounts AS ca ON ca.id = pcr.client_account_id
		WHERE ($1 = '' OR pcr.status = $1)
		ORDER BY pcr.created_at ASC`, status)
	if err != nil {
		return nil, fmt.Errorf("list plan change requests: %w", err)
	}
	defer rows.Close()

	requests := make([]Request, 0)
	for rows.Next() {
		request, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

func (repository *Repository) approve(ctx context.Context, actorID, requestID string, initialBalanceCents, totalLimitCents int64) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin approve plan change request: %w", err)
	}
	defer tx.Rollback(ctx)

	request, err := repository.lockRequest(ctx, tx, requestID)
	if err != nil {
		return err
	}
	if request.Status != string(domain.PlanChangeRequestPending) {
		return ErrInvalidTransition
	}

	profile, err := repository.lockProfile(ctx, tx, request.ClientID)
	if err != nil {
		return err
	}
	if profile.planType != request.FromPlan || !profile.canApprovePlanChange() {
		return ErrFinancialStateBlocked
	}

	nextPrepaidBalance, nextPostpaidLimit, err := approvalBillingState(request.ToPlan, initialBalanceCents, totalLimitCents)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE billing_profiles
		SET plan_type = $2,
		    prepaid_balance_cents = $3,
		    postpaid_total_limit_cents = $4,
		    postpaid_consumed_cents = 0,
		    version = version + 1,
		    updated_at = NOW()
		WHERE client_account_id = $1`,
		request.ClientID, request.ToPlan, nextPrepaidBalance, nextPostpaidLimit,
	)
	if err != nil {
		return fmt.Errorf("update billing profile for plan change: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE plan_change_requests
		SET status = $2,
		    decided_by_user_id = $3,
		    decided_at = NOW()
		WHERE id = $1`,
		requestID, domain.PlanChangeRequestApproved, actorID,
	)
	if err != nil {
		return fmt.Errorf("approve plan change request: %w", err)
	}

	if err := insertAudit(ctx, tx, actorID, "plan_change.approved", "plan_change_request", requestID, "", map[string]any{"status": request.Status, "planType": request.FromPlan}, map[string]any{"status": domain.PlanChangeRequestApproved, "planType": request.ToPlan}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Repository) reject(ctx context.Context, actorID, requestID, reason string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reject plan change request: %w", err)
	}
	defer tx.Rollback(ctx)

	request, err := repository.lockRequest(ctx, tx, requestID)
	if err != nil {
		return err
	}
	if request.Status != string(domain.PlanChangeRequestPending) {
		return ErrInvalidTransition
	}

	_, err = tx.Exec(ctx, `
		UPDATE plan_change_requests
		SET status = $2,
		    decided_by_user_id = $3,
		    rejection_reason = $4,
		    decided_at = NOW()
		WHERE id = $1`,
		requestID, domain.PlanChangeRequestRejected, actorID, reason,
	)
	if err != nil {
		return fmt.Errorf("reject plan change request: %w", err)
	}

	if err := insertAudit(ctx, tx, actorID, "plan_change.rejected", "plan_change_request", requestID, reason, map[string]any{"status": request.Status}, map[string]any{"status": domain.PlanChangeRequestRejected}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type billingSnapshot struct {
	planType         string
	prepaidBalance   int64
	postpaidConsumed int64
}

func (snapshot billingSnapshot) canApprovePlanChange() bool {
	switch snapshot.planType {
	case string(domain.PlanPrepaid):
		return snapshot.prepaidBalance == 0
	case string(domain.PlanPostpaid):
		return snapshot.postpaidConsumed == 0
	default:
		return false
	}
}

func (repository *Repository) lockProfile(ctx context.Context, tx pgx.Tx, clientID string) (billingSnapshot, error) {
	var snapshot billingSnapshot
	var status string
	err := tx.QueryRow(ctx, `
		SELECT ca.status,
		       bp.plan_type,
		       bp.prepaid_balance_cents,
		       bp.postpaid_consumed_cents
		FROM client_accounts AS ca
		INNER JOIN billing_profiles AS bp ON bp.client_account_id = ca.id
		WHERE ca.id = $1
		FOR UPDATE OF ca, bp`,
		clientID,
	).Scan(&status, &snapshot.planType, &snapshot.prepaidBalance, &snapshot.postpaidConsumed)
	if errors.Is(err, pgx.ErrNoRows) {
		return billingSnapshot{}, ErrClientNotActive
	}
	if err != nil {
		return billingSnapshot{}, fmt.Errorf("lock billing profile: %w", err)
	}
	if status != string(domain.ClientStatusActive) {
		return billingSnapshot{}, ErrClientNotActive
	}
	return snapshot, nil
}

func (repository *Repository) insertRequest(ctx context.Context, tx pgx.Tx, requestID, clientID, userID, fromPlan, toPlan string) (Request, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO plan_change_requests (
			id,
			client_account_id,
			from_plan,
			to_plan,
			status,
			requested_by_user_id
		) VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id::text,
		          client_account_id::text,
		          '',
		          from_plan,
		          to_plan,
		          status,
		          requested_by_user_id::text,
		          cancelled_by_user_id::text,
		          decided_by_user_id::text,
		          rejection_reason,
		          created_at,
		          cancelled_at,
		          decided_at`,
		requestID, clientID, fromPlan, toPlan, userID,
	)
	request, err := scanRequest(row)
	if err != nil {
		return Request{}, fmt.Errorf("insert plan change request: %w", err)
	}
	return request, nil
}

func (repository *Repository) lockRequest(ctx context.Context, tx pgx.Tx, requestID string) (Request, error) {
	row := tx.QueryRow(ctx, `
		SELECT pcr.id::text,
		       pcr.client_account_id::text,
		       ca.name,
		       pcr.from_plan,
		       pcr.to_plan,
		       pcr.status,
		       pcr.requested_by_user_id::text,
		       pcr.cancelled_by_user_id::text,
		       pcr.decided_by_user_id::text,
		       pcr.rejection_reason,
		       pcr.created_at,
		       pcr.cancelled_at,
		       pcr.decided_at
		FROM plan_change_requests AS pcr
		INNER JOIN client_accounts AS ca ON ca.id = pcr.client_account_id
		WHERE pcr.id = $1
		FOR UPDATE OF pcr`,
		requestID,
	)
	request, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrRequestNotFound
	}
	if err != nil {
		return Request{}, fmt.Errorf("lock plan change request: %w", err)
	}
	return request, nil
}

type requestScanner interface {
	Scan(dest ...any) error
}

func scanRequest(scanner requestScanner) (Request, error) {
	var request Request
	err := scanner.Scan(
		&request.ID,
		&request.ClientID,
		&request.ClientName,
		&request.FromPlan,
		&request.ToPlan,
		&request.Status,
		&request.RequestedByUserID,
		&request.CancelledByUserID,
		&request.DecidedByUserID,
		&request.RejectionReason,
		&request.CreatedAt,
		&request.CancelledAt,
		&request.DecidedAt,
	)
	return request, err
}

func approvalBillingState(targetPlan string, initialBalanceCents, totalLimitCents int64) (int64, int64, error) {
	switch targetPlan {
	case string(domain.PlanPrepaid):
		if totalLimitCents != 0 {
			return 0, 0, ErrApprovalPayload
		}
		return initialBalanceCents, 0, nil
	case string(domain.PlanPostpaid):
		if initialBalanceCents != 0 {
			return 0, 0, ErrApprovalPayload
		}
		return 0, totalLimitCents, nil
	default:
		return 0, 0, ErrInvalidTargetPlan
	}
}

func insertAudit(ctx context.Context, tx pgx.Tx, actorID, action, targetType, targetID, reason string, previous, next any) error {
	previousJSON, _ := json.Marshal(previous)
	nextJSON, _ := json.Marshal(next)
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id, actor_user_id, action, target_type, target_id, reason, previous_values, new_values)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7::jsonb, $8::jsonb)`,
		uuid.NewString(), actorID, action, targetType, targetID, reason, nullableJSON(previousJSON), nullableJSON(nextJSON),
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
