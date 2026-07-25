# Booking Service

Production-style concurrent booking service written in Go.

## Features

- PostgreSQL transactional booking storage
- PostgreSQL exclusion constraint to prevent double booking
- Redis distributed locks
- Redis lock ownership tokens
- Redis lock heartbeat renewal
- Idempotent booking requests
- Transactional outbox pattern
- RabbitMQ event publishing
- RabbitMQ publisher confirms
- RabbitMQ fanout exchange and downstream consumer queues
- Payment authorization/capture flow (mock)
- Docker Compose development environment

## Project Structure

```
booking-service/
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── api/
│   │   ├── handler.go
│   │   ├── handler_integration.go
│   │   └── response.go
│   │
│   ├── booking/
│   │   ├── idempotency.go
│   │   ├── idempotency_test.go
│   │   ├── models.go
│   │   ├── payment.go
│   │   ├── service.go
│   │   ├── service_e2e_test.go
│   │   └── service_test.go
│   │
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   │
│   ├── domain/
│   │   └── models.go
│   │
│   ├── lock/
│   │   ├── heartbeat.go
│   │   ├── redis.go
│   │   └── scripts.go
│   │
│   ├── logger/
│   │   └── logger.go
│   │
│   ├── outbox/
│   │   └── publisher.go
│   │
│   ├── rabbitmq/
│   │   ├── consumer.go
│   │   ├── publisher.go
│   │   ├── setup.go
│   │   └── workerpool.go
│   │
│   └── repository/
│       ├── booking_repository.go
│       ├── outbox_repository.go
│       └── postgres.go
│
├── migrations/
│   ├── 001_bookings.sql
│   ├── 002_outbox.sql
│   ├── 003_idempotency.sql
│   └── 004_listings.sql
│
├── test/
│   └── concurrency/
│       ├── concurrency_test.go
│       └── concurrencymultiple_test.go
│
├── .env
├── .gitignore
├── docker-compose.yml
├── dockerfile
├── go.mod
├── go.sum
└── README.md
```



## Architecture

```
Client
  ↓
HTTP API
  ↓
Booking Service
  ↓
Redis Lock (per listing/date)
  ↓
Payment Authorization
  ↓
PostgreSQL Transaction
  ├── INSERT booking
  ├── INSERT outbox event
  └── Payment Capture
  ↓
COMMIT
  ↓
Confirm Redis Lock (long TTL)
  ↓
Outbox Worker → RabbitMQ fanout exchange
  ↓
Downstream queues (q_db_consumer, q_notification_consumer, q_smart_lock_consumer)
```

The server declares RabbitMQ exchange/queues and publishes events via the outbox worker. Consumer handlers are implemented in `internal/rabbitmq/consumer.go` but are not started by `cmd/server` — downstream services would consume from the declared queues.

## Requirements

- Go 1.25+
- Docker
- Docker Compose

## Quick Start

Start everything with a single command:

```bash
make dev
```

This will:
- Start Docker Compose infrastructure (PostgreSQL, Redis, RabbitMQ)
- Run database migrations automatically
- Install Go dependencies
- Start the server

Server: `http://localhost:8080`

### Other Useful Commands

```bash
make dev-stop    # Stop infrastructure
make dev-reset   # Reset database, Redis, and RabbitMQ queues
make logs        # View infrastructure logs
make test        # Run tests
make build       # Build the server binary
make help        # Show all available commands
```

## Infrastructure

| Service             | Port  |
| ------------------- | ----- |
| PostgreSQL          | 5432  |
| Redis               | 6379  |
| RabbitMQ            | 5672  |
| RabbitMQ Management | 15672 |

## Environment Variables

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/booking_db?sslmode=disable
REDIS_ADDR=localhost:6379
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
HTTP_PORT=8080

# Optional (defaults shown)
LOCK_TTL_SECONDS=30
BOOKING_TTL_SECONDS=2592000
```

## Manual Setup (Optional)

If you prefer manual setup instead of using `make dev`:

```bash
# Start infrastructure
docker compose up -d

# Run migrations manually
psql postgres://postgres:postgres@localhost:5432/booking_db < migrations/001_bookings.sql
psql postgres://postgres:postgres@localhost:5432/booking_db < migrations/002_outbox.sql
psql postgres://postgres:postgres@localhost:5432/booking_db < migrations/003_idempotency.sql
psql postgres://postgres:postgres@localhost:5432/booking_db < migrations/004_listings.sql

# Install dependencies and run
go mod tidy
go run ./cmd/server
```

## Create Booking

```http
POST /api/v1/bookings
Content-Type: application/json
Idempotency-Key: test-key-1
```

Body:

```json
{
  "listing_id": "11111111-1111-4111-8111-111111111111",
  "user_id": "44444444-4444-4444-8444-444444444444",
  "check_in": "2026-08-01",
  "check_out": "2026-08-05",
  "payment_method_id": "card_test"
}
```

Response (`200 OK`):

```json
{
  "booking_id": "generated-uuid",
  "status": "CONFIRMED"
}
```

The `Idempotency-Key` header is required. Repeating the same key within 24 hours returns the original response without creating a duplicate booking.

The `listing_id` must exist in the `listings` table (seeded by migration `004`).

## Concurrency Protection

### Redis Lock

Prevents multiple users from reserving the same listing dates simultaneously.

```
User A → lock Aug 1–4 → succeeds
User B → lock Aug 1–4 → rejected (409 DATES_ALREADY_RESERVED)
```

### PostgreSQL Constraint

Final source of truth. Even if Redis fails, overlapping bookings are blocked:

```sql
EXCLUDE USING gist
(
  listing_id WITH =,
  booking_period WITH &&
)
```

### Outbox Pattern

Within one database transaction:

```
BEGIN
  INSERT booking
  INSERT outbox event
  Payment capture
COMMIT
```

Then asynchronously:

```
Outbox Worker → RabbitMQ → downstream consumer queues
```

## RabbitMQ Queues

The server declares a fanout exchange (`booking.events`) bound to:

- `q_db_consumer`
- `q_notification_consumer`
- `q_smart_lock_consumer`

Each queue receives booking events independently. Wire up consumers in separate services or extend `cmd/server/main.go` to start them locally.

## Development

Format:

```bash
gofmt -w .
```

Test:

```bash
go test ./...
```

Build:

```bash
go build ./cmd/server
```

## Local Run and Manual Verification

Quick start:

```bash
make dev
```

Example booking request:

```bash
curl -X POST http://localhost:8080/api/v1/bookings \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-key-1" \
  -d '{
    "listing_id": "11111111-1111-4111-8111-111111111111",
    "user_id": "44444444-4444-4444-8444-444444444444",
    "check_in": "2026-08-01",
    "check_out": "2026-08-03",
    "payment_method_id": "card_test"
  }'
```

Reset local state for a clean test run:

```bash
docker exec -i booking-postgres psql -U postgres -d booking_db -c "TRUNCATE TABLE bookings, outbox_events, idempotency_keys, listings RESTART IDENTITY CASCADE;"
docker exec -i booking-postgres psql -U postgres -d booking_db < migrations/004_listings.sql
docker exec booking-redis redis-cli FLUSHALL
```

