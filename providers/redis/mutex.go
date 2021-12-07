package redis

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	defaultLockID        = "8d0d7b2641d711ea8f655f8773827ca0"
	defaultCheckInterval = 100 * time.Millisecond
	defaultExpire        = 15 * time.Second
)

type MutexConnector interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// Mutex provides a distributed mutex across multiple instances via Redis
type Mutex struct {
	conn          MutexConnector
	lockKey       string
	lockValue     string
	checkInterval time.Duration
	expire        time.Duration
	locked        bool
	// RWMutex to lock inside an instance
	mu sync.RWMutex
}

type MutexOption func(*Mutex)

func WithCheckInterval(checkInterval time.Duration) MutexOption {
	return func(c *Mutex) {
		c.checkInterval = checkInterval
	}
}

func WithExpire(expire time.Duration) MutexOption {
	return func(c *Mutex) {
		c.expire = expire
	}
}

func WithLockKey(lockKey string) MutexOption {
	return func(c *Mutex) {
		c.lockKey = lockKey
	}
}

// NewMutex creates new distributed redis mutex
func NewMutex(conn MutexConnector, opts ...MutexOption) (*Mutex, error) {
	mu := &Mutex{
		conn:          conn,
		lockKey:       defaultLockID,
		checkInterval: defaultCheckInterval,
		expire:        defaultExpire,
		locked:        false,
	}

	newUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, errors.Wrap(err, "generate lock value")
	}

	mu.lockValue = newUUID.String()

	for _, opt := range opts {
		opt(mu)
	}

	return mu, nil
}

// Lock sets Redis-lock item. It is blocking call which will wait until
// redis lock key will be deleted, pretty much like simple mutex.
func (m *Mutex) Lock(ctx context.Context) (err error) {
	m.mu.Lock()
	return m.commonLock(ctx)
}

// Unlock deletes Redis-lock item.
func (m *Mutex) Unlock(ctx context.Context) (err error) {
	m.mu.Unlock()
	return m.commonUnlock(ctx)
}

// Extend attempts to extend the timeout of a Redis-lock.
func (m *Mutex) Extend(ctx context.Context, timeout time.Duration) (err error) {
	return m.commonExtend(ctx, timeout)
}

func (m *Mutex) commonLock(ctx context.Context) (err error) {
	var result bool

	result, err = m.conn.SetNX(ctx, m.lockKey, m.lockValue, m.expire).Result()
	if err != nil {
		return errors.Wrap(err, "set nx lock")
	}

	if !result {
		m.locked = true

		for {
			result, err = m.conn.SetNX(ctx, m.lockKey, m.lockValue, m.expire).Result()
			if err != nil {
				return errors.Wrap(err, "set nx lock")
			}

			if result || !m.locked {
				return nil
			}

			time.Sleep(m.checkInterval)
		}
	}

	return nil
}

func (m *Mutex) commonUnlock(ctx context.Context) (err error) {
	if m.locked {
		cmdString, err := m.conn.Get(ctx, m.lockKey).Result()
		if err != nil {
			return errors.Wrap(err, "get lock from redis")
		}

		if m.lockValue == cmdString {
			_, err = m.conn.Del(ctx, m.lockKey).Result()
			if err != nil {
				return errors.Wrap(err, "del lock from redis")
			}

			m.locked = false
		}
	}

	return nil
}

func (m *Mutex) commonExtend(ctx context.Context, timeout time.Duration) (err error) {
	if m.locked {
		cmdString, err := m.conn.Get(ctx, m.lockKey).Result()
		if err != nil {
			return errors.Wrap(err, "get lock from redis")
		}

		if m.lockValue == cmdString {
			_, err = m.conn.Expire(ctx, m.lockKey, timeout).Result()
			if err != nil {
				return errors.Wrap(err, "expire lock in redis")
			}

			m.locked = false
		}
	}

	return nil
}
