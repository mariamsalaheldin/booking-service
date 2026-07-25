CREATE TABLE IF NOT EXISTS outbox_events (

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),


    aggregate_type VARCHAR(64)
    NOT NULL,


    aggregate_id VARCHAR(64)
    NOT NULL,


    event_type VARCHAR(64)
    NOT NULL,


    payload JSONB
    NOT NULL,


    processed_at TIMESTAMPTZ NULL,


    created_at TIMESTAMPTZ
    NOT NULL
    DEFAULT NOW()
);



CREATE INDEX IF NOT EXISTS idx_outbox_unprocessed

ON outbox_events(created_at)

WHERE processed_at IS NULL;