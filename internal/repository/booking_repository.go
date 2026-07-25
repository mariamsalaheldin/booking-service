package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mariamsalaheldin/booking-service/internal/domain"
)

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{
		db: db,
	}
}

func (r *BookingRepository) Create(
	ctx context.Context,
	tx *sql.Tx,
	b domain.Booking,
) (uuid.UUID, error) {
	var id uuid.UUID

	err := tx.QueryRowContext(
		ctx,
		`
		INSERT INTO bookings
		(
			listing_id,
			user_id,
			check_in,
			check_out,
			status
		)
		VALUES
		($1,$2,$3,$4,$5)
		RETURNING id
		`,
		b.ListingID,
		b.UserID,
		b.CheckIn,
		b.CheckOut,
		b.Status,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (r *BookingRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Booking, error) {
	result := &domain.Booking{}

	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			id,
			listing_id,
			user_id,
			check_in,
			check_out,
			status,
			created_at
		FROM bookings
		WHERE id=$1
		`,
		id,
	).Scan(
		&result.ID,
		&result.ListingID,
		&result.UserID,
		&result.CheckIn,
		&result.CheckOut,
		&result.Status,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *BookingRepository) Cancel(
	ctx context.Context,
	id uuid.UUID,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`
		UPDATE bookings
		SET status='CANCELLED'
		WHERE id=$1
		`,
		id,
	)

	return err
}

func NewBooking(
	listingID uuid.UUID,
	userID uuid.UUID,
	checkIn time.Time,
	checkOut time.Time,
) domain.Booking {
	return domain.Booking{
		ListingID: listingID,
		UserID:    userID,
		CheckIn:   checkIn,
		CheckOut:  checkOut,
		Status:    "CONFIRMED",
	}
}

type ListingRepository struct {
	db *sql.DB
}

func NewListingRepository(db *sql.DB) *ListingRepository {
	return &ListingRepository{db: db}
}

func (r *ListingRepository) Exists(
	ctx context.Context,
	listingID uuid.UUID,
) (bool, error) {
	var exists bool

	err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM listings WHERE id=$1)`,
		listingID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
