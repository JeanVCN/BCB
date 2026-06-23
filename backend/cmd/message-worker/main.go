package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"bcb/backend/internal/config"
	"bcb/backend/internal/database"
	"bcb/backend/internal/modules/billing"
	"bcb/backend/internal/modules/messages"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid worker configuration", "error", err)
		os.Exit(1)
	}

	if cfg.RunMigrations {
		if err := database.Migrate(cfg.DatabaseURL); err != nil {
			logger.Error("database migration failed", "error", err)
			os.Exit(1)
		}
	}

	pool, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	billingService := billing.NewService(billing.NewRepository(pool), billing.NewLockManager(nil))
	repository := messages.NewRepository(pool, billingService)

	logger.Info("message worker started")
	messages.NewWorker(repository, logger).Run(ctx)
}
