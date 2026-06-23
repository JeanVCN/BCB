package messages

import (
	"context"
	"errors"
	"strings"

	"bcb/backend/internal/modules/billing"
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
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	parsedChannel, validChannel := parseChannel(channel)
	parsedPriority, validPriority := parsePriority(priority)
	if content == "" || !validChannel || !validPriority {
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
		Channel:           parsedChannel,
		Priority:          parsedPriority,
		CostCents:         messageCost(parsedPriority),
		IdempotencyKey:    idempotencyKey,
		RequestHash:       requestHash(content, parsedChannel, parsedPriority),
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
