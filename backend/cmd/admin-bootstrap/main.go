package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"bcb/backend/internal/config"
	"bcb/backend/internal/database"
	"bcb/backend/internal/modules/access"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid application configuration", "error", err)
		os.Exit(1)
	}
	login := strings.TrimSpace(strings.ToLower(os.Getenv("ADMIN_LOGIN")))
	password := os.Getenv("ADMIN_PASSWORD")
	if login == "" {
		logger.Error("ADMIN_LOGIN is required")
		os.Exit(1)
	}
	if err := access.ValidatePassword(password); err != nil {
		logger.Error("ADMIN_PASSWORD is invalid", "error", err)
		os.Exit(1)
	}
	hash, err := access.HashPassword(password)
	if err != nil {
		logger.Error("password hashing failed", "error", err)
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
	if err := access.NewRepository(pool).BootstrapAdmin(context.Background(), login, hash); err != nil {
		logger.Error("administrator bootstrap failed", "error", err)
		os.Exit(1)
	}
	logger.Info("administrator bootstrap completed", "login", login)
}
