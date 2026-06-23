package messages

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type Worker struct {
	repository *Repository
	logger     *slog.Logger
	id         string
}

func NewWorker(repository *Repository, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{repository: repository, logger: logger, id: "bcb-simple-worker"}
}

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
		job, found, err := worker.repository.ClaimNextJob(ctx, worker.id)
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
	case "sent":
		return worker.repository.CompleteJob(ctx, job)
	case "permanent_failure":
		return worker.repository.FailJob(ctx, job, "permanent_failure", errorCode)
	default:
		if job.AttemptCount < maxAttempts {
			return worker.repository.RetryJob(ctx, job, time.Now().UTC().Add(retryDelay(job.AttemptCount)), errorCode)
		}
		return worker.repository.FailJob(ctx, job, "transient_failure", errorCode)
	}
}

func simulateDispatch(job dispatchJob) (string, string) {
	content := strings.ToLower(job.Content)
	switch {
	case strings.Contains(content, "[fail]"):
		return "permanent_failure", "simulated_permanent_failure"
	case strings.Contains(content, "[retry]"):
		return "transient_failure", "simulated_transient_failure"
	default:
		return "sent", ""
	}
}
