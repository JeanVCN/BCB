package messages

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"bcb/backend/internal/modules/billing"
)

const (
	normalCostCents = int64(25)
	urgentCostCents = int64(50)
	maxAttempts     = 4
)

type Service struct {
	repository *Repository
	locks      *billing.LockManager
}

func NewService(repository *Repository, locks *billing.LockManager) *Service {
	return &Service{repository: repository, locks: locks}
}

func (service *Service) Send(ctx context.Context, clientID, requestedByUserID, conversationID, content, channel, priority, idempotencyKey string) (SendResult, error) {
	content = strings.TrimSpace(content)
	channel = strings.TrimSpace(strings.ToLower(channel))
	priority = strings.TrimSpace(strings.ToLower(priority))
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if content == "" || !validChannel(channel) || !validPriority(priority) {
		return SendResult{}, ErrInvalidMessage
	}
	if idempotencyKey == "" {
		return SendResult{}, billing.ErrMissingIdempotencyKey
	}

	command := SendCommand{
		ClientID:          clientID,
		ConversationID:    conversationID,
		RequestedByUserID: requestedByUserID,
		Content:           content,
		Channel:           channel,
		Priority:          priority,
		CostCents:         cost(priority),
		IdempotencyKey:    idempotencyKey,
		RequestHash:       RequestHash(content, channel, priority),
	}

	var result SendResult
	err := service.locks.WithClientLock(ctx, clientID, func() error {
		var err error
		result, err = service.repository.Send(ctx, command)
		if errors.Is(err, ErrAlreadyProcessed) {
			return nil
		}
		return err
	})
	return result, err
}

func (service *Service) List(ctx context.Context, clientID, conversationID string) ([]Message, error) {
	return service.repository.List(ctx, clientID, conversationID)
}

func cost(priority string) int64 {
	if priority == "urgent" {
		return urgentCostCents
	}
	return normalCostCents
}

func validChannel(channel string) bool {
	return channel == "sms" || channel == "whatsapp"
}

func validPriority(priority string) bool {
	return priority == "normal" || priority == "urgent"
}

func retryDelay(attempt int) time.Duration {
	base := 4 * time.Second
	switch attempt {
	case 1:
		base = time.Second
	case 2:
		base = 2 * time.Second
	}
	return base + retryJitter()
}

func retryJitter() time.Duration {
	value, err := rand.Int(rand.Reader, big.NewInt(250))
	if err != nil {
		return 0
	}
	return time.Duration(value.Int64()) * time.Millisecond
}
