package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Readiness struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

// Ready verifies that all dependencies required to serve traffic are reachable.
func (readiness Readiness) Ready(ctx context.Context) error {
	if err := readiness.Postgres.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if err := readiness.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	return nil
}
