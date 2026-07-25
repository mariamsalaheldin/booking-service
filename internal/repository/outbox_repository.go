package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/mariamsalaheldin/booking-service/internal/domain"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{
		db: db,
	}
}

func (r *OutboxRepository) Create(
	ctx context.Context,
	tx *sql.Tx,
	event domain.OutboxEvent,
) error {
	_, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO outbox_events
		(
			id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload
		)
		VALUES
		($1,$2,$3,$4,$5)
		`,
		event.ID,
		event.AggregateType,
		event.AggregateID,
		event.EventType,
		event.Payload,
	)

	return err
}

func (r *OutboxRepository) GetUnprocessed(
	ctx context.Context,
	tx *sql.Tx,
	limit int,
) ([]domain.OutboxEvent, error) {
	rows, err := tx.QueryContext(
		ctx,
		`
		SELECT
			id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			processed_at,
			created_at
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
		`,
		limit,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var events []domain.OutboxEvent

	for rows.Next() {

		var event domain.OutboxEvent

		err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.ProcessedAt,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

func (r *OutboxRepository) MarkProcessed(
	ctx context.Context,
	tx *sql.Tx,
	id uuid.UUID,
) error {
	_, err := tx.ExecContext(
		ctx,
		`
		UPDATE outbox_events
		SET processed_at = NOW()
		WHERE id=$1
		`,
		id,
	)

	return err
}
