package access

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (limiter *RateLimiter) Blocked(ctx context.Context, login string) (time.Duration, bool, error) {
	duration, err := limiter.client.TTL(ctx, "auth:block:"+login).Result()
	if err != nil {
		return 0, false, fmt.Errorf("check login rate limit: %w", err)
	}
	return duration, duration > 0, nil
}

func (limiter *RateLimiter) RegisterFailure(ctx context.Context, login string) error {
	key := "auth:failures:" + login
	count, err := limiter.client.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("increment login failures: %w", err)
	}
	if count == 1 {
		limiter.client.Expire(ctx, key, time.Hour)
	}
	if count < 3 {
		return nil
	}

	shift := min(count-3, 5)
	penalty := 30 * time.Second * time.Duration(1<<shift)
	return limiter.client.Set(ctx, "auth:block:"+login, "1", penalty).Err()
}

func (limiter *RateLimiter) Reset(ctx context.Context, login string) error {
	return limiter.client.Del(ctx, "auth:failures:"+login, "auth:block:"+login).Err()
}
