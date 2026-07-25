CREATE EXTENSION IF NOT EXISTS btree_gist;


CREATE TABLE IF NOT EXISTS bookings (

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    listing_id UUID NOT NULL,

    user_id UUID NOT NULL,


    check_in DATE NOT NULL,

    check_out DATE NOT NULL,


    booking_period DATERANGE
    GENERATED ALWAYS AS
    (
        daterange(
            check_in,
            check_out,
            '[)'
        )
    )
    STORED,


    status VARCHAR(32)
    NOT NULL
    DEFAULT 'CONFIRMED',


    created_at TIMESTAMPTZ
    NOT NULL
    DEFAULT NOW(),



    CONSTRAINT no_booking_overlap

    EXCLUDE USING gist
    (

        listing_id WITH =,

        booking_period WITH &&

    )
);