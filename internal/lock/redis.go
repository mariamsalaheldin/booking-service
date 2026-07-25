package lock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Manager struct {
	client *redis.Client

	acquire *redis.Script
	release *redis.Script
	confirm *redis.Script
	refresh *redis.Script
}

func NewManager(addr string) *Manager {
	client := redis.NewClient(
		&redis.Options{
			Addr: addr,
		},
	)

	return &Manager{
		client: client,

		acquire: redis.NewScript(acquireScript),
		release: redis.NewScript(releaseScript),
		confirm: redis.NewScript(confirmScript),
		refresh: redis.NewScript(refreshScript),
	}
}

type Lock struct {
	Token string

	Keys []string
}

func (m *Manager) Acquire(
	ctx context.Context,
	keys []string,
	ttl time.Duration,
) (*Lock, bool, error) {
	token := uuid.NewString()

	result, err := m.acquire.Run(
		ctx,
		m.client,
		keys,
		int(ttl.Seconds()),
		token,
	).Int()
	if err != nil {
		return nil, false, err
	}

	if result == 0 {
		return nil, false, nil
	}

	return &Lock{
		Token: token,
		Keys:  keys,
	}, true, nil
}

func (m *Manager) Release(
	ctx context.Context,
	lock *Lock,
) error {
	_, err := m.release.Run(
		ctx,
		m.client,
		lock.Keys,
		lock.Token,
	).Int()

	return err
}

func (m *Manager) Confirm(
	ctx context.Context,
	lock *Lock,
	ttl time.Duration,
) error {
	_, err := m.confirm.Run(
		ctx,
		m.client,
		lock.Keys,
		lock.Token,
		int(ttl.Seconds()),
	).Int()

	return err
}

func (m *Manager) Refresh(
	ctx context.Context,
	lock *Lock,
	ttl time.Duration,
) error {
	_, err := m.refresh.Run(
		ctx,
		m.client,
		lock.Keys,
		lock.Token,
		int(ttl.Seconds()),
	).Int()

	return err
}

func (m *Manager) Close() error {
	return m.client.Close()
}
