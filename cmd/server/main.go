package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/mariamsalaheldin/booking-service/internal/api"
	"github.com/mariamsalaheldin/booking-service/internal/booking"
	"github.com/mariamsalaheldin/booking-service/internal/config"
	"github.com/mariamsalaheldin/booking-service/internal/lock"
	"github.com/mariamsalaheldin/booking-service/internal/migrate"
	"github.com/mariamsalaheldin/booking-service/internal/outbox"
	"github.com/mariamsalaheldin/booking-service/internal/rabbitmq"
	"github.com/mariamsalaheldin/booking-service/internal/repository"
)

func main() {
	cfg := config.Load()

	logger := log.Default()

	if err := cfg.Validate(); err != nil {
		logger.Fatalf("invalid config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	server, cleanup, err := buildHTTPServer(
		cfg,
		logger,
		ctx,
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer cleanup()

	go func() {
		logger.Println("HTTP server running on", cfg.HTTPPort)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	_ = server.Shutdown(shutdownCtx)
	logger.Println("server stopped")
}

func buildHTTPServer(
	cfg config.Config,
	logger *log.Logger,
	ctx context.Context,
) (*http.Server, func(), error) {
	db, err := repository.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}

	if err := migrate.Run(db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("migration failed: %w", err)
	}

	lockManager := lock.NewManager(cfg.RedisAddr)

	rmqConn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		_ = db.Close()
		lockManager.Close()
		return nil, nil, err
	}

	if err := rabbitmq.Setup(rmqConn); err != nil {
		_ = rmqConn.Close()
		_ = db.Close()
		lockManager.Close()
		return nil, nil, err
	}

	bookingRepo := repository.NewBookingRepository(db.DB)
	listingRepo := repository.NewListingRepository(db.DB)
	outboxRepo := repository.NewOutboxRepository(db.DB)
	payment := booking.NewMockPaymentService()

	bookingService, err := booking.NewService(
		db,
		bookingRepo,
		outboxRepo,
		lockManager,
		payment,
		listingRepo,
	)
	if err != nil {
		_ = rmqConn.Close()
		_ = db.Close()
		lockManager.Close()
		return nil, nil, err
	}

	httpHandler := api.NewHandler(bookingService)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/bookings", httpHandler.CreateBooking)

	rabbitPublisher := rabbitmq.NewPublisher(rmqConn)
	outboxPublisher := outbox.NewPublisher(db, outboxRepo, rabbitPublisher)
	go outboxPublisher.Start(ctx)

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	cleanup := func() {
		_ = db.Close()
		lockManager.Close()
		_ = rmqConn.Close()
	}

	return server, cleanup, nil
}