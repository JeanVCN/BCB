package billing

import (
	"context"
	"errors"
	"fmt"

	"bcb/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) profile(ctx context.Context, clientID string) (Profile, error) {
	row := repository.pool.QueryRow(ctx, `
		SELECT bp.plan_type,
		       bp.prepaid_balance_cents,
		       bp.postpaid_total_limit_cents,
		       bp.postpaid_consumed_cents,
		       bp.updated_at,
		       ca.status
		FROM billing_profiles AS bp
		JOIN client_accounts AS ca ON ca.id = bp.client_account_id
		WHERE bp.client_account_id = $1`, clientID)
	profile, status, err := scanProfileWithStatus(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("read billing profile: %w", err)
	}
	if status != string(domain.ClientStatusActive) {
		return Profile{}, ErrClientNotActive
	}
	return profile, nil
}

func (repository *Repository) addCredit(ctx context.Context, actorID, clientID string, amountCents int64, reason, idempotencyKey, requestHash string) error {
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
	if profile.PlanType != string(domain.PlanPrepaid) {
		return ErrPlanMismatch
	}

	_, err = tx.Exec(ctx, `
		UPDATE billing_profiles AS bp
		SET prepaid_balance_cents = bp.prepaid_balance_cents + $2,
		    version = bp.version + 1,
		    updated_at = NOW()
		WHERE bp.client_account_id = $1`, clientID, amountCents)
	if err != nil {
		return fmt.Errorf("increase prepaid balance: %w", err)
	}

	if err := repository.insertFinancialTransaction(ctx, tx, financialTransactionInput{
		ID:                  newID(),
		ClientID:            clientID,
		Type:                domain.FinancialTransactionCredit,
		AmountCents:         amountCents,
		ActorUserID:         actorID,
		IdempotencyKey:      idempotencyKey,
		IdempotencyRecordID: idempotencyRecordID,
		Reason:              reason,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (repository *Repository) adjustPostpaidLimit(ctx context.Context, actorID, clientID string, totalLimitCents int64, reason, idempotencyKey, requestHash string) error {
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
	if profile.PlanType != string(domain.PlanPostpaid) {
		return ErrPlanMismatch
	}
	if totalLimitCents < profile.PostpaidConsumedCents {
		return ErrLimitBelowConsumed
	}

	_, err = tx.Exec(ctx, `
		UPDATE billing_profiles AS bp
		SET postpaid_total_limit_cents = $2,
		    version = bp.version + 1,
		    updated_at = NOW()
		WHERE bp.client_account_id = $1`, clientID, totalLimitCents)
	if err != nil {
		return fmt.Errorf("update postpaid limit: %w", err)
	}

	if err := insertAudit(ctx, tx, actorID, "billing.postpaid_limit_adjusted", clientID, reason, map[string]any{"totalLimitCents": profile.PostpaidTotalLimitCents}, map[string]any{"totalLimitCents": totalLimitCents}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Repository) zeroCurrentBalance(ctx context.Context, actorID, clientID, reason, idempotencyKey, requestHash string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin zero current balance: %w", err)
	}
	defer tx.Rollback(ctx)

	idempotencyRecordID, err := repository.registerIdempotency(ctx, tx, clientID, "admin.zero_current_balance", idempotencyKey, requestHash)
	if err != nil {
		return err
	}

	profile, err := repository.lockProfile(ctx, tx, clientID)
	if err != nil {
		return err
	}

	switch profile.PlanType {
	case string(domain.PlanPrepaid):
		if err := repository.zeroPrepaidBalance(ctx, tx, actorID, clientID, reason, idempotencyKey, idempotencyRecordID, profile); err != nil {
			return err
		}
	case string(domain.PlanPostpaid):
		if err := repository.zeroPostpaidConsumption(ctx, tx, actorID, clientID, reason, idempotencyKey, idempotencyRecordID, profile); err != nil {
			return err
		}
	default:
		return ErrPlanMismatch
	}

	return tx.Commit(ctx)
}

func (repository *Repository) zeroPrepaidBalance(ctx context.Context, tx pgx.Tx, actorID, clientID, reason, idempotencyKey, idempotencyRecordID string, profile Profile) error {
	_, err := tx.Exec(ctx, `
		UPDATE billing_profiles AS bp
		SET prepaid_balance_cents = 0,
		    version = bp.version + 1,
		    updated_at = NOW()
		WHERE bp.client_account_id = $1`, clientID)
	if err != nil {
		return fmt.Errorf("zero prepaid balance: %w", err)
	}
	if profile.PrepaidBalanceCents > 0 {
		if err := repository.insertFinancialTransaction(ctx, tx, financialTransactionInput{
			ID:                  newID(),
			ClientID:            clientID,
			Type:                domain.FinancialTransactionDebit,
			AmountCents:         profile.PrepaidBalanceCents,
			ActorUserID:         actorID,
			IdempotencyKey:      idempotencyKey,
			IdempotencyRecordID: idempotencyRecordID,
			Reason:              reasonOrDefault(reason, "Zeramento administrativo de saldo pré-pago"),
		}); err != nil {
			return err
		}
	}
	return insertAudit(ctx, tx, actorID, "billing.prepaid_balance_zeroed", clientID, reason, map[string]any{"prepaidBalanceCents": profile.PrepaidBalanceCents}, map[string]any{"prepaidBalanceCents": 0})
}

func (repository *Repository) zeroPostpaidConsumption(ctx context.Context, tx pgx.Tx, actorID, clientID, reason, idempotencyKey, idempotencyRecordID string, profile Profile) error {
	_, err := tx.Exec(ctx, `
		UPDATE billing_profiles AS bp
		SET postpaid_consumed_cents = 0,
		    version = bp.version + 1,
		    updated_at = NOW()
		WHERE bp.client_account_id = $1`, clientID)
	if err != nil {
		return fmt.Errorf("zero postpaid consumption: %w", err)
	}
	if profile.PostpaidConsumedCents > 0 {
		if err := repository.insertFinancialTransaction(ctx, tx, financialTransactionInput{
			ID:                  newID(),
			ClientID:            clientID,
			Type:                domain.FinancialTransactionConsumptionReversal,
			AmountCents:         profile.PostpaidConsumedCents,
			ActorUserID:         actorID,
			IdempotencyKey:      idempotencyKey,
			IdempotencyRecordID: idempotencyRecordID,
			Reason:              reasonOrDefault(reason, "Zeramento administrativo de consumo pós-pago"),
		}); err != nil {
			return err
		}
	}
	return insertAudit(ctx, tx, actorID, "billing.postpaid_consumption_zeroed", clientID, reason, map[string]any{"postpaidConsumedCents": profile.PostpaidConsumedCents}, map[string]any{"postpaidConsumedCents": 0})
}

func reasonOrDefault(reason, fallback string) string {
	if reason != "" {
		return reason
	}
	return fallback
}

func (repository *Repository) lockProfile(ctx context.Context, tx pgx.Tx, clientID string) (Profile, error) {
	row := tx.QueryRow(ctx, `
		SELECT bp.plan_type,
		       bp.prepaid_balance_cents,
		       bp.postpaid_total_limit_cents,
		       bp.postpaid_consumed_cents,
		       bp.updated_at
		FROM billing_profiles AS bp
		WHERE bp.client_account_id = $1
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

func (repository *Repository) lockActiveProfile(ctx context.Context, tx pgx.Tx, clientID string) (Profile, error) {
	row := tx.QueryRow(ctx, `
		SELECT bp.plan_type,
		       bp.prepaid_balance_cents,
		       bp.postpaid_total_limit_cents,
		       bp.postpaid_consumed_cents,
		       bp.updated_at,
		       ca.status
		FROM billing_profiles AS bp
		JOIN client_accounts AS ca ON ca.id = bp.client_account_id
		WHERE bp.client_account_id = $1
		FOR UPDATE OF bp, ca`, clientID)
	profile, status, err := scanProfileWithStatus(row)
	if errors.Is(err, pgx.ErrNoRows) || status != string(domain.ClientStatusActive) {
		return Profile{}, ErrClientNotActive
	}
	if err != nil {
		return Profile{}, fmt.Errorf("lock active billing profile: %w", err)
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
	fillProfileAvailability(&profile)
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
	fillProfileAvailability(&profile)
	return profile, status, nil
}

func fillProfileAvailability(profile *Profile) {
	profile.PostpaidAvailableCents = profile.PostpaidTotalLimitCents - profile.PostpaidConsumedCents
	if profile.PlanType == string(domain.PlanPrepaid) {
		profile.CurrentPlanAvailableCents = profile.PrepaidBalanceCents
	} else {
		profile.CurrentPlanAvailableCents = profile.PostpaidAvailableCents
	}
}
