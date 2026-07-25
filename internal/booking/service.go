package booking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mariamsalaheldin/booking-service/internal/lock"
	"github.com/mariamsalaheldin/booking-service/internal/repository"
)

var (
	ErrListingNotFound           = errors.New("listing not found")
	ErrInvalidDates              = errors.New("checkout must be after checkin")
	ErrLockUnavailable           = errors.New("dates already reserved")
	ErrPaymentAuthorizationFailed = errors.New("payment authorization failed")
	ErrPaymentCaptureFailed      = errors.New("payment capture failed")
	ErrBookingCreationFailed     = errors.New("booking creation failed")
	ErrOutboxCreationFailed      = errors.New("outbox creation failed")
	ErrDatabase                  = errors.New("database error")
	ErrLockConfirmationFailed    = errors.New("lock confirmation failed")
)

type listingStore interface {
	Exists(ctx context.Context, listingID uuid.UUID) (bool, error)
}

type bookingRepository interface {
	Create(ctx context.Context, tx *sql.Tx, b Booking) (uuid.UUID, error)
}

type outboxRepository interface {
	Create(ctx context.Context, tx *sql.Tx, event OutboxEvent) error
}

type lockManager interface {
	Acquire(ctx context.Context, keys []string, ttl time.Duration) (*lock.Lock, bool, error)
	Release(ctx context.Context, lock *lock.Lock) error
	Confirm(ctx context.Context, lock *lock.Lock, ttl time.Duration) error
}

type Service struct {
	db *repository.Postgres

	bookings bookingRepository
	outbox   outboxRepository

	lockManager lockManager

	idempotencyStore IdempotencyStore

	listings listingStore

	payment PaymentService
}

func NewService(
	db *repository.Postgres,
	bookings bookingRepository,
	outbox outboxRepository,
	lockManager lockManager,
	payment PaymentService,
	listings *repository.ListingRepository,
) (*Service, error) {
	idempotencyStore, err := NewPostgresIdempotencyStore(db)
	if err != nil {
		return nil, fmt.Errorf("create idempotency store: %w", err)
	}

	return &Service{
		db: db,

		bookings: bookings,
		outbox:   outbox,

		lockManager: lockManager,

		idempotencyStore: idempotencyStore,

		listings: listings,

		payment: payment,
	}, nil
}

func (s *Service) CreateBooking(
	ctx context.Context,
	req BookingRequest,
	idempotencyKey string,
) (response *BookingResponse, err error) {
	if s.idempotencyStore != nil && idempotencyKey != "" {
		if response, ok := s.idempotencyStore.Get(ctx, idempotencyKey); ok {
			return response, nil
		}
	}

	checkIn, err := time.Parse(
		"2006-01-02",
		req.CheckIn,
	)
	if err != nil {
		return nil, err
	}

	checkOut, err := time.Parse(
		"2006-01-02",
		req.CheckOut,
	)
	if err != nil {
		return nil, err
	}

	if !checkOut.After(checkIn) {
		return nil, ErrInvalidDates
	}

	if s.listings != nil {
		exists, err := s.listings.Exists(ctx, req.ListingID)
		if err != nil {
			return nil, fmt.Errorf("check listing: %w", err)
		}
		if !exists {
			return nil, ErrListingNotFound
		}
	}

	dates := generateDates(
		req.ListingID,
		checkIn,
		checkOut,
	)

	lockObj, acquired, err := s.lockManager.Acquire(
		ctx,
		dates,
		30*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	if !acquired {
		return nil, ErrLockUnavailable
	}

	if manager, ok := s.lockManager.(*lock.Manager); ok {
		heartbeat := lock.NewHeartbeat(
			manager,
			lockObj,
			30*time.Second,
		)

		heartbeat.Start(ctx)

		defer heartbeat.Stop()
	}

	auth, err := s.payment.Authorize(
		req.PaymentMethodID,
		250,
	)
	if err != nil {
		_ = s.lockManager.Release(
			ctx,
			lockObj,
		)

		return nil, fmt.Errorf("%w: %v", ErrPaymentAuthorizationFailed, err)
	}

	tx, err := s.db.DB.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		_ = s.lockManager.Release(
			ctx,
			lockObj,
		)
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	newBooking := NewBooking(
		req.ListingID,
		req.UserID,
		checkIn,
		checkOut,
	)

	bookingID, err := s.bookings.Create(
		ctx,
		tx,
		newBooking,
	)
	if err != nil {
		_ = s.payment.Void(
			auth.AuthorizationID,
		)
		_ = s.lockManager.Release(
			ctx,
			lockObj,
		)

		return nil, fmt.Errorf("%w: %v", ErrBookingCreationFailed, err)
	}

	payload, err := json.Marshal(
		EventPayload{
			BookingID: bookingID.String(),

			ListingID: req.ListingID.String(),

			UserID: req.UserID.String(),

			CheckIn: req.CheckIn,

			CheckOut: req.CheckOut,

			Timestamp: time.Now().UTC(),
		},
	)
	if err != nil {
		return nil, err
	}

	event := OutboxEvent{
		ID: uuid.New(),

		AggregateType: "BOOKING",

		AggregateID: bookingID.String(),

		EventType: "BOOKING_CONFIRMED",

		Payload: payload,
	}

	err = s.outbox.Create(
		ctx,
		tx,
		event,
	)
	if err != nil {
		_ = s.payment.Void(
			auth.AuthorizationID,
		)
		_ = s.lockManager.Release(
			ctx,
			lockObj,
		)

		return nil, fmt.Errorf("%w: %v", ErrOutboxCreationFailed, err)
	}

	err = s.payment.Capture(
		auth.AuthorizationID,
	)
	if err != nil {
		_ = s.payment.Void(
			auth.AuthorizationID,
		)
		_ = tx.Rollback()
		return nil, fmt.Errorf("%w: %v", ErrPaymentCaptureFailed, err)
	}

	if err := tx.Commit(); err != nil {
		_ = s.payment.Void(
			auth.AuthorizationID,
		)
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}

	err = s.lockManager.Confirm(
		ctx,
		lockObj,
		30*24*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLockConfirmationFailed, err)
	}

	response = &BookingResponse{
		BookingID: bookingID.String(),

		Status: "CONFIRMED",
	}

	if s.idempotencyStore != nil && idempotencyKey != "" {
		if err := s.idempotencyStore.Set(ctx, idempotencyKey, *response, 24*time.Hour); err != nil {
			return nil, fmt.Errorf("store idempotency response: %w", err)
		}
	}

	return response, nil
}

func generateDates(
	listingID uuid.UUID,
	start time.Time,
	end time.Time,
) []string {
	var dates []string

	for current := start; current.Before(end); current = current.AddDate(0, 0, 1) {
		dates = append(
			dates,
			"lock:listing:"+
				listingID.String()+
				":"+
				current.Format("2006-01-02"),
		)
	}

	return dates
}
