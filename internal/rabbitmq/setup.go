package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const ExchangeName = "booking.events"

var Queues = []string{
	"q_db_consumer",
	"q_notification_consumer",
	"q_smart_lock_consumer",
}

func Setup(
	conn *amqp.Connection,
) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	defer ch.Close()

	err = ch.ExchangeDeclare(
		ExchangeName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for _, queue := range Queues {

		_, err = ch.QueueDeclare(
			queue,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return err
		}

		err = ch.QueueBind(
			queue,
			"",
			ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return err
		}
	}

	log.Println("RabbitMQ setup complete")

	return nil
}
