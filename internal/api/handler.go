package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mariamsalaheldin/booking-service/internal/booking"
)

type bookingService interface {
	CreateBooking(
		ctx context.Context,
		req booking.BookingRequest,
		idempotencyKey string,
	) (*booking.BookingResponse, error)
}

type Handler struct {
	service bookingService
}

func NewHandler(
	service bookingService,
) *Handler {

	return &Handler{
		service: service,
	}
}


func (h *Handler) CreateBooking(
	w http.ResponseWriter,
	r *http.Request,
) {

	var req booking.BookingRequest


	err := json.NewDecoder(
		r.Body,
	).Decode(&req)


	if err != nil {

		Error(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)

		return
	}



	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		Error(
			w,
			http.StatusBadRequest,
			"MISSING_IDEMPOTENCY_KEY",
			"idempotency key is required",
		)
		return
	}

	result, err := h.service.CreateBooking(
		r.Context(),
		req,
		idempotencyKey,
	)



	if err != nil {
		status := http.StatusConflict
		code := "BOOKING_FAILED"
		message := err.Error()

		switch {
		case errors.Is(err, booking.ErrListingNotFound):
			status = http.StatusNotFound
			code = "LISTING_NOT_FOUND"
		case errors.Is(err, booking.ErrInvalidDates):
			status = http.StatusBadRequest
			code = "INVALID_DATES"
		case errors.Is(err, booking.ErrLockUnavailable):
			status = http.StatusConflict
			code = "DATES_ALREADY_RESERVED"
		case errors.Is(err, booking.ErrPaymentAuthorizationFailed):
			status = http.StatusPaymentRequired
			code = "PAYMENT_AUTHORIZATION_FAILED"
		case errors.Is(err, booking.ErrPaymentCaptureFailed):
			status = http.StatusBadGateway
			code = "PAYMENT_CAPTURE_FAILED"
		case errors.Is(err, booking.ErrBookingCreationFailed):
			status = http.StatusInternalServerError
			code = "BOOKING_CREATION_FAILED"
		case errors.Is(err, booking.ErrOutboxCreationFailed):
			status = http.StatusInternalServerError
			code = "OUTBOX_CREATION_FAILED"
		case errors.Is(err, booking.ErrDatabase):
			status = http.StatusInternalServerError
			code = "DATABASE_ERROR"
		case errors.Is(err, booking.ErrLockConfirmationFailed):
			status = http.StatusInternalServerError
			code = "LOCK_CONFIRMATION_FAILED"
		}

		Error(
			w,
			status,
			code,
			message,
		)

		return
	}



	JSON(
		w,
		http.StatusOK,
		result,
	)
}