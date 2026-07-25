package booking

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type PaymentService interface {
	Authorize(
		paymentMethodID string,
		amount float64,
	) (*PaymentAuthorization, error)

	Capture(
		authorizationID string,
	) error

	Void(
		authorizationID string,
	) error
}


type MockPaymentService struct{}


func NewMockPaymentService() *MockPaymentService {
	return &MockPaymentService{}
}


func (p *MockPaymentService) Authorize(
	paymentMethodID string,
	amount float64,
) (*PaymentAuthorization, error) {

	if paymentMethodID == "fail_me" {
		return nil, errors.New("payment authorization failed")
	}


	return &PaymentAuthorization{
		AuthorizationID: uuid.NewString(),
		PaymentMethodID: paymentMethodID,
		Amount:          amount,
		Currency:        "USD",
		Status:          "AUTHORIZED",
		AuthorizedAt:    time.Now().UTC(),
	}, nil
}



func (p *MockPaymentService) Capture(
	authorizationID string,
) error {

	if authorizationID == "" {
		return errors.New("missing authorization id")
	}

	return nil
}



func (p *MockPaymentService) Void(
	authorizationID string,
) error {

	if authorizationID == "" {
		return errors.New("missing authorization id")
	}

	return nil
}