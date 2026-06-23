package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bcb/backend/internal/config"
	"bcb/backend/internal/database"
	"bcb/backend/internal/httpserver"
	"bcb/backend/internal/modules"

	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid application configuration", "error", err)
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

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error("redis connection failed", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr: ":" + cfg.HTTPPort,
		Handler: httpserver.NewRouter(httpserver.Dependencies{
			Modules: modules.New(modules.Dependencies{
				Config:   cfg,
				Postgres: pool,
				Redis:    redisClient,
			}),
			Readiness: database.Readiness{Postgres: pool, Redis: redisClient},
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		logger.Info("http server started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownSignal.Done()
	logger.Info("shutdown signal received")

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("http server stopped")
}
