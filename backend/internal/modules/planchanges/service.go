package planchanges

import (
	"context"
	"errors"
	"strings"

	"bcb/backend/internal/domain"
)

var (
	ErrInvalidTargetPlan      = errors.New("target plan must be prepaid or postpaid")
	ErrInvalidInitialBalance  = errors.New("initial balance must be non-negative")
	ErrInvalidPostpaidLimit   = errors.New("postpaid limit must be non-negative")
	ErrMissingRejectionReason = errors.New("rejection reason is required")
)

type Service struct {
	repository *Repository
}

func newService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) create(ctx context.Context, clientID, userID, targetPlan string) (Request, error) {
	targetPlan = strings.TrimSpace(targetPlan)
	if targetPlan != string(domain.PlanPrepaid) && targetPlan != string(domain.PlanPostpaid) {
		return Request{}, ErrInvalidTargetPlan
	}
	return service.repository.create(ctx, clientID, userID, targetPlan)
}

func (service *Service) current(ctx context.Context, clientID string) (Request, error) {
	return service.repository.current(ctx, clientID)
}

func (service *Service) cancel(ctx context.Context, clientID, userID, requestID string) error {
	return service.repository.cancel(ctx, clientID, userID, requestID)
}

func (service *Service) summary(ctx context.Context) (Summary, error) {
	return service.repository.summary(ctx)
}

func (service *Service) list(ctx context.Context, status string) ([]Request, error) {
	return service.repository.list(ctx, strings.TrimSpace(status))
}

func (service *Service) approve(ctx context.Context, actorID, requestID string, initialBalanceCents, totalLimitCents int64) error {
	if initialBalanceCents < 0 {
		return ErrInvalidInitialBalance
	}
	if totalLimitCents < 0 {
		return ErrInvalidPostpaidLimit
	}
	return service.repository.approve(ctx, actorID, requestID, initialBalanceCents, totalLimitCents)
}

func (service *Service) reject(ctx context.Context, actorID, requestID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrMissingRejectionReason
	}
	return service.repository.reject(ctx, actorID, requestID, reason)
}
