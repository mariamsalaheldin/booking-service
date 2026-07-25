package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string

	RedisAddr string

	RabbitMQURL string

	HTTPPort string

	LockTTLSeconds int

	BookingTTLSeconds int
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		DatabaseURL: getEnv(
			"DATABASE_URL",
			"postgres://postgres:postgres@localhost:5432/booking_db?sslmode=disable",
		),

		RedisAddr: getEnv(
			"REDIS_ADDR",
			"localhost:6379",
		),

		RabbitMQURL: getEnv(
			"RABBITMQ_URL",
			"amqp://guest:guest@localhost:5672/",
		),

		HTTPPort: getEnv(
			"HTTP_PORT",
			"8080",
		),

		LockTTLSeconds: getEnvInt(
			"LOCK_TTL_SECONDS",
			30,
		),

		BookingTTLSeconds: getEnvInt(
			"BOOKING_TTL_SECONDS",
			60*60*24*30,
		),
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("database URL is required")
	}
	if strings.TrimSpace(c.RedisAddr) == "" {
		return fmt.Errorf("redis address is required")
	}
	if strings.TrimSpace(c.RabbitMQURL) == "" {
		return fmt.Errorf("rabbitmq URL is required")
	}
	if strings.TrimSpace(c.HTTPPort) == "" {
		return fmt.Errorf("http port is required")
	}
	if _, err := strconv.Atoi(c.HTTPPort); err != nil {
		return fmt.Errorf("http port must be numeric: %w", err)
	}
	if c.LockTTLSeconds <= 0 {
		return fmt.Errorf("lock TTL must be greater than zero")
	}
	if c.BookingTTLSeconds <= 0 {
		return fmt.Errorf("booking TTL must be greater than zero")
	}
	return nil
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return n
}
