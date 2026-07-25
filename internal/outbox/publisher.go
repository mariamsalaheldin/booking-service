package outbox

import (
	"context"
	"time"

	"github.com/mariamsalaheldin/booking-service/internal/rabbitmq"
	"github.com/mariamsalaheldin/booking-service/internal/repository"
)

type Publisher struct {
	db        *repository.Postgres
	outbox    *repository.OutboxRepository
	publisher *rabbitmq.Publisher
}

func NewPublisher(
	db *repository.Postgres,
	outbox *repository.OutboxRepository,
	publisher *rabbitmq.Publisher,
) *Publisher {
	return &Publisher{
		db:        db,
		outbox:    outbox,
		publisher: publisher,
	}
}

func (p *Publisher) Start(
	ctx context.Context,
) {
	ticker := time.NewTicker(
		500 * time.Millisecond,
	)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():
			return

		case <-ticker.C:

			_ = p.processBatch(ctx)
		}
	}
}

func (p *Publisher) processBatch(
	ctx context.Context,
) error {
	tx, err := p.db.DB.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	events, err := p.outbox.GetUnprocessed(
		ctx,
		tx,
		50,
	)
	if err != nil {
		return err
	}

	for _, event := range events {

		err = p.publisher.Publish(
			ctx,
			event.Payload,
		)
		if err != nil {
			return err
		}

		err = p.outbox.MarkProcessed(
			ctx,
			tx,
			event.ID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
