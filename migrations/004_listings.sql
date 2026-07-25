CREATE TABLE IF NOT EXISTS listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO listings (id, name)
VALUES 
    ('11111111-1111-4111-8111-111111111111', 'seed-listing-1'),
    ('22222222-2222-4222-8222-222222222222', 'seed-listing-2'),
    ('33333333-3333-4333-8333-333333333333', 'seed-listing-3'),
    ('44444444-4444-4444-8444-444444444444', 'seed-listing-4'),
    ('55555555-5555-4555-8555-555555555555', 'seed-listing-5'),
    ('66666666-6666-4666-8666-666666666666', 'seed-listing-6'),
    ('77777777-7777-4777-8777-777777777777', 'seed-listing-7'),
    ('88888888-8888-4888-8888-888888888888', 'seed-listing-8'),
    ('99999999-9999-4999-8999-999999999999', 'seed-listing-9'),
    ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', 'seed-listing-10')
ON CONFLICT (id) DO NOTHING;

ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_listing_fk;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_listing_fk
    FOREIGN KEY (listing_id)
    REFERENCES listings(id)
    ON DELETE RESTRICT;