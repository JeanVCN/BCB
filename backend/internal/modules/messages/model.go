package messages

import "time"

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	Content        string     `json:"content"`
	Channel        string     `json:"channel"`
	Priority       string     `json:"priority"`
	CostCents      int64      `json:"costCents"`
	Status         string     `json:"status"`
	FailureCode    *string    `json:"failureCode,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	QueuedAt       time.Time  `json:"queuedAt"`
	ProcessingAt   *time.Time `json:"processingAt,omitempty"`
	SentAt         *time.Time `json:"sentAt,omitempty"`
	FailedAt       *time.Time `json:"failedAt,omitempty"`
}

type SendResult struct {
	Message
	Billing BillingSummary `json:"billing"`
}

type BillingSummary struct {
	PlanType                  string `json:"planType"`
	PrepaidBalanceCents       int64  `json:"prepaidBalanceCents"`
	PostpaidTotalLimitCents   int64  `json:"postpaidTotalLimitCents"`
	PostpaidConsumedCents     int64  `json:"postpaidConsumedCents"`
	PostpaidAvailableCents    int64  `json:"postpaidAvailableCents"`
	CurrentPlanAvailableCents int64  `json:"currentPlanAvailableCents"`
}

type dispatchJob struct {
	ID           string
	MessageID    string
	AttemptCount int
	Content      string
	StartedAt    time.Time
}
