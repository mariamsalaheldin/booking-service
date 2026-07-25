package booking

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mariamsalaheldin/booking-service/internal/lock"
	"github.com/mariamsalaheldin/booking-service/internal/repository"
)

func TestCreateBookingE2E(t *testing.T) {
	db, err := repository.NewPostgres("postgres://postgres:postgres@localhost:5432/booking_db?sslmode=disable")
	if err != nil {
		t.Skipf("postgres not available for integration test: %v", err)
	}
	defer db.Close()

	for _, migration := range []string{"001_bookings.sql", "002_outbox.sql", "003_idempotency.sql", "004_listings.sql"} {
		if err := applyMigration(t, db.DB, migration); err != nil {
			t.Fatalf("apply migration %s: %v", migration, err)
		}
	}

	if _, err := db.DB.Exec(`TRUNCATE TABLE bookings, outbox_events, idempotency_keys, listings RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	listingID := uuid.New()
	checkIn := time.Now().AddDate(0, 0, 1).UTC()
	checkOut := checkIn.AddDate(0, 0, 2)

	if _, err := db.DB.Exec(`INSERT INTO listings (id, name) VALUES ($1, $2)`, listingID, "seed-listing"); err != nil {
		t.Fatalf("seed listing: %v", err)
	}

	lockManager := lock.NewManager("localhost:6379")
	defer lockManager.Close()

	bookingRepo := repository.NewBookingRepository(db.DB)
	listingRepo := repository.NewListingRepository(db.DB)
	outboxRepo := repository.NewOutboxRepository(db.DB)
	payment := NewMockPaymentService()

	svc, err := NewService(db, bookingRepo, outboxRepo, lockManager, payment, listingRepo)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	response, err := svc.CreateBooking(ctx, BookingRequest{
		ListingID:       listingID,
		UserID:          uuid.New(),
		CheckIn:         checkIn.Format("2006-01-02"),
		CheckOut:        checkOut.Format("2006-01-02"),
		PaymentMethodID: "pm_test",
	}, "e2e-key")
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}

	if response == nil || response.BookingID == "" {
		t.Fatalf("expected non-empty booking response, got %#v", response)
	}

	var bookingCount int
	if err := db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&bookingCount); err != nil {
		t.Fatalf("count bookings: %v", err)
	}
	if bookingCount != 1 {
		t.Fatalf("expected 1 booking after successful create, got %d", bookingCount)
	}

	var outboxCount int
	if err := db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events`).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox events: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected 1 outbox event after successful create, got %d", outboxCount)
	}

	var idempotencyCount int
	if err := db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM idempotency_keys WHERE key=$1`, "e2e-key").Scan(&idempotencyCount); err != nil {
		t.Fatalf("count idempotency rows: %v", err)
	}
	if idempotencyCount != 1 {
		t.Fatalf("expected 1 idempotency row after successful create, got %d", idempotencyCount)
	}
}

func applyMigration(t *testing.T, db *sql.DB, migration string) error {
	t.Helper()

	path := filepath.Join("..", "..", "migrations", migration)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", migration, err)
	}

	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("execute migration %s: %w", migration, err)
	}

	return nil
}
