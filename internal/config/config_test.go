package config

import (
	"os"
	"testing"
)

func TestLoadUsesDefaultValues(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "REDIS_ADDR", "RABBITMQ_URL", "HTTP_PORT", "LOCK_TTL_SECONDS", "BOOKING_TTL_SECONDS"} {
		_ = os.Unsetenv(key)
	}

	cfg := Load()

	if cfg.DatabaseURL == "" {
		t.Fatal("expected default database URL")
	}
	if cfg.RedisAddr == "" {
		t.Fatal("expected default redis address")
	}
	if cfg.RabbitMQURL == "" {
		t.Fatal("expected default rabbitmq url")
	}
	if cfg.HTTPPort == "" {
		t.Fatal("expected default http port")
	}
}
