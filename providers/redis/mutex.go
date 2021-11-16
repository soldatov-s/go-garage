package redis

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/soldatov-s/go-garage/providers/redis/rejson"
)

const (
	defaultLockID        = "8d0d7b2641d711ea8f655f8773827ca0"
	defaultCheckInterval = 100 * time.Millisecond
	defaultExpire        = 15 * time.Second
)

// Mutex provides a distributed mutex across multiple instances via Redis
type Mutex struct {
	conn          *rejson.Client
	lockKey       string
	lockValue     string
	checkInterval time.Duration
	expire        time.Duration
	locked        bool
	// RWMutex to lock inside an instance
	mu sync.RWMutex
}

// NewMutex creates new distributed redis mutex
func NewMutex(conn *rejson.Client, expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutexByID(conn, defaultLockID, expire, checkInterval)
}

// NewMutexByID creates new distributed redis mutex by ID
func NewMutexByID(conn *rejson.Client, lockKey string, expire, checkInterval time.Duration) (*Mutex, error) {
	checkIntervalValue := checkInterval
	if checkIntervalValue == 0 {
		checkIntervalValue = defaultCheckInterval
	}

	expireValue := expire
	if expireValue == 0 {
		expireValue = defaultExpire
	}

	return &Mutex{
		conn:          conn,
		lockKey:       lockKey,
		checkInterval: checkIntervalValue,
		expire:        expireValue,
		locked:        false,
	}, nil
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

	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	m.lockValue = newUUID.String()

	result, err = m.conn.SetNX(ctx, m.lockKey, m.lockValue, m.expire).Result()
	if err != nil {
		return err
	}

	if !result {
		m.locked = true

		for {
			result, err = m.conn.SetNX(ctx, m.lockKey, m.lockValue, m.expire).Result()
			if err != nil {
				return err
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
			return err
		}

		if m.lockValue == cmdString {
			_, err = m.conn.Del(m.conn.Context(), m.lockKey).Result()
			if err != nil {
				return err
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
			return err
		}

		if m.lockValue == cmdString {
			_, err = m.conn.Expire(ctx, m.lockKey, timeout).Result()
			if err != nil {
				return err
			}

			m.locked = false
		}
	}

	return nil
}
