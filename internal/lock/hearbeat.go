package lock

import (
	"context"
	"time"
)

type Heartbeat struct {
	manager *Manager
	lock    *Lock
	ttl     time.Duration

	stop chan struct{}
	done chan struct{}
}

func NewHeartbeat(
	manager *Manager,
	lock *Lock,
	ttl time.Duration,
) *Heartbeat {
	return &Heartbeat{
		manager: manager,
		lock:    lock,
		ttl:     ttl,

		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (h *Heartbeat) Start(ctx context.Context) {
	go func() {
		defer close(h.done)

		ticker := time.NewTicker(
			h.ttl / 3,
		)

		defer ticker.Stop()

		for {
			select {

			case <-ctx.Done():
				return

			case <-h.stop:
				return

			case <-ticker.C:

				_ = h.manager.Refresh(
					ctx,
					h.lock,
					h.ttl,
				)
			}
		}
	}()
}

func (h *Heartbeat) Stop() {
	close(h.stop)

	<-h.done
}
