package conversations

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidConversation = errors.New("invalid conversation")

type Service struct {
	repository *Repository
}

func newService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) createOrGet(ctx context.Context, clientID, name, phone string) (Conversation, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Conversation{}, false, fmt.Errorf("%w: recipient name is required", ErrInvalidConversation)
	}

	normalizedPhone, err := normalizePhone(phone)
	if err != nil {
		return Conversation{}, false, fmt.Errorf("%w: %v", ErrInvalidConversation, err)
	}

	return service.repository.createOrGet(ctx, clientID, name, normalizedPhone)
}

func (service *Service) list(ctx context.Context, clientID string) ([]Conversation, error) {
	return service.repository.list(ctx, clientID)
}

func (service *Service) messages(ctx context.Context, clientID, conversationID string) ([]Message, error) {
	return service.repository.messages(ctx, clientID, conversationID)
}
