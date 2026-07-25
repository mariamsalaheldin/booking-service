package booking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mariamsalaheldin/booking-service/internal/lock"
)

type stubListingStore struct {
	exists bool
	err    error
}

func (s stubListingStore) Exists(
	ctx context.Context,
	listingID uuid.UUID,
) (bool, error) {
	return s.exists, s.err
}

type stubLockManager struct {
	acquired bool
	err      error
}

func (s stubLockManager) Acquire(
	ctx context.Context,
	keys []string,
	ttl time.Duration,
) (*lock.Lock, bool, error) {
	return &lock.Lock{Keys: keys}, s.acquired, s.err
}

func (s stubLockManager) Release(ctx context.Context, lock *lock.Lock) error {
	return nil
}

func (s stubLockManager) Confirm(ctx context.Context, lock *lock.Lock, ttl time.Duration) error {
	return nil
}

func TestCreateBookingReturnsListingNotFound(t *testing.T) {
	svc := &Service{
		listings: stubListingStore{exists: false},
	}

	_, err := svc.CreateBooking(
		context.Background(),
		BookingRequest{
			ListingID:       uuid.New(),
			UserID:          uuid.New(),
			CheckIn:         "2026-08-01",
			CheckOut:        "2026-08-03",
			PaymentMethodID: "pm_test",
		},
		"",
	)

	if !errors.Is(err, ErrListingNotFound) {
		t.Fatalf("expected ErrListingNotFound, got %v", err)
	}
}

func TestCreateBookingReturnsLockUnavailable(t *testing.T) {
	svc := &Service{
		listings: stubListingStore{exists: true},
		lockManager: stubLockManager{acquired: false},
		payment: NewMockPaymentService(),
	}

	_, err := svc.CreateBooking(
		context.Background(),
		BookingRequest{
			ListingID:       uuid.New(),
			UserID:          uuid.New(),
			CheckIn:         "2026-08-01",
			CheckOut:        "2026-08-03",
			PaymentMethodID: "pm_test",
		},
		"",
	)

	if !errors.Is(err, ErrLockUnavailable) {
		t.Fatalf("expected ErrLockUnavailable, got %v", err)
	}
}

func TestCreateBookingReturnsPaymentAuthorizationError(t *testing.T) {
	svc := &Service{
		listings: stubListingStore{exists: true},
		lockManager: stubLockManager{acquired: true},
		payment: NewMockPaymentService(),
	}

	_, err := svc.CreateBooking(
		context.Background(),
		BookingRequest{
			ListingID:       uuid.New(),
			UserID:          uuid.New(),
			CheckIn:         "2026-08-01",
			CheckOut:        "2026-08-03",
			PaymentMethodID: "fail_me",
		},
		"",
	)

	if !errors.Is(err, ErrPaymentAuthorizationFailed) {
		t.Fatalf("expected ErrPaymentAuthorizationFailed, got %v", err)
	}
}
