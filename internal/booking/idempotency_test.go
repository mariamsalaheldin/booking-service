package booking

import (
	"testing"
)

func TestNewPostgresIdempotencyStore(t *testing.T) {
	if _, err := NewPostgresIdempotencyStore(nil); err == nil {
		t.Fatal("expected constructor to reject a nil database handle")
	}
}
