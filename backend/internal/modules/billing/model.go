package billing

import "time"

type Profile struct {
	PlanType                  string    `json:"planType"`
	PrepaidBalanceCents       int64     `json:"prepaidBalanceCents"`
	PostpaidTotalLimitCents   int64     `json:"postpaidTotalLimitCents"`
	PostpaidConsumedCents     int64     `json:"postpaidConsumedCents"`
	PostpaidAvailableCents    int64     `json:"postpaidAvailableCents"`
	CurrentPlanAvailableCents int64     `json:"currentPlanAvailableCents"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

type Transaction struct {
	ID                    string    `json:"id"`
	Type                  string    `json:"type"`
	AmountCents           int64     `json:"amountCents"`
	MessageID             *string   `json:"messageId,omitempty"`
	ReversesTransactionID *string   `json:"reversesTransactionId,omitempty"`
	ActorUserID           *string   `json:"actorUserId,omitempty"`
	IdempotencyKey        string    `json:"idempotencyKey"`
	Reason                *string   `json:"reason,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
}
