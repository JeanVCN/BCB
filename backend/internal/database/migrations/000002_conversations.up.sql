CREATE TABLE recipients (
    id UUID PRIMARY KEY,
    client_account_id UUID NOT NULL REFERENCES client_accounts(id),
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (client_account_id, phone)
);

CREATE TABLE conversations (
    id UUID PRIMARY KEY,
    client_account_id UUID NOT NULL REFERENCES client_accounts(id),
    recipient_id UUID NOT NULL REFERENCES recipients(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ,
    UNIQUE (client_account_id, recipient_id)
);

CREATE INDEX conversations_client_activity_idx ON conversations (
    client_account_id,
    last_activity_at DESC NULLS LAST,
    created_at DESC
);
