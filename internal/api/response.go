package api

import (
	"encoding/json"
	"net/http"

	"github.com/mariamsalaheldin/booking-service/internal/booking"
)

func JSON(
	w http.ResponseWriter,
	status int,
	data any,
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}



func Error(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {

	JSON(
		w,
		status,
		booking.ErrorResponse{
			Error: booking.APIError{
				Code: code,
				Message: message,
			},
		},
	)
}