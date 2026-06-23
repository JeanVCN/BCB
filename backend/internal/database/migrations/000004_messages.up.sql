CREATE TABLE messages (
    id UUID PRIMARY KEY,
    client_account_id UUID NOT NULL REFERENCES client_accounts(id),
    conversation_id UUID NOT NULL REFERENCES conversations(id),
    recipient_id UUID NOT NULL REFERENCES recipients(id),
    content TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (channel IN ('sms', 'whatsapp')),
    priority TEXT NOT NULL CHECK (priority IN ('normal', 'urgent')),
    cost_cents BIGINT NOT NULL CHECK (cost_cents > 0),
    status TEXT NOT NULL CHECK (status IN ('queued', 'processing', 'sent', 'failed')),
    requested_by_user_id UUID NOT NULL REFERENCES users(id),
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    billing_transaction_id UUID NOT NULL REFERENCES financial_transactions(id),
    queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processing_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (client_account_id, idempotency_key)
);

CREATE TABLE dispatch_jobs (
    id UUID PRIMARY KEY,
    message_id UUID NOT NULL UNIQUE REFERENCES messages(id),
    priority_rank INTEGER NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('pending', 'processing', 'completed', 'failed')),
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_by TEXT,
    claimed_at TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0 AND attempt_count <= 4),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE delivery_attempts (
    id UUID PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES messages(id),
    attempt_number INTEGER NOT NULL CHECK (attempt_number BETWEEN 1 AND 4),
    outcome TEXT NOT NULL CHECK (outcome IN ('sent', 'transient_failure', 'permanent_failure')),
    error_code TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    next_retry_at TIMESTAMPTZ,
    UNIQUE (message_id, attempt_number)
);

CREATE INDEX messages_conversation_created_idx ON messages(conversation_id, created_at ASC);
CREATE INDEX dispatch_jobs_pending_idx ON dispatch_jobs(state, available_at, priority_rank, created_at, id);
CREATE INDEX delivery_attempts_message_idx ON delivery_attempts(message_id, attempt_number);
