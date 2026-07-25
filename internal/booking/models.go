package booking

import (
	"time"

	"github.com/google/uuid"

	"github.com/mariamsalaheldin/booking-service/internal/domain"
)

// Shared schemas are defined in the domain package to avoid
// package import cycles between booking and repository.

type Booking = domain.Booking

type BookingRequest = domain.BookingRequest

type BookingResponse = domain.BookingResponse

type EventPayload = domain.EventPayload

type OutboxEvent = domain.OutboxEvent

type PaymentAuthorization = domain.PaymentAuthorization

type APIError = domain.APIError

type ErrorResponse = domain.ErrorResponse

func NewBooking(
	listingID uuid.UUID,
	userID uuid.UUID,
	checkIn time.Time,
	checkOut time.Time,
) Booking {
	return Booking{
		ListingID: listingID,
		UserID:    userID,
		CheckIn:   checkIn,
		CheckOut:  checkOut,
		Status:    "CONFIRMED",
	}
}