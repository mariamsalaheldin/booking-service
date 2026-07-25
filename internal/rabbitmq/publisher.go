package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn *amqp.Connection
}

func NewPublisher(
	conn *amqp.Connection,
) *Publisher {
	return &Publisher{
		conn: conn,
	}
}

func (p *Publisher) Publish(
	ctx context.Context,
	payload []byte,
) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}

	defer ch.Close()

	err = ch.Confirm(false)
	if err != nil {
		return err
	}

	confirms := ch.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	err = ch.PublishWithContext(
		ctx,
		ExchangeName,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",

			DeliveryMode: amqp.Persistent,

			Body: payload,
		},
	)
	if err != nil {
		return err
	}

	select {

	case confirmation := <-confirms:

		if !confirmation.Ack {
			return errPublisherRejected
		}

		return nil

	case <-ctx.Done():

		return ctx.Err()
	}
}

var errPublisherRejected = &PublisherError{
	message: "rabbitmq rejected message",
}

type PublisherError struct {
	message string
}

func (e *PublisherError) Error() string {
	return e.message
}
