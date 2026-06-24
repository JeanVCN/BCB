package planchanges

import "time"

type Request struct {
	ID                string     `json:"id"`
	ClientID          string     `json:"clientId"`
	ClientName        string     `json:"clientName,omitempty"`
	FromPlan          string     `json:"fromPlan"`
	ToPlan            string     `json:"toPlan"`
	Status            string     `json:"status"`
	RequestedByUserID string     `json:"requestedByUserId"`
	CancelledByUserID *string    `json:"cancelledByUserId,omitempty"`
	DecidedByUserID   *string    `json:"decidedByUserId,omitempty"`
	RejectionReason   *string    `json:"rejectionReason,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	CancelledAt       *time.Time `json:"cancelledAt,omitempty"`
	DecidedAt         *time.Time `json:"decidedAt,omitempty"`
}

type Summary struct {
	PendingClientActivations int64 `json:"pendingClientActivations"`
	PendingPlanChanges       int64 `json:"pendingPlanChanges"`
}
