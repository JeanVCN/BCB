package billing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrLockUnavailable = errors.New("billing lock unavailable")

type LockManager struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewLockManager creates a Redis-backed lock manager for billing mutations.
func NewLockManager(redis *redis.Client) *LockManager {
	return &LockManager{redis: redis, ttl: 10 * time.Second}
}

// WithClientLock runs an action while holding the billing lock for a client account.
func (manager *LockManager) WithClientLock(ctx context.Context, clientID string, action func() error) error {
	if manager.redis == nil {
		return ErrLockUnavailable
	}
	token, err := randomToken()
	if err != nil {
		return fmt.Errorf("create lock token: %w", err)
	}
	key := "billing:client:" + clientID
	acquired, err := manager.redis.SetNX(ctx, key, token, manager.ttl).Result()
	if err != nil || !acquired {
		return ErrLockUnavailable
	}
	defer manager.unlock(ctx, key, token)

	return action()
}

func (manager *LockManager) unlock(ctx context.Context, key, token string) {
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0`
	_ = manager.redis.Eval(ctx, script, []string{key}, token).Err()
}

func randomToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
