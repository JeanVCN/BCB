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

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) CreateOrGet(ctx context.Context, clientID, name, phone string) (Conversation, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Conversation{}, false, fmt.Errorf("%w: recipient name is required", ErrInvalidConversation)
	}

	normalizedPhone, err := NormalizePhone(phone)
	if err != nil {
		return Conversation{}, false, fmt.Errorf("%w: %v", ErrInvalidConversation, err)
	}

	return service.repository.CreateOrGet(ctx, clientID, name, normalizedPhone)
}

func (service *Service) List(ctx context.Context, clientID string) ([]Conversation, error) {
	return service.repository.List(ctx, clientID)
}

func (service *Service) Messages(ctx context.Context, clientID, conversationID string) ([]Message, error) {
	return service.repository.Messages(ctx, clientID, conversationID)
}
