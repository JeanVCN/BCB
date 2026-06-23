CREATE TABLE idempotency_records (
    id UUID PRIMARY KEY,
    client_account_id UUID NOT NULL REFERENCES client_accounts(id),
    operation TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (client_account_id, operation, idempotency_key)
);

CREATE TABLE financial_transactions (
    id UUID PRIMARY KEY,
    client_account_id UUID NOT NULL REFERENCES client_accounts(id),
    type TEXT NOT NULL CHECK (type IN ('credit', 'debit', 'consumption', 'refund', 'consumption_reversal')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    message_id UUID,
    reverses_transaction_id UUID REFERENCES financial_transactions(id),
    actor_user_id UUID REFERENCES users(id),
    idempotency_key TEXT NOT NULL,
    idempotency_record_id UUID REFERENCES idempotency_records(id),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX financial_transactions_client_created_idx ON financial_transactions(client_account_id, created_at DESC);
