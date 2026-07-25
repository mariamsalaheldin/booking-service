.PHONY: dev dev-stop dev-reset logs test build help

help:
	@echo "Available commands:"
	@echo "  make dev         - Start infrastructure, run migrations, and start the server"
	@echo "  make dev-stop    - Stop infrastructure"
	@echo "  make dev-reset   - Reset database, Redis, and RabbitMQ queues"
	@echo "  make logs        - View infrastructure logs"
	@echo "  make test        - Run tests"
	@echo "  make build       - Build the server binary"
	@echo "  make help        - Show this help message"

dev:
	@echo "Starting infrastructure..."
	docker compose up -d
	@echo "Waiting for services to be ready..."
	@powershell -Command "Start-Sleep -Seconds 10"
	@echo "Installing dependencies..."
	go mod tidy
	@echo "Starting server (migrations will run automatically)..."
	go run ./cmd/server

dev-stop:
	@echo "Stopping infrastructure..."
	docker compose down

dev-reset:
	@echo "Resetting database state..."
	docker exec -i booking-postgres psql -U postgres -d booking_db -c "TRUNCATE TABLE bookings, outbox_events, idempotency_keys, listings RESTART IDENTITY CASCADE;"
	docker exec -i booking-postgres psql -U postgres -d booking_db < migrations/004_listings.sql
	@echo "Flushing Redis..."
	docker exec booking-redis redis-cli FLUSHALL
	@echo "Purging RabbitMQ queues..."
	docker exec booking-rabbitmq rabbitmqctl purge_queue q_db_consumer || true
	docker exec booking-rabbitmq rabbitmqctl purge_queue q_notification_consumer || true
	docker exec booking-rabbitmq rabbitmqctl purge_queue q_smart_lock_consumer || true
	@echo "Reset complete"

logs:
	docker compose logs -f

test:
	go test ./...

build:
	go build -o bin/server ./cmd/server
