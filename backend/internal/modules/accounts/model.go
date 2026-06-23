package accounts

import "time"

type Client struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	DocumentType  string    `json:"documentType"`
	Document      string    `json:"documentId"`
	Status        string    `json:"status"`
	RequestedPlan string    `json:"requestedPlan"`
	StatusReason  *string   `json:"statusReason,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Registration struct {
	Name          string
	DocumentType  string
	Document      string
	PasswordHash  string
	RequestedPlan string
}

type Activation struct {
	PlanType                string
	InitialBalanceCents     int64
	PostpaidTotalLimitCents int64
}
