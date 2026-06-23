CREATE TABLE client_accounts (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    document_type TEXT NOT NULL CHECK (document_type IN ('cpf', 'cnpj')),
    document TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'inactive', 'rejected')),
    requested_plan TEXT NOT NULL CHECK (requested_plan IN ('prepaid', 'postpaid')),
    status_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    role TEXT NOT NULL CHECK (role IN ('admin', 'client')),
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    client_account_id UUID UNIQUE REFERENCES client_accounts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (role = 'admin' AND client_account_id IS NULL) OR
        (role = 'client' AND client_account_id IS NOT NULL)
    )
);

CREATE TABLE billing_profiles (
    client_account_id UUID PRIMARY KEY REFERENCES client_accounts(id),
    plan_type TEXT NOT NULL CHECK (plan_type IN ('prepaid', 'postpaid')),
    prepaid_balance_cents BIGINT NOT NULL DEFAULT 0 CHECK (prepaid_balance_cents >= 0),
    postpaid_total_limit_cents BIGINT NOT NULL DEFAULT 0 CHECK (postpaid_total_limit_cents >= 0),
    postpaid_consumed_cents BIGINT NOT NULL DEFAULT 0 CHECK (postpaid_consumed_cents >= 0),
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (postpaid_consumed_cents <= postpaid_total_limit_cents)
);

CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    reason TEXT,
    previous_values JSONB,
    new_values JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX client_accounts_status_idx ON client_accounts(status, created_at);
CREATE INDEX audit_events_target_idx ON audit_events(target_type, target_id, created_at DESC);
