package messages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bcb/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) claimNextJob(ctx context.Context, workerID string) (dispatchJob, bool, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return dispatchJob{}, false, fmt.Errorf("begin claim job: %w", err)
	}
	defer tx.Rollback(ctx)

	var job dispatchJob
	err = tx.QueryRow(ctx, `
		SELECT dj.id::text,
		       m.id::text,
		       dj.attempt_count + 1,
		       m.content,
		       NOW()
		FROM dispatch_jobs AS dj
		JOIN messages AS m ON m.id = dj.message_id
		WHERE dj.state = $1 AND dj.available_at <= NOW()
		ORDER BY dj.priority_rank ASC, dj.created_at ASC, dj.id ASC
		FOR UPDATE OF dj SKIP LOCKED
		LIMIT 1`,
		domain.DispatchJobPending,
	).Scan(&job.ID, &job.MessageID, &job.AttemptCount, &job.Content, &job.StartedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return dispatchJob{}, false, nil
	}
	if err != nil {
		return dispatchJob{}, false, fmt.Errorf("select dispatch job: %w", err)
	}

	if err := repository.markJobClaimed(ctx, tx, job, workerID); err != nil {
		return dispatchJob{}, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return dispatchJob{}, false, fmt.Errorf("commit job claim: %w", err)
	}
	return job, true, nil
}

func (repository *Repository) completeJob(ctx context.Context, job dispatchJob) error {
	return repository.finishJob(ctx, job, jobFinish{
		AttemptOutcome: domain.DeliveryAttemptSent,
		MessageStatus:  domain.MessageStatusSent,
		JobState:       domain.DispatchJobCompleted,
	})
}

func (repository *Repository) retryJob(ctx context.Context, job dispatchJob, nextRetryAt time.Time, errorCode string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin retry job: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := repository.insertDeliveryAttempt(ctx, tx, deliveryAttempt{
		Job:         job,
		Outcome:     domain.DeliveryAttemptTransientFailure,
		ErrorCode:   errorCode,
		NextRetryAt: &nextRetryAt,
	}); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE dispatch_jobs AS dj
		SET state = $2,
		    available_at = $3,
		    claimed_by = NULL,
		    claimed_at = NULL,
		    updated_at = NOW()
		WHERE dj.id = $1`, job.ID, domain.DispatchJobPending, nextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("schedule retry: %w", err)
	}
	return tx.Commit(ctx)
}

func (repository *Repository) failJob(ctx context.Context, job dispatchJob, outcome, errorCode string) error {
	return repository.finishJob(ctx, job, jobFinish{
		AttemptOutcome: domain.DeliveryAttemptOutcome(outcome),
		MessageStatus:  domain.MessageStatusFailed,
		JobState:       domain.DispatchJobFailed,
		ErrorCode:      errorCode,
	})
}

type jobFinish struct {
	AttemptOutcome domain.DeliveryAttemptOutcome
	MessageStatus  domain.MessageStatus
	JobState       domain.DispatchJobState
	ErrorCode      string
}

func (repository *Repository) finishJob(ctx context.Context, job dispatchJob, finish jobFinish) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin finish job: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := repository.insertDeliveryAttempt(ctx, tx, deliveryAttempt{
		Job:       job,
		Outcome:   finish.AttemptOutcome,
		ErrorCode: finish.ErrorCode,
	}); err != nil {
		return err
	}

	if err := repository.markMessageTerminal(ctx, tx, job.MessageID, finish.MessageStatus, finish.ErrorCode); err != nil {
		return err
	}
	if finish.MessageStatus == domain.MessageStatusFailed {
		if err := repository.billing.ReverseMessageCharge(ctx, tx, job.MessageID); err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE dispatch_jobs AS dj
		SET state = $2,
		    updated_at = NOW()
		WHERE dj.id = $1`, job.ID, finish.JobState,
	)
	if err != nil {
		return fmt.Errorf("finish dispatch job: %w", err)
	}
	return tx.Commit(ctx)
}

func (repository *Repository) markJobClaimed(ctx context.Context, tx pgx.Tx, job dispatchJob, workerID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE dispatch_jobs AS dj
		SET state = $2,
		    attempt_count = attempt_count + 1,
		    claimed_by = $3,
		    claimed_at = NOW(),
		    updated_at = NOW()
		WHERE dj.id = $1`, job.ID, domain.DispatchJobProcessing, workerID)
	if err != nil {
		return fmt.Errorf("claim dispatch job: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages AS m
		SET status = $2,
		    processing_at = COALESCE(m.processing_at, NOW()),
		    updated_at = NOW()
		WHERE m.id = $1 AND m.status <> $3`, job.MessageID, domain.MessageStatusProcessing, domain.MessageStatusSent)
	if err != nil {
		return fmt.Errorf("mark message processing: %w", err)
	}
	return nil
}

func (repository *Repository) markMessageTerminal(ctx context.Context, tx pgx.Tx, messageID string, status domain.MessageStatus, errorCode string) error {
	if status == domain.MessageStatusSent {
		_, err := tx.Exec(ctx, `
			UPDATE messages AS m
			SET status = $2,
			    sent_at = NOW(),
			    updated_at = NOW()
			WHERE m.id = $1 AND m.status <> $2`, messageID, status)
		if err != nil {
			return fmt.Errorf("mark message sent: %w", err)
		}
		return nil
	}

	_, err := tx.Exec(ctx, `
		UPDATE messages AS m
		SET status = $2,
		    failed_at = NOW(),
		    failure_code = $3,
		    updated_at = NOW()
		WHERE m.id = $1 AND m.status <> $2`, messageID, status, errorCode)
	if err != nil {
		return fmt.Errorf("mark message failed: %w", err)
	}
	return nil
}

type deliveryAttempt struct {
	Job         dispatchJob
	Outcome     domain.DeliveryAttemptOutcome
	ErrorCode   string
	NextRetryAt *time.Time
}

func (repository *Repository) insertDeliveryAttempt(ctx context.Context, tx pgx.Tx, attempt deliveryAttempt) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO delivery_attempts AS da (
			id,
			message_id,
			attempt_number,
			outcome,
			error_code,
			started_at,
			finished_at,
			next_retry_at
		) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NOW(), $7)
		ON CONFLICT (message_id, attempt_number) DO NOTHING`,
		uuid.NewString(),
		attempt.Job.MessageID,
		attempt.Job.AttemptCount,
		attempt.Outcome,
		attempt.ErrorCode,
		attempt.Job.StartedAt,
		attempt.NextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("insert delivery attempt: %w", err)
	}
	return nil
}
