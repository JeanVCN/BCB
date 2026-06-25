package messages

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"bcb/backend/internal/domain"
)

type Worker struct {
	repository *Repository
	logger     *slog.Logger
	id         string
}

// NewWorker creates a message dispatch worker using the provided repository.
func NewWorker(repository *Repository, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{repository: repository, logger: logger, id: "bcb-simple-worker"}
}

// Run processes dispatch jobs until the context is cancelled.
func (worker *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			worker.logger.Info("message worker stopped")
			return
		case <-ticker.C:
			worker.runOnce(ctx)
		}
	}
}

func (worker *Worker) runOnce(ctx context.Context) {
	for {
		job, found, err := worker.repository.claimNextJob(ctx, worker.id)
		if err != nil {
			worker.logger.Error("failed to claim dispatch job", "error", err)
			return
		}
		if !found {
			return
		}
		if err := worker.process(ctx, job); err != nil {
			worker.logger.Error("failed to process dispatch job", "messageId", job.MessageID, "error", err)
			return
		}
	}
}

func (worker *Worker) process(ctx context.Context, job dispatchJob) error {
	time.Sleep(200 * time.Millisecond)

	outcome, errorCode := simulateDispatch(job)
	switch outcome {
	case domain.DeliveryAttemptSent:
		return worker.repository.completeJob(ctx, job)
	case domain.DeliveryAttemptPermanentFailure:
		return worker.repository.failJob(ctx, job, string(domain.DeliveryAttemptPermanentFailure), errorCode)
	default:
		if job.AttemptCount < maxAttempts {
			return worker.repository.retryJob(ctx, job, time.Now().UTC().Add(retryDelay(job.AttemptCount)), errorCode)
		}
		return worker.repository.failJob(ctx, job, string(domain.DeliveryAttemptTransientFailure), errorCode)
	}
}

func simulateDispatch(job dispatchJob) (domain.DeliveryAttemptOutcome, string) {
	content := strings.ToLower(job.Content)
	switch {
	case strings.Contains(content, "[fail]"):
		return domain.DeliveryAttemptPermanentFailure, "simulated_permanent_failure"
	case strings.Contains(content, "[retry]"):
		return domain.DeliveryAttemptTransientFailure, "simulated_transient_failure"
	default:
		return domain.DeliveryAttemptSent, ""
	}
}
