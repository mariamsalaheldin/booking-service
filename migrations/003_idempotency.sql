CREATE TABLE IF NOT EXISTS idempotency_keys (

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),


    key VARCHAR(255)
    NOT NULL
    UNIQUE,


    response JSONB
    NOT NULL,


    status_code INT
    NOT NULL,


    created_at TIMESTAMPTZ
    NOT NULL
    DEFAULT NOW(),


    expires_at TIMESTAMPTZ
    NOT NULL
);



CREATE INDEX IF NOT EXISTS idx_idempotency_expiry

ON idempotency_keys(expires_at);