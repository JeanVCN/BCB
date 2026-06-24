CREATE TABLE plan_change_requests (
    id UUID PRIMARY KEY,
    client_account_id UUID NOT NULL REFERENCES client_accounts(id),
    from_plan TEXT NOT NULL CHECK (from_plan IN ('prepaid', 'postpaid')),
    to_plan TEXT NOT NULL CHECK (to_plan IN ('prepaid', 'postpaid')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    requested_by_user_id UUID NOT NULL REFERENCES users(id),
    cancelled_by_user_id UUID REFERENCES users(id),
    decided_by_user_id UUID REFERENCES users(id),
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at TIMESTAMPTZ,
    decided_at TIMESTAMPTZ,
    CHECK (from_plan <> to_plan),
    CHECK (
        (status = 'rejected' AND rejection_reason IS NOT NULL) OR
        (status <> 'rejected' AND rejection_reason IS NULL)
    )
);

CREATE UNIQUE INDEX plan_change_requests_one_pending_idx
    ON plan_change_requests(client_account_id)
    WHERE status = 'pending';

CREATE INDEX plan_change_requests_status_created_idx
    ON plan_change_requests(status, created_at ASC);

CREATE INDEX plan_change_requests_client_created_idx
    ON plan_change_requests(client_account_id, created_at DESC);
