package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mariamsalaheldin/booking-service/internal/booking"
)

type stubService struct {
	err            error
	capturedKey    string
	called         bool
}

func (s *stubService) CreateBooking(
	ctx context.Context,
	req booking.BookingRequest,
	idempotencyKey string,
) (*booking.BookingResponse, error) {
	s.called = true
	s.capturedKey = idempotencyKey
	return nil, s.err
}

func TestHandlerRejectsInvalidRequestBody(t *testing.T) {
	h := NewHandler(nil)

	req := httptest.NewRequest(
		"POST",
		"/api/v1/bookings",
		strings.NewReader(`{not-json}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	h.CreateBooking(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandlerMapsListingNotFoundToNotFound(t *testing.T) {
	h := &Handler{service: &stubService{err: booking.ErrListingNotFound}}

	req := httptest.NewRequest(
		"POST",
		"/api/v1/bookings",
		strings.NewReader(`{"listing_id":"11111111-1111-4111-8111-111111111111","user_id":"22222222-2222-4222-8222-222222222222","check_in":"2026-08-01","check_out":"2026-08-03","payment_method_id":"pm_test"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "listing-key")

	rr := httptest.NewRecorder()

	h.CreateBooking(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandlerUsesIdempotencyKeyHeader(t *testing.T) {
	service := &stubService{}
	h := &Handler{service: service}

	req := httptest.NewRequest(
		"POST",
		"/api/v1/bookings",
		strings.NewReader(`{"listing_id":"11111111-1111-4111-8111-111111111111","user_id":"22222222-2222-4222-8222-222222222222","check_in":"2026-08-01","check_out":"2026-08-03","payment_method_id":"pm_test"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-key")

	rr := httptest.NewRecorder()

	h.CreateBooking(rr, req)

	if !service.called {
		t.Fatal("expected service to be called")
	}
	if service.capturedKey != "test-key" {
		t.Fatalf("expected idempotency key test-key, got %q", service.capturedKey)
	}
}

func TestHandlerRejectsMissingIdempotencyKey(t *testing.T) {
	h := &Handler{service: &stubService{}}

	req := httptest.NewRequest(
		"POST",
		"/api/v1/bookings",
		strings.NewReader(`{"listing_id":"11111111-1111-4111-8111-111111111111","user_id":"22222222-2222-4222-8222-222222222222","check_in":"2026-08-01","check_out":"2026-08-03","payment_method_id":"pm_test"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	h.CreateBooking(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
