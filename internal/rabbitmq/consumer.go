package rabbitmq

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Handler func(
	ctx context.Context,
	body []byte,
) error

type Consumer struct {
	conn *amqp.Connection
}

func NewConsumer(
	conn *amqp.Connection,
) *Consumer {
	return &Consumer{
		conn: conn,
	}
}

func (c *Consumer) Consume(
	ctx context.Context,
	queue string,
	handler Handler,
) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	defer ch.Close()

	err = ch.Qos(
		10,
		0,
		false,
	)
	if err != nil {
		return err
	}

	messages, err := ch.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {

		case <-ctx.Done():

			return nil

		case msg, ok := <-messages:

			if !ok {
				return nil
			}

			err := handler(
				ctx,
				msg.Body,
			)
			if err != nil {

				log.Println(
					"message failed:",
					err,
				)

				_ = msg.Nack(
					false,
					true,
				)

				continue
			}

			_ = msg.Ack(false)
		}
	}
}
