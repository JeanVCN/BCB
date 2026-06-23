package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bcb/backend/internal/domain"
	"bcb/backend/internal/modules/access"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Register(ctx context.Context, name, documentType, document, password, plan string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("name is required")
	}
	if plan != string(domain.PlanPrepaid) && plan != string(domain.PlanPostpaid) {
		return "", errors.New("requestedPlan must be prepaid or postpaid")
	}
	normalizedDocument, err := NormalizeDocument(document, documentType)
	if err != nil {
		return "", err
	}
	if err := access.ValidatePassword(password); err != nil {
		return "", fmt.Errorf("password: %w", err)
	}
	hash, err := access.HashPassword(password)
	if err != nil {
		return "", err
	}
	return service.repository.Register(ctx, Registration{
		Name: name, DocumentType: documentType, Document: normalizedDocument,
		PasswordHash: hash, RequestedPlan: plan,
	})
}
