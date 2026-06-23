package domain

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleClient Role = "client"
)

type DocumentType string

const (
	DocumentCPF  DocumentType = "cpf"
	DocumentCNPJ DocumentType = "cnpj"
)

type PlanType string

const (
	PlanPrepaid  PlanType = "prepaid"
	PlanPostpaid PlanType = "postpaid"
)

type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

type Priority string

const (
	PriorityNormal Priority = "normal"
	PriorityUrgent Priority = "urgent"
)

type ClientStatus string

const (
	ClientStatusPending  ClientStatus = "pending"
	ClientStatusActive   ClientStatus = "active"
	ClientStatusInactive ClientStatus = "inactive"
	ClientStatusRejected ClientStatus = "rejected"
)

type MessageStatus string

const (
	MessageStatusQueued     MessageStatus = "queued"
	MessageStatusProcessing MessageStatus = "processing"
	MessageStatusSent       MessageStatus = "sent"
	MessageStatusFailed     MessageStatus = "failed"
)

type DispatchJobState string

const (
	DispatchJobPending    DispatchJobState = "pending"
	DispatchJobProcessing DispatchJobState = "processing"
	DispatchJobCompleted  DispatchJobState = "completed"
	DispatchJobFailed     DispatchJobState = "failed"
)

type FinancialTransactionType string

const (
	FinancialTransactionCredit              FinancialTransactionType = "credit"
	FinancialTransactionDebit               FinancialTransactionType = "debit"
	FinancialTransactionConsumption         FinancialTransactionType = "consumption"
	FinancialTransactionRefund              FinancialTransactionType = "refund"
	FinancialTransactionConsumptionReversal FinancialTransactionType = "consumption_reversal"
)

type DeliveryAttemptOutcome string

const (
	DeliveryAttemptSent             DeliveryAttemptOutcome = "sent"
	DeliveryAttemptTransientFailure DeliveryAttemptOutcome = "transient_failure"
	DeliveryAttemptPermanentFailure DeliveryAttemptOutcome = "permanent_failure"
)
