package booking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/mariamsalaheldin/booking-service/internal/repository"
)

type IdempotencyStore interface {
	Get(
		ctx context.Context,
		key string,
	) (*BookingResponse, bool)

	Set(
		ctx context.Context,
		key string,
		response BookingResponse,
		ttl time.Duration,
	) error
}

type PostgresIdempotencyStore struct {
	db *sql.DB
}

func NewPostgresIdempotencyStore(
	db *repository.Postgres,
) (*PostgresIdempotencyStore, error) {
	if db == nil || db.DB == nil {
		return nil, errors.New("nil postgres db")
	}

	return &PostgresIdempotencyStore{
		db: db.DB,
	}, nil
}

func (s *PostgresIdempotencyStore) Get(
	ctx context.Context,
	key string,
) (*BookingResponse, bool) {
	var payload []byte

	err := s.db.QueryRowContext(
		ctx,
		`
		SELECT response
		FROM idempotency_keys
		WHERE key=$1
		AND expires_at > NOW()
		`,
		key,
	).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false
		}

		return nil, false
	}

	var response BookingResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, false
	}

	return &response, true
}

func (s *PostgresIdempotencyStore) Set(
	ctx context.Context,
	key string,
	response BookingResponse,
	ttl time.Duration,
) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(ttl).UTC()

	_, err = s.db.ExecContext(
		ctx,
		`
		INSERT INTO idempotency_keys
		(
			key,
			response,
			status_code,
			expires_at
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key)
		DO UPDATE SET
			response = EXCLUDED.response,
			status_code = EXCLUDED.status_code,
			expires_at = EXCLUDED.expires_at
		`,
		key,
		payload,
		http.StatusOK,
		expiresAt,
	)
	if err != nil {
		return err
	}

	return nil
}