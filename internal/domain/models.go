package domain

import (
	"time"

	"github.com/google/uuid"
)

type Booking struct {
	ID        uuid.UUID `json:"id"`
	ListingID uuid.UUID `json:"listing_id"`
	UserID    uuid.UUID `json:"user_id"`

	CheckIn  time.Time `json:"check_in"`
	CheckOut time.Time `json:"check_out"`

	Status string `json:"status"`

	CreatedAt time.Time `json:"created_at"`
}

type BookingRequest struct {
	ListingID uuid.UUID `json:"listing_id"`
	UserID    uuid.UUID `json:"user_id"`

	CheckIn  string `json:"check_in"`
	CheckOut string `json:"check_out"`

	PaymentMethodID string `json:"payment_method_id"`
}

type BookingResponse struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
}

type EventPayload struct {
	BookingID string `json:"booking_id"`
	ListingID string `json:"listing_id"`
	UserID    string `json:"user_id"`

	CheckIn  string `json:"check_in"`
	CheckOut string `json:"check_out"`

	Timestamp time.Time `json:"timestamp"`
}

type OutboxEvent struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte

	ProcessedAt *time.Time
	CreatedAt   time.Time
}

type PaymentAuthorization struct {
	AuthorizationID string
	PaymentMethodID string
	Amount          float64
	Currency        string
	Status          string
	AuthorizedAt    time.Time
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}
